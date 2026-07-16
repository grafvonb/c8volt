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

// ElementEnrichedProcessInstance pairs a selected process instance with the
// runtime element instances owned by that process instance.
type ElementEnrichedProcessInstance struct {
	Item     ProcessInstance
	Elements []Element
}

// ElementEnrichedProcessInstances preserves selected process-instance order
// while attaching zero or more runtime element instances to each item.
type ElementEnrichedProcessInstances struct {
	Total int32
	Items []ElementEnrichedProcessInstance
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
