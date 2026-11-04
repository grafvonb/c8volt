// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

// formatRunProcessInstanceFailure counts only returned successes. Unreturned work may
// have failed, remained unattempted, or been submitted without successful observation.
func formatRunProcessInstanceFailure(cmd *cobra.Command, requested, returned int, err error) string {
	reason := boundedCommandFailureReason(err)
	if cmd.Flags().Changed("set-bid") && ferrors.Classify(err) == ferrors.ClassUnsupported {
		reason = "business ID creation requires Camunda 8.10"
	}
	return fmt.Sprintf("creating process instances: requested %d; successful %d; without successful result %d; %s", requested, returned, requested-returned, reason)
}

// renderRunProcessInstanceResult keeps JSON on the shared mutation envelope while rendering human output as a compact creation result.
func renderRunProcessInstanceResult(cmd *cobra.Command, result process.ProcessInstances, runErr error) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		if runErr != nil {
			envelope := resultEnvelopeForError(cmd, runErr)
			envelope.Payload = result
			return renderResultEnvelope(cmd, envelope)
		}
		return renderCommandResult(cmd, result)
	}
	mode := pickMode()
	if mode == RenderModeKeysOnly {
		for _, item := range result.Items {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), item.Key); err != nil {
				return err
			}
		}
		return nil
	}
	rows := make([]flatRow, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, flatRowRunProcessInstanceWithTimezone(item, commandShowTimezoneOffset(cmd)))
	}
	for _, line := range formatFlatRows(rows) {
		renderHumanLine(cmd, "%s", line)
	}
	renderHumanLine(cmd, "found: %d", len(result.Items))
	return nil
}

// flatRowRunProcessInstanceWithTimezone uses only returned facts and quotes opaque identities safely.
func flatRowRunProcessInstanceWithTimezone(item process.ProcessInstance, showTimezoneOffset bool) flatRow {
	parent := "p:<root>"
	if item.ParentKey != "" {
		parent = "p:" + item.ParentKey
	}
	version := fmt.Sprintf("v%d", item.ProcessVersion)
	if item.ProcessVersionTag != "" {
		version += "/" + item.ProcessVersionTag
	}
	row := flatRow{
		item.Key,
		item.TenantId,
		item.BpmnProcessId,
		processInstanceBusinessIDColumn(item.BusinessId),
		version,
	}
	if item.State != "" {
		row = append(row, string(item.State))
	}
	row = append(row, "s:"+toolx.FormatTimestamp(item.StartDate, showTimezoneOffset), parent)
	return row
}

// sortRunProcessInstancesForOutput preserves chronological counted output with deterministic key ties.
func sortRunProcessInstancesForOutput(items []process.ProcessInstance) {
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.StartDate != right.StartDate {
			if left.StartDate == "" {
				return false
			}
			if right.StartDate == "" {
				return true
			}
			return left.StartDate < right.StartDate
		}
		return runProcessInstanceKeyLess(left.Key, right.Key)
	})
}

// runProcessInstanceKeyLess sorts numeric keys numerically and other identities lexically.
func runProcessInstanceKeyLess(left, right string) bool {
	li, lerr := strconv.ParseUint(left, 10, 64)
	ri, rerr := strconv.ParseUint(right, 10, 64)
	if lerr == nil && rerr == nil {
		return li < ri
	}
	return left < right
}
