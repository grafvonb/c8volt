// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

// elementView renders one runtime element in the selected command output mode.
func elementView(cmd *cobra.Command, item element.Element) error {
	return itemView(cmd, item, pickMode(), oneLineElementWithTimezoneForMode(cmd), elementKey)
}

// elementsView renders collected runtime elements as aligned rows, keys, or JSON.
func elementsView(cmd *cobra.Command, result element.SearchResult) error {
	return listOrJSONFlat(cmd, result, result.Items, pickMode(), flatRowElementWithTimezoneForMode(cmd), elementKey)
}

// oneLineElement renders a compact single element row using default timestamp display.
func oneLineElement(item element.Element) string {
	return oneLineElementWithTimezone(item, false)
}

// oneLineElementWithTimezoneForMode binds command timezone display for keyed output.
func oneLineElementWithTimezoneForMode(cmd *cobra.Command) func(element.Element) string {
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	return func(item element.Element) string {
		return oneLineElementWithTimezone(item, showTimezoneOffset)
	}
}

// oneLineElementWithTimezone renders one compact row without alignment padding.
func oneLineElementWithTimezone(item element.Element, showTimezoneOffset bool) string {
	return compactFlatRow(flatRowElementWithTimezone(item, showTimezoneOffset))
}

// elementKey returns the stable element instance key for machine output modes.
func elementKey(item element.Element) string {
	return item.ElementInstanceKey
}

// flatRowElement renders a row using default timestamp display for unit tests and callers without command context.
func flatRowElement(item element.Element) flatRow {
	return flatRowElementWithTimezone(item, false)
}

// flatRowElementWithTimezoneForMode binds command timezone display for list output.
func flatRowElementWithTimezoneForMode(cmd *cobra.Command) func(element.Element) flatRow {
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	return func(item element.Element) flatRow {
		return flatRowElementWithTimezone(item, showTimezoneOffset)
	}
}

// flatRowElementWithTimezone formats element timestamps and optional incident markers in the compact row grammar.
func flatRowElementWithTimezone(item element.Element, showTimezoneOffset bool) flatRow {
	parts := flatRow{
		item.ElementInstanceKey,
		item.TenantId,
		item.Type,
		item.State,
		prefixedElementField("s", toolx.FormatTimestamp(item.StartDate, showTimezoneOffset)),
		prefixedElementField("e", toolx.FormatTimestamp(item.EndDate, showTimezoneOffset)),
		prefixedElementField("pi", item.ProcessInstanceKey),
		prefixedElementField("pd", item.ProcessDefinitionKey),
		prefixedElementField("element", item.ElementId),
	}
	if marker := elementIncidentMarker(item); marker != "" {
		parts = append(parts, marker)
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
