package domain

import "slices"

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
	Active    int64 `json:"active,omitempty"`
	Canceled  int64 `json:"canceled,omitempty"`
	Completed int64 `json:"completed,omitempty"`
	Incidents int64 `json:"incidents,omitempty"`
}

type ProcessDefinitionFilter struct {
	BpmnProcessId     string `json:"bpmnProcessId,omitempty"`
	Key               string `json:"key,omitempty"`
	TenantId          string `json:"tenantId,omitempty"`
	ProcessVersion    int32  `json:"processVersion,omitempty"`
	ProcessVersionTag string `json:"processVersionTag,omitempty"`
	IsLatestVersion   bool   `json:"isLatestVersion,omitempty"`
}

type ProcessDefinitionStatisticsFilter struct {
	TenantId string `json:"tenantId,omitempty"`
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
