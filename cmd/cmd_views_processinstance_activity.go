// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

type processInstanceActivityItem struct {
	Item          process.ProcessInstance                  `json:"item"`
	Variables     []process.ProcessInstanceVariable        `json:"variables,omitempty"`
	Incidents     []incident.ProcessInstanceIncidentDetail `json:"incidents,omitempty"`
	Elements      []process.ProcessInstanceElement         `json:"elements,omitempty"`
	ShowIncidents bool                                     `json:"-"`
}

// MarshalJSON includes requested-but-empty enrichment sections as empty arrays
// while leaving unrequested sections absent from the payload.
func (it processInstanceActivityItem) MarshalJSON() ([]byte, error) {
	payload := map[string]any{"item": it.Item}
	if it.Variables != nil {
		payload["variables"] = it.Variables
	}
	if it.ShowIncidents {
		if it.Incidents == nil {
			payload["incidents"] = []incident.ProcessInstanceIncidentDetail{}
		} else {
			payload["incidents"] = it.Incidents
		}
	}
	if it.Elements != nil {
		payload["elements"] = it.Elements
	}
	return json.Marshal(payload)
}

type processInstanceActivityInstances struct {
	Total int32                         `json:"total"`
	Items []processInstanceActivityItem `json:"items"`
}

type processInstanceActivityInstancesJSONWithMeta struct {
	Total int32                         `json:"total,omitempty"`
	Items []processInstanceActivityItem `json:"items,omitempty"`
	Meta  processInstanceAgeMeta        `json:"meta"`
}

func processInstanceActivityInstancesView(cmd *cobra.Command, resp processInstanceActivityInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, processInstanceActivityInstancesWithAgeMeta(resp))
	}
	needsIndirectIncidentWarning := renderProcessInstanceActivityRows(cmd, resp.Items)
	if needsIndirectIncidentWarning {
		renderHumanWarningLine(cmd, indirectProcessTreeIncidentWarning)
	}
	renderOutputLine(cmd, "found: %d", len(resp.Items))
	return nil
}

func processInstanceActivityInstancesWithAgeMeta(resp processInstanceActivityInstances) processInstanceActivityInstancesJSONWithMeta {
	meta := processInstanceAgeMeta{WithAge: true, AgeDaysBy: map[string]int{}}
	for _, it := range resp.Items {
		if age, ok := processInstanceAgeDays(it.Item.StartDate); ok {
			meta.AgeDaysBy[it.Item.Key] = age
		}
	}
	if len(meta.AgeDaysBy) == 0 {
		meta.AgeDaysBy = nil
	}
	return processInstanceActivityInstancesJSONWithMeta{
		Total: resp.Total,
		Items: resp.Items,
		Meta:  meta,
	}
}

func renderProcessInstanceActivityRows(cmd *cobra.Command, items []processInstanceActivityItem) bool {
	rows := make([]flatRow, 0, len(items))
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	capturedNow := time.Now().UTC()
	for _, it := range items {
		rows = append(rows, flatRowPIWithTimezone(it.Item, showTimezoneOffset))
	}
	lines := formatFlatRows(rows)
	needsIndirectIncidentWarning := false
	for i, it := range items {
		renderOutputLine(cmd, "%s", lines[i])
		detailLines, needsWarning := formatProcessInstanceActivityLinesWithElementsWithTimezone("", it.Variables, it.Incidents, it.Elements, it.ShowIncidents, it.Item.Incident, 0, showTimezoneOffset, capturedNow)
		for _, line := range detailLines {
			renderOutputLine(cmd, "%s", line)
		}
		needsIndirectIncidentWarning = needsIndirectIncidentWarning || needsWarning
	}
	return needsIndirectIncidentWarning
}

func formatProcessInstanceActivityLines(prefix string, variables []process.ProcessInstanceVariable, incidents []incident.ProcessInstanceIncidentDetail, showIncidents bool, hasIncidentMarker bool, followingChildren int) ([]string, bool) {
	return formatProcessInstanceActivityLinesWithTimezone(prefix, variables, incidents, showIncidents, hasIncidentMarker, followingChildren, false)
}

func formatProcessInstanceActivityLinesWithTimezone(prefix string, variables []process.ProcessInstanceVariable, incidents []incident.ProcessInstanceIncidentDetail, showIncidents bool, hasIncidentMarker bool, followingChildren int, showTimezoneOffset bool) ([]string, bool) {
	return formatProcessInstanceActivityLinesWithElementsWithTimezone(prefix, variables, incidents, nil, showIncidents, hasIncidentMarker, followingChildren, showTimezoneOffset, time.Now().UTC())
}

func formatProcessInstanceActivityLinesWithElementsWithTimezone(prefix string, variables []process.ProcessInstanceVariable, incidents []incident.ProcessInstanceIncidentDetail, elements []process.ProcessInstanceElement, showIncidents bool, hasIncidentMarker bool, followingChildren int, showTimezoneOffset bool, capturedNow time.Time) ([]string, bool) {
	hasVars := len(variables) > 0
	hasIncidents := showIncidents && (len(incidents) > 0 || hasIncidentMarker)
	hasElements := len(elements) > 0
	sectionCount := 0
	if hasVars {
		sectionCount++
	}
	if hasIncidents {
		sectionCount++
	}
	if hasElements {
		sectionCount++
	}
	if sectionCount == 0 {
		return nil, false
	}

	totalBranches := sectionCount + followingChildren
	sectionIndex := 0
	lines := make([]string, 0, sectionCount+len(variables)+len(incidents)+len(elements)+1)
	if hasVars {
		branch := incidentTreeBranch(sectionIndex, totalBranches)
		childPrefix := treeChildPrefix(prefix, sectionIndex, totalBranches)
		lines = append(lines, prefix+branch+"vars:")
		for i, variable := range variables {
			lines = append(lines, childPrefix+incidentTreeBranch(i, len(variables))+processInstanceVariableHumanLine(variable))
		}
		sectionIndex++
	}

	needsIndirectIncidentWarning := false
	if hasIncidents {
		branch := incidentTreeBranch(sectionIndex, totalBranches)
		childPrefix := treeChildPrefix(prefix, sectionIndex, totalBranches)
		lines = append(lines, prefix+branch+"incidents:")
		if len(incidents) > 0 {
			for i, incident := range incidents {
				lines = append(lines, childPrefix+incidentTreeBranch(i, len(incidents))+incidentHumanLineWithTimezone(incident, showTimezoneOffset))
			}
		} else {
			lines = append(lines, childPrefix+"└─ "+indirectProcessTreeIncidentNote)
			needsIndirectIncidentWarning = true
		}
		sectionIndex++
	}

	if hasElements {
		branch := incidentTreeBranch(sectionIndex, totalBranches)
		childPrefix := treeChildPrefix(prefix, sectionIndex, totalBranches)
		lines = append(lines, prefix+branch+"elements:")
		elementLines := formatProcessInstanceElementTreeLines(elements, showTimezoneOffset, capturedNow)
		for _, line := range elementLines {
			lines = append(lines, childPrefix+line)
		}
	}
	return lines, needsIndirectIncidentWarning
}

func treeChildPrefix(prefix string, branchIndex, totalBranches int) string {
	if branchIndex == totalBranches-1 {
		return prefix + "   "
	}
	return prefix + "│  "
}

func writeProcessInstanceActivityLines(out *strings.Builder, prefix string, variables []process.ProcessInstanceVariable, incidents []incident.ProcessInstanceIncidentDetail, showIncidents bool, hasIncidentMarker bool, followingChildren int) bool {
	return writeProcessInstanceActivityLinesWithTimezone(out, prefix, variables, incidents, showIncidents, hasIncidentMarker, followingChildren, false)
}

// writeProcessInstanceActivityLinesWithTimezone keeps existing variable and
// incident call sites on the element-aware formatter with no element section.
func writeProcessInstanceActivityLinesWithTimezone(out *strings.Builder, prefix string, variables []process.ProcessInstanceVariable, incidents []incident.ProcessInstanceIncidentDetail, showIncidents bool, hasIncidentMarker bool, followingChildren int, showTimezoneOffset bool) bool {
	lines, needsWarning := formatProcessInstanceActivityLinesWithElementsWithTimezone(prefix, variables, incidents, nil, showIncidents, hasIncidentMarker, followingChildren, showTimezoneOffset, time.Now().UTC())
	for _, line := range lines {
		out.WriteByte('\n')
		out.WriteString(line)
	}
	return needsWarning
}

// writeProcessInstanceActivityLinesWithElementsWithTimezone appends all
// requested activity sections while preserving the caller's tree prefix.
func writeProcessInstanceActivityLinesWithElementsWithTimezone(out *strings.Builder, prefix string, variables []process.ProcessInstanceVariable, incidents []incident.ProcessInstanceIncidentDetail, elements []process.ProcessInstanceElement, showIncidents bool, hasIncidentMarker bool, followingChildren int, showTimezoneOffset bool, capturedNow time.Time) bool {
	lines, needsWarning := formatProcessInstanceActivityLinesWithElementsWithTimezone(prefix, variables, incidents, elements, showIncidents, hasIncidentMarker, followingChildren, showTimezoneOffset, capturedNow)
	for _, line := range lines {
		out.WriteByte('\n')
		out.WriteString(line)
	}
	return needsWarning
}

func activityFromIncidentEnriched(resp process.IncidentEnrichedProcessInstances) processInstanceActivityInstances {
	items := make([]processInstanceActivityItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, processInstanceActivityItem{
			Item:          it.Item,
			Incidents:     it.Incidents,
			ShowIncidents: true,
		})
	}
	return processInstanceActivityInstances{Total: resp.Total, Items: items}
}

func activityFromVariableEnriched(resp process.VariableEnrichedProcessInstances) processInstanceActivityInstances {
	items := make([]processInstanceActivityItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, processInstanceActivityItem{
			Item:      it.Item,
			Variables: it.Variables,
		})
	}
	return processInstanceActivityInstances{Total: resp.Total, Items: items}
}

func activityFromElementEnriched(resp process.ElementEnrichedProcessInstances) processInstanceActivityInstances {
	items := make([]processInstanceActivityItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, processInstanceActivityItem{
			Item:     it.Item,
			Elements: it.Elements,
		})
	}
	return processInstanceActivityInstances{Total: resp.Total, Items: items}
}

func elementEnrichedProcessInstancesView(cmd *cobra.Command, resp process.ElementEnrichedProcessInstances) error {
	return processInstanceActivityInstancesView(cmd, activityFromElementEnriched(resp))
}

type processInstanceActivityEnrichments struct {
	Incidents *process.IncidentEnrichedProcessInstances
	Variables *process.VariableEnrichedProcessInstances
	Elements  *process.ElementEnrichedProcessInstances
}

// mergeProcessInstanceActivity preserves the selected process-instance order
// while attaching whichever enrichment sections the command requested.
func mergeProcessInstanceActivity(pis process.ProcessInstances, enrichments processInstanceActivityEnrichments) processInstanceActivityInstances {
	incidentsByKey := map[string][]incident.ProcessInstanceIncidentDetail{}
	if enrichments.Incidents != nil {
		incidentsByKey = make(map[string][]incident.ProcessInstanceIncidentDetail, len(enrichments.Incidents.Items))
		for _, it := range enrichments.Incidents.Items {
			incidentsByKey[it.Item.Key] = it.Incidents
		}
	}

	varsByKey := map[string][]process.ProcessInstanceVariable{}
	if enrichments.Variables != nil {
		varsByKey = make(map[string][]process.ProcessInstanceVariable, len(enrichments.Variables.Items))
		for _, it := range enrichments.Variables.Items {
			varsByKey[it.Item.Key] = it.Variables
		}
	}

	elementsByKey := map[string][]process.ProcessInstanceElement{}
	if enrichments.Elements != nil {
		elementsByKey = make(map[string][]process.ProcessInstanceElement, len(enrichments.Elements.Items))
		for _, it := range enrichments.Elements.Items {
			elementsByKey[it.Item.Key] = it.Elements
		}
	}

	items := make([]processInstanceActivityItem, 0, len(pis.Items))
	for _, item := range pis.Items {
		var variables []process.ProcessInstanceVariable
		if enrichments.Variables != nil {
			variables = varsByKey[item.Key]
			if variables == nil {
				variables = []process.ProcessInstanceVariable{}
			}
		}
		var incidents []incident.ProcessInstanceIncidentDetail
		if enrichments.Incidents != nil {
			incidents = incidentsByKey[item.Key]
			if incidents == nil {
				incidents = []incident.ProcessInstanceIncidentDetail{}
			}
		}
		var elements []process.ProcessInstanceElement
		if enrichments.Elements != nil {
			elements = elementsByKey[item.Key]
			if elements == nil {
				elements = []process.ProcessInstanceElement{}
			}
		}
		items = append(items, processInstanceActivityItem{
			Item:          item,
			Variables:     variables,
			Incidents:     incidents,
			Elements:      elements,
			ShowIncidents: enrichments.Incidents != nil,
		})
	}
	return processInstanceActivityInstances{
		Total: pis.Total,
		Items: items,
	}
}

// mergeIncidentAndVariableActivity keeps the existing two-section call sites
// and tests on the general merger used by combined element enrichment.
func mergeIncidentAndVariableActivity(incidents process.IncidentEnrichedProcessInstances, variables process.VariableEnrichedProcessInstances) processInstanceActivityInstances {
	pis := process.ProcessInstances{Total: incidents.Total, Items: make([]process.ProcessInstance, 0, len(incidents.Items))}
	for _, it := range incidents.Items {
		pis.Items = append(pis.Items, it.Item)
	}
	return mergeProcessInstanceActivity(pis, processInstanceActivityEnrichments{
		Incidents: &incidents,
		Variables: &variables,
	})
}

func formatProcessInstanceElementRows(elements []process.ProcessInstanceElement, showTimezoneOffset bool, capturedNow time.Time) []string {
	rows := make([]flatRow, 0, len(elements))
	for _, element := range elements {
		rows = append(rows, flatRowProcessInstanceElementWithTimezone(element, showTimezoneOffset, capturedNow))
	}
	return formatFlatRows(rows)
}

func formatProcessInstanceElementTreeLines(elements []process.ProcessInstanceElement, showTimezoneOffset bool, capturedNow time.Time) []string {
	elementRows := formatProcessInstanceElementRows(elements, showTimezoneOffset, capturedNow)
	lines := make([]string, 0, len(elementRows))
	for i, element := range elements {
		lines = append(lines, incidentTreeBranch(i, len(elements))+elementRows[i])
		listenerRows := formatProcessInstanceElementListenerRows(element.Listeners, showTimezoneOffset)
		if len(listenerRows) == 0 {
			continue
		}
		childPrefix := treeChildPrefix("", i, len(elements))
		lines = append(lines, childPrefix+"└─ listeners:")
		listenerPrefix := childPrefix + "   "
		for j, row := range listenerRows {
			lines = append(lines, listenerPrefix+incidentTreeBranch(j, len(listenerRows))+row)
		}
	}
	return lines
}

func flatRowProcessInstanceElementWithTimezone(item process.ProcessInstanceElement, showTimezoneOffset bool, capturedNow time.Time) flatRow {
	parts := flatRow{
		item.ElementInstanceKey,
		item.Type,
		item.ElementId,
		item.State,
		prefixedElementField("s", toolx.FormatTimestamp(item.StartDate, showTimezoneOffset)),
		prefixedElementField("e", toolx.FormatTimestamp(item.EndDate, showTimezoneOffset)),
		prefixedElementField("dur", runtimeElementDuration(item.StartDate, item.EndDate, item.State, capturedNow)),
	}
	if marker := processInstanceElementIncidentMarker(item); marker != "" {
		parts = append(parts, marker)
	}
	return parts
}

func formatProcessInstanceElementListenerRows(listeners *[]process.RuntimeListenerJob, showTimezoneOffset bool) []string {
	if listeners == nil || len(*listeners) == 0 {
		return nil
	}
	rows := make([]flatRow, 0, len(*listeners))
	for _, listener := range *listeners {
		rows = append(rows, flatRowProcessInstanceElementListenerWithTimezone(listener, showTimezoneOffset))
	}
	return formatFlatRows(rows)
}

func flatRowProcessInstanceElementListenerWithTimezone(item process.RuntimeListenerJob, showTimezoneOffset bool) flatRow {
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

func processInstanceElementIncidentMarker(item process.ProcessInstanceElement) string {
	if !item.HasIncident {
		return ""
	}
	if item.IncidentKey != "" {
		return "inc!:" + item.IncidentKey
	}
	return "inc!"
}

func processInstancesFromTraversal(result process.TraversalResult) process.ProcessInstances {
	items := make([]process.ProcessInstance, 0, len(result.Keys))
	for _, key := range result.Keys {
		if item, ok := result.Chain[key]; ok {
			items = append(items, item)
		}
	}
	return process.ProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}
}

// activityItemsFromTraversal preserves traversal order while overlaying any
// requested activity enrichments by owning process-instance key.
func activityItemsFromTraversal(result process.TraversalResult, incidents process.IncidentEnrichedTraversalResult, variables process.VariableEnrichedProcessInstances, elements process.ElementEnrichedProcessInstances, showIncidents bool) []processInstanceActivityItem {
	incidentsByKey := make(map[string][]incident.ProcessInstanceIncidentDetail, len(incidents.Items))
	for _, item := range incidents.Items {
		incidentsByKey[item.Item.Key] = item.Incidents
	}
	varsByKey := make(map[string][]process.ProcessInstanceVariable, len(variables.Items))
	for _, item := range variables.Items {
		varsByKey[item.Item.Key] = item.Variables
	}
	elementsByKey := make(map[string][]process.ProcessInstanceElement, len(elements.Items))
	for _, item := range elements.Items {
		elementsByKey[item.Item.Key] = item.Elements
	}
	showElements := elements.Items != nil

	items := make([]processInstanceActivityItem, 0, len(result.Keys))
	for _, key := range result.Keys {
		item, ok := result.Chain[key]
		if !ok {
			continue
		}
		items = append(items, processInstanceActivityItem{
			Item:          item,
			Variables:     varsByKey[key],
			Incidents:     incidentsByKey[key],
			Elements:      traversalElementsForKey(elementsByKey, key, showElements),
			ShowIncidents: showIncidents,
		})
	}
	return items
}

// traversalElementsForKey distinguishes unrequested element enrichment from a
// requested lookup that returned no rows for a walked process instance.
func traversalElementsForKey(elementsByKey map[string][]process.ProcessInstanceElement, key string, showElements bool) []process.ProcessInstanceElement {
	if !showElements {
		return nil
	}
	if elements := elementsByKey[key]; elements != nil {
		return elements
	}
	return []process.ProcessInstanceElement{}
}
