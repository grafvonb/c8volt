// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type IncidentEnrichedProcessInstance struct {
	Item      ProcessInstance
	Incidents []ProcessInstanceIncidentDetail
}

type IncidentEnrichedProcessInstances struct {
	Total int32
	Items []IncidentEnrichedProcessInstance
}

type VariableEnrichedProcessInstance struct {
	Item      ProcessInstance
	Variables []ProcessInstanceVariable
}

type VariableEnrichedProcessInstances struct {
	Total int32
	Items []VariableEnrichedProcessInstance
}

type IncidentEnrichedTraversalItem struct {
	Item      ProcessInstance
	Incidents []ProcessInstanceIncidentDetail
}

type IncidentEnrichedTraversalResult struct {
	Mode             string
	Outcome          string
	StartKey         string
	RootKey          string
	Keys             []string
	Edges            map[string][]string
	Items            []IncidentEnrichedTraversalItem
	MissingAncestors []MissingAncestor
	Warning          string
}
