// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/spf13/cobra"
)

func elementView(cmd *cobra.Command, item element.Element) error {
	return itemView(cmd, item, pickMode(), oneLineElement, elementKey)
}

func elementsView(cmd *cobra.Command, result element.SearchResult) error {
	return listOrJSONFlat(cmd, result, result.Items, pickMode(), flatRowElement, elementKey)
}

func oneLineElement(item element.Element) string {
	return compactFlatRow(flatRowElement(item))
}

func elementKey(item element.Element) string {
	return item.ElementInstanceKey
}

func flatRowElement(item element.Element) flatRow {
	parts := flatRow{
		item.ElementInstanceKey,
		item.TenantId,
		item.Type,
		item.State,
		prefixedElementField("s", item.StartDate),
		prefixedElementField("e", item.EndDate),
		prefixedElementField("pi", item.ProcessInstanceKey),
		prefixedElementField("pd", item.ProcessDefinitionKey),
		prefixedElementField("element", item.ElementId),
	}
	if marker := elementIncidentMarker(item); marker != "" {
		parts = append(parts, marker)
	}
	return parts
}

func prefixedElementField(prefix string, value string) string {
	if value == "" {
		return ""
	}
	return prefix + ":" + value
}

func elementIncidentMarker(item element.Element) string {
	if !item.HasIncident {
		return ""
	}
	if item.IncidentKey != "" {
		return "inc!:" + item.IncidentKey
	}
	return "inc!"
}
