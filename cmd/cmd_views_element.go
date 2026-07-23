// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strconv"
	"time"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

// elementView renders one runtime element in the selected command output mode.
func elementView(cmd *cobra.Command, item element.Element) error {
	capturedNow := time.Now().UTC()
	mode := pickMode()
	if mode == RenderModeJSON {
		return renderJSONPayload(cmd, mode, item)
	}
	if mode == RenderModeKeysOnly {
		renderOutputLine(cmd, "%s", elementKey(item))
		return nil
	}
	renderElementRowsWithListeners(cmd, []element.Element{item}, capturedNow)
	return nil
}

// elementsView renders collected runtime elements as aligned rows, keys, or JSON.
func elementsView(cmd *cobra.Command, result element.SearchResult) error {
	capturedNow := time.Now().UTC()
	mode := pickMode()
	if mode == RenderModeJSON {
		return renderJSONPayload(cmd, mode, result)
	}
	if mode == RenderModeKeysOnly {
		for _, item := range result.Items {
			renderOutputLine(cmd, "%s", elementKey(item))
		}
		return nil
	}
	renderElementRowsWithListeners(cmd, result.Items, capturedNow)
	renderOutputLine(cmd, "found: %d", len(result.Items))
	return nil
}

// oneLineElement renders a compact single element row using default timestamp display.
func oneLineElement(item element.Element) string {
	return oneLineElementWithTimezone(item, false, time.Now().UTC())
}

// oneLineElementWithTimezoneForMode binds command timezone display for keyed output.
func oneLineElementWithTimezoneForMode(cmd *cobra.Command, capturedNow time.Time) func(element.Element) string {
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	return func(item element.Element) string {
		return oneLineElementWithTimezone(item, showTimezoneOffset, capturedNow)
	}
}

// oneLineElementWithTimezone renders one compact row without alignment padding.
func oneLineElementWithTimezone(item element.Element, showTimezoneOffset bool, capturedNow time.Time) string {
	return compactFlatRow(flatRowElementWithTimezone(item, showTimezoneOffset, capturedNow))
}

// elementKey returns the stable element instance key for machine output modes.
func elementKey(item element.Element) string {
	return item.ElementInstanceKey
}

// flatRowElement renders a row using default timestamp display for unit tests and callers without command context.
func flatRowElement(item element.Element) flatRow {
	return flatRowElementWithTimezone(item, false, time.Now().UTC())
}

// flatRowElementWithTimezoneForMode binds command timezone display for list output.
func flatRowElementWithTimezoneForMode(cmd *cobra.Command, capturedNow time.Time) func(element.Element) flatRow {
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	return func(item element.Element) flatRow {
		return flatRowElementWithTimezone(item, showTimezoneOffset, capturedNow)
	}
}

// flatRowElementWithTimezone formats element timestamps and optional incident markers in the compact row grammar.
func flatRowElementWithTimezone(item element.Element, showTimezoneOffset bool, capturedNow time.Time) flatRow {
	parts := flatRow{
		item.ElementInstanceKey,
		item.TenantId,
		item.Type,
		item.ElementId,
		item.State,
		prefixedElementField("s", toolx.FormatTimestamp(item.StartDate, showTimezoneOffset)),
		prefixedElementField("e", toolx.FormatTimestamp(item.EndDate, showTimezoneOffset)),
		prefixedElementField("dur", runtimeElementDuration(item.StartDate, item.EndDate, item.State, capturedNow)),
		prefixedElementField("pi", item.ProcessInstanceKey),
		prefixedElementField("pd", item.ProcessDefinitionKey),
	}
	if marker := elementIncidentMarker(item); marker != "" {
		parts = append(parts, marker)
	}
	return parts
}

// renderElementRowsWithListeners prints aligned element rows and optional nested listener rows.
func renderElementRowsWithListeners(cmd *cobra.Command, items []element.Element, capturedNow time.Time) {
	rows := make([]flatRow, 0, len(items))
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	for _, item := range items {
		rows = append(rows, flatRowElementWithTimezone(item, showTimezoneOffset, capturedNow))
	}
	elementLines := formatFlatRows(rows)
	for i, item := range items {
		renderOutputLine(cmd, "%s", elementLines[i])
		listenerRows := formatElementListenerRows(item.Listeners, showTimezoneOffset)
		if len(listenerRows) == 0 {
			continue
		}
		renderOutputLine(cmd, "└─ listeners:")
		for j, row := range listenerRows {
			renderOutputLine(cmd, "   %s%s", incidentTreeBranch(j, len(listenerRows)), row)
		}
	}
}

// formatElementListenerRows renders element-owned listener jobs using the shared listener row grammar.
func formatElementListenerRows(listeners *[]element.RuntimeListenerJob, showTimezoneOffset bool) []string {
	if listeners == nil || len(*listeners) == 0 {
		return nil
	}
	rows := make([]flatRow, 0, len(*listeners))
	for _, listener := range *listeners {
		rows = append(rows, flatRowElementListenerWithTimezone(listener, showTimezoneOffset))
	}
	return formatFlatRows(rows)
}

// flatRowElementListenerWithTimezone formats one runtime listener job below an element row.
func flatRowElementListenerWithTimezone(item element.RuntimeListenerJob, showTimezoneOffset bool) flatRow {
	parts := flatRow{
		item.JobKey,
		item.Kind,
		prefixedJobField("lsnr", item.ListenerEventType),
		item.State,
		prefixedJobField("tp", item.Type),
		"r:" + strconv.FormatInt(int64(item.Retries), 10),
		prefixedJobField("worker", item.Worker),
	}
	if item.Deadline != nil {
		parts = append(parts, "d:"+toolx.FormatTime(*item.Deadline, showTimezoneOffset))
	} else {
		parts = append(parts, "")
	}
	if item.ErrorCode != "" {
		parts = append(parts, "ec:"+item.ErrorCode)
	} else {
		parts = append(parts, "")
	}
	if item.ErrorMessage != "" {
		parts = append(parts, "err:"+truncateHumanMessage(item.ErrorMessage, flagGetErrorMessageLimit))
	} else {
		parts = append(parts, "")
	}
	return parts
}

// prefixedElementField omits empty optional columns before flat-row alignment.
func prefixedElementField(prefix string, value string) string {
	if value == "" {
		return ""
	}
	return prefix + ":" + value
}

// elementIncidentMarker returns the single incident marker allowed in an element row.
func elementIncidentMarker(item element.Element) string {
	if !item.HasIncident {
		return ""
	}
	if item.IncidentKey != "" {
		return "inc!:" + item.IncidentKey
	}
	return "inc!"
}
