// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

type ProcessDefinition struct {
	BpmnProcessId     string                       `json:"bpmnProcessId,omitempty"`
	Key               string                       `json:"key,omitempty"`
	Name              string                       `json:"name,omitempty"`
	TenantId          string                       `json:"tenantId,omitempty"`
	ProcessVersion    int32                        `json:"processVersion,omitempty"`
	ProcessVersionTag string                       `json:"versionTag,omitempty"`
	Statistics        *ProcessDefinitionStatistics `json:"statistics,omitempty"`
}

type ProcessDefinitionStatistics struct {
	Active                 int64 `json:"active,omitempty"`
	Canceled               int64 `json:"canceled,omitempty"`
	Completed              int64 `json:"completed,omitempty"`
	Incidents              int64 `json:"incidents,omitempty"`
	IncidentCountSupported bool  `json:"incidentCountSupported,omitempty"`
}

type ProcessDefinitionFilter struct {
	BpmnProcessId     string `json:"bpmnProcessId,omitempty"`
	Key               string `json:"key,omitempty"`
	TenantId          string `json:"tenantId,omitempty"`
	ProcessVersion    int32  `json:"processVersion,omitempty"`
	ProcessVersionTag string `json:"processVersionTag,omitempty"`
	IsLatestVersion   bool   `json:"isLatestVersion,omitempty"`
}

func (f ProcessDefinitionFilter) String() string {
	parts := make([]string, 0, 6)
	parts = appendStringFilter(parts, "bpmnProcessId", f.BpmnProcessId)
	parts = appendStringFilter(parts, "key", f.Key)
	parts = appendStringFilter(parts, "tenantId", f.TenantId)
	if f.ProcessVersion != 0 {
		parts = append(parts, fmt.Sprintf("processVersion=%d", f.ProcessVersion))
	}
	parts = appendStringFilter(parts, "processVersionTag", f.ProcessVersionTag)
	if f.IsLatestVersion {
		parts = append(parts, "isLatestVersion=true")
	}
	return formatFilterParts(parts)
}

type ProcessDefinitionStatisticsFilter struct {
	TenantId string `json:"tenantId,omitempty"`
}

func (f ProcessDefinitionStatisticsFilter) String() string {
	parts := make([]string, 0, 1)
	parts = appendStringFilter(parts, "tenantId", f.TenantId)
	return formatFilterParts(parts)
}

func SortByVersionDesc(pds []ProcessDefinition) {
	slices.SortFunc(pds, func(a, b ProcessDefinition) int {
		switch {
		case a.ProcessVersion > b.ProcessVersion:
			return -1 // a before b
		case a.ProcessVersion < b.ProcessVersion:
			return 1 // b before a
		default:
			return 0
		}
	})
}

func SortByBpmnProcessIdAscThenByVersionDesc(pds []ProcessDefinition) {
	slices.SortFunc(pds, func(a, b ProcessDefinition) int {
		if a.BpmnProcessId < b.BpmnProcessId {
			return -1
		}
		if a.BpmnProcessId > b.BpmnProcessId {
			return 1
		}
		switch {
		case a.ProcessVersion > b.ProcessVersion:
			return -1
		case a.ProcessVersion < b.ProcessVersion:
			return 1
		default:
			return 0
		}
	})
}
