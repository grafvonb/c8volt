// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"github.com/grafvonb/c8volt/c8volt/incident"
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

type walkActivityTraversalPayload struct {
	Mode             process.TraversalMode         `json:"mode"`
	Outcome          process.TraversalOutcome      `json:"outcome"`
	RootKey          string                        `json:"rootKey,omitempty"`
	Keys             []string                      `json:"keys,omitempty"`
	Edges            map[string][]string           `json:"edges,omitempty"`
	Items            []processInstanceActivityItem `json:"items,omitempty"`
	MissingAncestors []process.MissingAncestor     `json:"missingAncestors,omitempty"`
	Warning          string                        `json:"warning,omitempty"`
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
	if !flagWalkPIFlat {
		if len(result.Keys) == 0 {
			return nil
		}
		return renderIncidentEnrichedFamilyTree(cmd, result.RootKey, result.Edges, result.Items, flagWalkPIKey)
	}
	return incidentEnrichedPathView(cmd, result.Items, " ⇄ \n")
}

func walkActivityView(cmd *cobra.Command, mode string, result process.TraversalResult, items []processInstanceActivityItem) error {
	switch mode {
	case walkPIModeParent:
		if err := activityPathView(cmd, items, " ← \n"); err != nil {
			return err
		}
		printTraversalWarning(cmd, result)
		return nil
	case walkPIModeChildren:
		return activityPathView(cmd, items, " → \n")
	default:
		if !flagWalkPIFlat {
			if len(result.Keys) == 0 {
				return nil
			}
			if err := renderActivityFamilyTree(cmd, result.RootKey, result.Edges, items, flagWalkPIKey); err != nil {
				return err
			}
			printTraversalWarning(cmd, result)
			return nil
		}
		if err := activityPathView(cmd, items, " ⇄ \n"); err != nil {
			return err
		}
		printTraversalWarning(cmd, result)
		return nil
	}
}

func activityPathView(cmd *cobra.Command, items []processInstanceActivityItem, sep string) error {
	var out strings.Builder
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	for i, item := range items {
		if i > 0 {
			out.WriteString(sep)
		}
		out.WriteString(oneLinePIWithTimezone(item.Item, showTimezoneOffset))
		writeProcessInstanceActivityLinesWithTimezone(&out, "", item.Variables, item.Incidents, item.ShowIncidents, item.Item.Incident, 0, showTimezoneOffset)
	}
	renderOutputLine(cmd, "%s", out.String())
	return nil
}

// incidentEnrichedPathView renders incident details under their matching path rows.
func incidentEnrichedPathView(cmd *cobra.Command, items []process.IncidentEnrichedTraversalItem, sep string) error {
	var out strings.Builder
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	for i, item := range items {
		if i > 0 {
			out.WriteString(sep)
		}
		out.WriteString(oneLinePIWithTimezone(item.Item, showTimezoneOffset))
		writeProcessInstanceActivityLinesWithTimezone(&out, "", nil, item.Incidents, true, item.Item.Incident, 0, showTimezoneOffset)
	}
	renderOutputLine(cmd, "%s", out.String())
	return nil
}

// writeIncidentLines appends formatted incident lines as tree children.
func writeIncidentLines(out *strings.Builder, prefix string, incidents []incident.ProcessInstanceIncidentDetail) {
	writeIncidentTreeLinesWithTimezone(out, prefix, incidents, 0, false)
}

func writeIncidentTreeLines(out *strings.Builder, prefix string, incidents []incident.ProcessInstanceIncidentDetail, followingChildren int) {
	writeIncidentTreeLinesWithTimezone(out, prefix, incidents, followingChildren, false)
}

func writeIncidentTreeLinesWithTimezone(out *strings.Builder, prefix string, incidents []incident.ProcessInstanceIncidentDetail, followingChildren int, showTimezoneOffset bool) {
	for i, incident := range incidents {
		out.WriteByte('\n')
		out.WriteString(prefix)
		out.WriteString(incidentTreeBranch(i, len(incidents)+followingChildren))
		out.WriteString(incidentHumanLineWithTimezone(incident, showTimezoneOffset))
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

func activityTraversalPayload(result process.TraversalResult, items []processInstanceActivityItem) walkActivityTraversalPayload {
	return walkActivityTraversalPayload{
		Mode:             result.Mode,
		Outcome:          result.Outcome,
		RootKey:          result.RootKey,
		Keys:             append([]string(nil), result.Keys...),
		Edges:            result.Edges,
		Items:            items,
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
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	renderOutputLine(cmd, "%s", oneLinePIWithTimezone(rootItem.Item, showTimezoneOffset))
	rootChildren := edges[rootKey]
	for _, line := range formatMustActivityLinesWithTimezone("", nil, rootItem.Incidents, true, rootItem.Item.Incident, len(rootChildren), showTimezoneOffset) {
		renderOutputLine(cmd, "%s", line)
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
			out.WriteString(oneLinePIWithTimezone(item.Item, showTimezoneOffset))
			out.WriteString(marker)
			writeProcessInstanceActivityLinesWithTimezone(&out, nextPrefix, nil, item.Incidents, true, item.Item.Incident, len(edges[childKey]), showTimezoneOffset)
			renderOutputLine(cmd, "%s", out.String())
			walk(childKey, nextPrefix)
		}
	}
	walk(rootKey, "")
	return nil
}

func renderActivityFamilyTree(cmd *cobra.Command, rootKey string, edges map[string][]string, items []processInstanceActivityItem, markerKey string) error {
	itemsByKey := make(map[string]processInstanceActivityItem, len(items))
	for _, item := range items {
		itemsByKey[item.Item.Key] = item
	}

	rootItem, ok := itemsByKey[rootKey]
	if !ok {
		return fmt.Errorf("root %s not found in enriched traversal items", rootKey)
	}
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	renderOutputLine(cmd, "%s", oneLinePIWithTimezone(rootItem.Item, showTimezoneOffset))
	rootChildren := edges[rootKey]
	for _, line := range formatMustActivityLinesWithTimezone("", rootItem.Variables, rootItem.Incidents, rootItem.ShowIncidents, rootItem.Item.Incident, len(rootChildren), showTimezoneOffset) {
		renderOutputLine(cmd, "%s", line)
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
			out.WriteString(oneLinePIWithTimezone(item.Item, showTimezoneOffset))
			out.WriteString(marker)
			writeProcessInstanceActivityLinesWithTimezone(&out, nextPrefix, item.Variables, item.Incidents, item.ShowIncidents, item.Item.Incident, len(edges[childKey]), showTimezoneOffset)
			renderOutputLine(cmd, "%s", out.String())
			walk(childKey, nextPrefix)
		}
	}
	walk(rootKey, "")
	return nil
}

func formatMustActivityLines(prefix string, variables []process.ProcessInstanceVariable, incidents []incident.ProcessInstanceIncidentDetail, showIncidents bool, hasIncidentMarker bool, followingChildren int) []string {
	lines, _ := formatProcessInstanceActivityLines(prefix, variables, incidents, showIncidents, hasIncidentMarker, followingChildren)
	return lines
}

func formatMustActivityLinesWithTimezone(prefix string, variables []process.ProcessInstanceVariable, incidents []incident.ProcessInstanceIncidentDetail, showIncidents bool, hasIncidentMarker bool, followingChildren int, showTimezoneOffset bool) []string {
	lines, _ := formatProcessInstanceActivityLinesWithTimezone(prefix, variables, incidents, showIncidents, hasIncidentMarker, followingChildren, showTimezoneOffset)
	return lines
}
