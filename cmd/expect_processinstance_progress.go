// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"time"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// expectProcessInstanceCompletionPhase is the shared completion scope emitted
// by multi-key process-instance expectation waits.
const expectProcessInstanceCompletionPhase = "expect process instances"

// expectProcessInstanceSemanticProgressNow is overridden by command tests to
// exercise durable milestone pacing without real sleeps.
var expectProcessInstanceSemanticProgressNow = time.Now

// newExpectProcessInstanceSemanticProgress returns a reporter only for multi-key
// expectation scopes so single-target polling keeps its existing wait activity.
func newExpectProcessInstanceSemanticProgress(cmd *cobra.Command, total int) *opsSemanticProgressReporter {
	if total <= 1 {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	return newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			Phase:         expectProcessInstanceCompletionPhase,
			ActivityLabel: "waiting for process-instance expectations",
			CoreResource:  "process instance(s)",
			Total:         total,
			ConfirmedVerb: "satisfied",
			FailedVerb:    "failed",
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(channel),
		Now:    expectProcessInstanceSemanticProgressNow,
	})
}

// appendExpectProcessInstanceProgressOption routes expectation completion facts
// from the facade callback into the command-owned semantic reporter.
func appendExpectProcessInstanceProgressOption(opts []processOptions.FacadeOption, reporter *opsSemanticProgressReporter) []processOptions.FacadeOption {
	if reporter == nil {
		return opts
	}
	out := append([]processOptions.FacadeOption{}, opts...)
	return append(out, processOptions.WithProgress(func(event processOptions.ProgressEvent) {
		if event.Kind != processOptions.ProgressEventKindCompletion || event.Completion == nil {
			return
		}
		if strings.TrimSpace(event.Completion.Phase) != expectProcessInstanceCompletionPhase {
			return
		}
		reporter.Report(ops.ProgressEvent{
			Kind: ops.ProgressEventKindCompletion,
			Completion: &ops.CompletionProgress{
				Phase:            event.Completion.Phase,
				CoreResource:     event.Completion.CoreResource,
				Total:            event.Completion.Total,
				Identity:         event.Completion.Identity,
				Disposition:      ops.CompletionDisposition(event.Completion.Disposition),
				FailureDetail:    event.Completion.FailureDetail,
				AffectedResource: event.Completion.AffectedResource,
				AffectedCount:    event.Completion.AffectedCount,
			},
		})
	}))
}
