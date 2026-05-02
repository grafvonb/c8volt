// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

type walkIncidentEnrichedTraversalPayload struct {
	Mode             process.TraversalMode                   `json:"mode"`
	Outcome          process.TraversalOutcome                `json:"outcome"`
	RootKey          string                                  `json:"rootKey,omitempty"`
	Keys             []string                                `json:"keys,omitempty"`
	Edges            map[string][]string                     `json:"edges,omitempty"`
	Items            []process.IncidentEnrichedTraversalItem `json:"items,omitempty"`
	MissingAncestors []process.MissingAncestor               `json:"missingAncestors,omitempty"`
	Warning          string                                  `json:"warning,omitempty"`
}

// incidentEnrichedAncestorsView renders enriched ancestor walks in path order.
func incidentEnrichedAncestorsView(cmd *cobra.Command, result process.IncidentEnrichedTraversalResult) error {
	return incidentEnrichedPathView(cmd, result.Items, " ← \n")
}

// incidentEnrichedDescendantsView renders enriched descendant walks in path order.
func incidentEnrichedDescendantsView(cmd *cobra.Command, result process.IncidentEnrichedTraversalResult) error {
	return incidentEnrichedPathView(cmd, result.Items, " → \n")
}

// incidentEnrichedFamilyView renders enriched family walks as either path or tree output.
func incidentEnrichedFamilyView(cmd *cobra.Command, result process.IncidentEnrichedTraversalResult) error {
	if pickMode() == RenderModeTree {
		if len(result.Keys) == 0 {
			return nil
		}
		return renderIncidentEnrichedFamilyTree(cmd, result.RootKey, result.Edges, result.Items, flagWalkPIKey)
	}
	return incidentEnrichedPathView(cmd, result.Items, " ⇄ \n")
}

// incidentEnrichedPathView renders incident details under their matching path rows.
func incidentEnrichedPathView(cmd *cobra.Command, items []process.IncidentEnrichedTraversalItem, sep string) error {
	var out strings.Builder
	for i, item := range items {
		if i > 0 {
			out.WriteString(sep)
		}
		out.WriteString(oneLinePI(item.Item))
		writeIncidentLines(&out, "  ", item.Incidents)
	}
	renderOutputLine(cmd, "%s", out.String())
	return nil
}

// writeIncidentLines appends formatted incident lines using the caller's indentation prefix.
func writeIncidentLines(out *strings.Builder, prefix string, incidents []process.ProcessInstanceIncidentDetail) {
	for _, incident := range incidents {
		out.WriteByte('\n')
		out.WriteString(prefix)
		out.WriteString(incidentHumanLine(incident))
	}
}

// incidentEnrichedTraversalPayload preserves traversal metadata while attaching incident details for JSON output.
func incidentEnrichedTraversalPayload(result process.IncidentEnrichedTraversalResult) walkIncidentEnrichedTraversalPayload {
	return walkIncidentEnrichedTraversalPayload{
		Mode:             result.Mode,
		Outcome:          result.Outcome,
		RootKey:          result.RootKey,
		Keys:             append([]string(nil), result.Keys...),
		Edges:            result.Edges,
		Items:            result.Items,
		MissingAncestors: append([]process.MissingAncestor(nil), result.MissingAncestors...),
		Warning:          result.Warning,
	}
}

// printIncidentEnrichedTraversalWarning renders traversal warnings from enriched walk results.
func printIncidentEnrichedTraversalWarning(cmd *cobra.Command, result process.IncidentEnrichedTraversalResult) {
	if result.Warning == "" && len(result.MissingAncestors) == 0 {
		return
	}

	printTraversalWarningDetails(cmd, result.Warning, result.MissingAncestors)
}

// renderIncidentEnrichedFamilyTree prints enriched descendants as an ASCII tree starting from rootKey.
func renderIncidentEnrichedFamilyTree(cmd *cobra.Command, rootKey string, edges map[string][]string, items []process.IncidentEnrichedTraversalItem, markerKey string) error {
	itemsByKey := make(map[string]process.IncidentEnrichedTraversalItem, len(items))
	for _, item := range items {
		itemsByKey[item.Item.Key] = item
	}

	rootItem, ok := itemsByKey[rootKey]
	if !ok {
		return fmt.Errorf("root %s not found in enriched traversal items", rootKey)
	}
	renderOutputLine(cmd, "%s", oneLinePI(rootItem.Item))
	for _, incident := range rootItem.Incidents {
		renderOutputLine(cmd, "  %s", incidentHumanLine(incident))
	}

	var walk func(parentKey, prefix string)
	walk = func(parentKey, prefix string) {
		children := edges[parentKey]
		for i, childKey := range children {
			last := i == len(children)-1
			branch := "├─ "
			nextPrefix := prefix + "│  "
			if last {
				branch = "└─ "
				nextPrefix = prefix + "   "
			}
			item, ok := itemsByKey[childKey]
			if !ok {
				continue
			}
			marker := ""
			if childKey == markerKey {
				marker = " (--key)"
			}
			var out strings.Builder
			out.WriteString(prefix)
			out.WriteString(branch)
			out.WriteString(oneLinePI(item.Item))
			out.WriteString(marker)
			writeIncidentLines(&out, nextPrefix+"  ", item.Incidents)
			renderOutputLine(cmd, "%s", out.String())
			walk(childKey, nextPrefix)
		}
	}
	walk(rootKey, "")
	return nil
}
