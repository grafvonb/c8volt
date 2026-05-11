// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestProcessInstanceAgeDays(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	age, ok := processInstanceAgeDays("2026-01-28T12:27:33.233Z")
	require.True(t, ok)
	require.Equal(t, 4, age)

	_, ok = processInstanceAgeDays("not-a-date")
	require.False(t, ok)
}

func TestOneLinePI_RendersAge(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	line := oneLinePI(process.ProcessInstance{
		Key:            "2251799813758959",
		TenantId:       "<default>",
		BpmnProcessId:  "Process_18qgpch",
		ProcessVersion: 6,
		State:          process.StateTerminated,
		StartDate:      "2026-01-28T12:27:33.233Z",
		EndDate:        "2026-01-29T07:42:07.044Z",
		Incident:       false,
	})

	require.Contains(t, line, "(4 days ago)")
}

func TestOneLinePI_RendersAgeToday(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	line := oneLinePI(process.ProcessInstance{
		Key:            "2251799813758959",
		TenantId:       "<default>",
		BpmnProcessId:  "Process_18qgpch",
		ProcessVersion: 6,
		State:          process.StateTerminated,
		StartDate:      "2026-02-01T07:00:00.000+00:00",
		EndDate:        "2026-02-01T09:00:00.000+00:00",
		Incident:       false,
	})

	require.Contains(t, line, "(today)")
	require.NotContains(t, line, "day ago")
}

// Protects walk/path readability by keeping standalone process-instance rows free of alignment padding.
func TestOneLinePI_UsesSingleSpacesBetweenRenderedTokens(t *testing.T) {
	line := oneLinePI(process.ProcessInstance{
		Key:            "2251799813758959",
		TenantId:       "<default>",
		BpmnProcessId:  "Process_18qgpch",
		ProcessVersion: 6,
		State:          process.StateActive,
		StartDate:      "2026-02-01T07:00:00.000+00:00",
	})

	require.NotContains(t, line, "  ")
	require.Contains(t, line, "Process_18qgpch v6 ACTIVE")
}

// Protects the shared flat-list contract: align from observed values, but preserve every character.
func TestFormatFlatRows_AlignsColumnsWithoutTruncating(t *testing.T) {
	got := formatFlatRows([]flatRow{
		{"1", "tenant", "Short", "v1"},
		{"22", "t", "MuchLongerProcess", "v12"},
	})

	require.Equal(t, []string{
		"1  tenant Short             v1",
		"22 t      MuchLongerProcess v12",
	}, got)
}

// Verifies flat process-instance lists align BPMN IDs dynamically while preserving existing field order.
func TestListProcessInstancesView_AlignsFlatRowsDynamically(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := listProcessInstancesView(cmd, process.ProcessInstances{
		Items: []process.ProcessInstance{
			{
				Key:            "1",
				TenantId:       "tenant",
				BpmnProcessId:  "Short",
				ProcessVersion: 1,
				State:          process.StateActive,
				StartDate:      "2026-02-01T07:00:00.000+00:00",
			},
			{
				Key:               "22",
				TenantId:          "tenant",
				BpmnProcessId:     "MuchLongerProcess",
				ProcessVersion:    12,
				ProcessVersionTag: "stable",
				State:             process.StateCompleted,
				StartDate:         "2026-01-30T07:00:00.000+00:00",
				ParentKey:         "1",
				Incident:          true,
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, ""+
		"1  tenant Short             v1         ACTIVE    s:2026-02-01T07:00:00+00:00 p:<root>      (today)\n"+
		"22 tenant MuchLongerProcess v12/stable COMPLETED s:2026-01-30T07:00:00+00:00 p:1      inc! (2 days ago)\n"+
		"found: 2\n", buf.String())
}

// Verifies process-definition scan output uses the same dynamic alignment as process-instance lists.
func TestListProcessDefinitionsView_AlignsFlatRowsDynamically(t *testing.T) {
	cmd := &cobra.Command{Use: "process-definition"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := listProcessDefinitionsView(cmd, process.ProcessDefinitions{
		Items: []process.ProcessDefinition{
			{
				Key:            "1",
				TenantId:       "tenant",
				BpmnProcessId:  "Short",
				ProcessVersion: 1,
			},
			{
				Key:               "22",
				TenantId:          "tenant",
				BpmnProcessId:     "MuchLongerDefinition",
				ProcessVersion:    12,
				ProcessVersionTag: "stable",
				Statistics: &process.ProcessDefinitionStatistics{
					Active:                 4,
					Completed:              9,
					Canceled:               2,
					Incidents:              3,
					IncidentCountSupported: true,
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, ""+
		"1  tenant Short                v1\n"+
		"22 tenant MuchLongerDefinition v12/stable [ac:4 cp:9 cx:2 inc:3]\n"+
		"found: 2\n", buf.String())
}

func TestOneLinePI_IncidentMarkerOnlyWhenIncidentExists(t *testing.T) {
	base := process.ProcessInstance{
		Key:            "2251799813758959",
		TenantId:       "<default>",
		BpmnProcessId:  "Process_18qgpch",
		ProcessVersion: 6,
		State:          process.StateActive,
		StartDate:      "2026-02-01T07:00:00.000+00:00",
	}

	line := oneLinePI(base)
	require.NotContains(t, line, "inc!")
	require.NotContains(t, line, "i:false")

	base.Incident = true
	line = oneLinePI(base)
	require.Contains(t, line, " inc!")
	require.NotContains(t, line, "i:true")
}

func TestOneLinePI_RendersStartAndEndDateWithOffsetSeconds(t *testing.T) {
	line := oneLinePI(process.ProcessInstance{
		Key:            "2251799813758959",
		TenantId:       "<default>",
		BpmnProcessId:  "Process_18qgpch",
		ProcessVersion: 6,
		State:          process.StateActive,
		StartDate:      "2026-04-13T18:03:24.36Z",
		EndDate:        "2026-04-13T18:03:24Z",
	})

	require.Contains(t, line, "s:2026-04-13T18:03:24+00:00")
	require.Contains(t, line, "e:2026-04-13T18:03:24+00:00")
	require.NotContains(t, line, ".360")
	require.NotContains(t, line, "Z")
}

func TestHumanTimestamp_LeavesInvalidValuesUnchanged(t *testing.T) {
	require.Equal(t, "not-a-date", humanTimestamp("not-a-date"))
	require.Equal(t, "", humanTimestamp(""))
}

func TestProcessInstancesWithAgeMeta(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	payload := processInstancesWithAgeMeta(process.ProcessInstances{
		Total: 1,
		Items: []process.ProcessInstance{{
			Key:       "2251799813758959",
			StartDate: "2026-01-28T12:27:33.233Z",
		}},
	})

	require.True(t, payload.Meta.WithAge)
	require.Equal(t, 4, payload.Meta.AgeDaysBy["2251799813758959"])
}

func TestIncidentHumanLineWithMessageLimit_RendersAlignedIncidentListFieldsAndAge(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	lines := formatIncidentListRows([]incident.ProcessInstanceIncidentDetail{
		{
			IncidentKey:            "2251799813685249",
			TenantId:               "tenant-a",
			State:                  "ACTIVE",
			ErrorType:              "JOB_NO_RETRIES",
			ErrorMessage:           "No retries left for a long-running job",
			CreationTime:           "2026-05-05T10:15:00Z",
			ProcessInstanceKey:     "2251799813711967",
			RootProcessInstanceKey: "2251799813711960",
			ProcessDefinitionKey:   "2251799813685200",
			ProcessDefinitionId:    "demo-process",
			FlowNodeId:             "task-a",
			FlowNodeInstanceKey:    "2251799813685300",
		},
		{
			IncidentKey:         "9",
			TenantId:            "<default>",
			State:               "RESOLVED",
			ErrorType:           "IO_MAPPING_ERROR",
			ErrorMessage:        "short",
			CreationTime:        "2026-05-08T10:15:00Z",
			ProcessInstanceKey:  "1",
			ProcessDefinitionId: "tiny-demo",
		},
	}, 15, false)

	require.Len(t, lines, 2)
	require.Contains(t, lines[0], "2251799813685249 tenant-a  JOB_NO_RETRIES   ACTIVE   j:n/a")
	require.Contains(t, lines[0], "j:n/a 2026-05-05T10:15:00+00:00 (4 days ago) demo-process pi:2251799813711967 root:2251799813711960")
	require.Contains(t, lines[0], "fn:task-a fni:2251799813685300 m:No retries left...")
	require.Contains(t, lines[1], "9                <default> IO_MAPPING_ERROR RESOLVED j:n/a")
	require.Contains(t, lines[1], "j:n/a 2026-05-08T10:15:00+00:00 (1 days ago) tiny-demo    pi:1")
	require.Contains(t, lines[1], "m:short")
	require.Less(t, strings.Index(lines[0], "ACTIVE"), strings.Index(lines[0], "j:n/a"))
	require.Less(t, strings.Index(lines[0], "j:n/a"), strings.Index(lines[0], "2026-05-05T10:15:00+00:00"))
	require.Less(t, strings.Index(lines[0], "2026-05-05T10:15:00+00:00"), strings.Index(lines[0], "demo-process"))
	require.Less(t, strings.Index(lines[0], "demo-process"), strings.Index(lines[0], "pi:2251799813711967"))
	require.Less(t, strings.Index(lines[0], "pi:2251799813711967"), strings.Index(lines[0], "root:2251799813711960"))
	require.Less(t, strings.Index(lines[0], "root:2251799813711960"), strings.Index(lines[0], "fn:task-a"))
	require.Less(t, strings.Index(lines[0], "fn:task-a"), strings.Index(lines[0], "fni:2251799813685300"))
	require.Less(t, strings.Index(lines[0], "fni:2251799813685300"), strings.Index(lines[0], "m:No"))
	require.Equal(t, strings.Index(lines[0], "2026-05-05T10:15:00+00:00"), strings.Index(lines[1], "2026-05-08T10:15:00+00:00"))
	require.NotContains(t, lines[0], "2251799813685200")
	require.NotContains(t, lines[0], "err:")
	require.Contains(t, lines[0], "m:No")
	require.NotContains(t, lines[0], "m: ")
}

func TestIncidentHumanLineWithMessageLimit_SkipsAgeForMissingOrInvalidCreationTime(t *testing.T) {
	line := incidentListHumanLineWithMessageLimit(incident.ProcessInstanceIncidentDetail{
		IncidentKey:  "2251799813685249",
		CreationTime: "not-a-date",
		ErrorMessage: "failed",
	}, 0)

	require.Contains(t, line, "not-a-date")
	require.NotContains(t, line, "days ago")
	require.NotContains(t, line, "(today)")
}

func TestIncidentEnrichedProcessInstancesView_JSONUsesSharedEnvelope(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := incidentEnrichedProcessInstancesView(cmd, process.IncidentEnrichedProcessInstances{
		Total: 1,
		Items: []process.IncidentEnrichedProcessInstance{{
			Item: process.ProcessInstance{Key: "123"},
			Incidents: []incident.ProcessInstanceIncidentDetail{{
				IncidentKey:        "incident-123",
				ProcessInstanceKey: "123",
				ErrorMessage:       "No retries left",
			}},
		}},
	})

	require.NoError(t, err)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), payload["total"])
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	incidents, ok := first["incidents"].([]any)
	require.True(t, ok)
	require.Len(t, incidents, 1)
}

func TestIncidentEnrichedProcessInstancesView_JSONKeepsFullMessagesAndAgeMeta(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	longMessage := "This incident message is intentionally long and must stay complete in JSON"
	err := incidentEnrichedProcessInstancesView(cmd, process.IncidentEnrichedProcessInstances{
		Total: 1,
		Items: []process.IncidentEnrichedProcessInstance{{
			Item: process.ProcessInstance{
				Key:       "2251799813758959",
				StartDate: "2026-01-28T12:27:33.233Z",
			},
			Incidents: []incident.ProcessInstanceIncidentDetail{{
				IncidentKey:        "incident-123",
				ProcessInstanceKey: "2251799813758959",
				ErrorMessage:       longMessage,
			}},
		}},
	})

	require.NoError(t, err)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	meta := requireJSONObject(t, payload["meta"])
	require.Equal(t, true, meta["withAge"])
	ageDaysBy := requireJSONObject(t, meta["ageDaysByKey"])
	require.Equal(t, float64(4), ageDaysBy["2251799813758959"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	incidents := requireJSONItems(t, first["incidents"], 1)
	incident := requireJSONObject(t, incidents[0])
	require.Equal(t, longMessage, incident["errorMessage"])
}

func TestIncidentEnrichedProcessInstancesView_HumanRowsKeepPerRowIncidentAssociation(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := incidentEnrichedProcessInstancesView(cmd, process.IncidentEnrichedProcessInstances{
		Total: 2,
		Items: []process.IncidentEnrichedProcessInstance{
			{
				Item: process.ProcessInstance{
					Key:            "123",
					TenantId:       "tenant",
					BpmnProcessId:  "demo-a",
					ProcessVersion: 3,
					State:          process.StateActive,
					StartDate:      "2026-03-23T18:00:00Z",
					Incident:       true,
				},
				Incidents: []incident.ProcessInstanceIncidentDetail{{
					IncidentKey:        "incident-123",
					ProcessInstanceKey: "123",
					State:              "ACTIVE",
					ErrorMessage:       "First key failed",
				}},
			},
			{
				Item: process.ProcessInstance{
					Key:            "124",
					TenantId:       "tenant",
					BpmnProcessId:  "demo-b",
					ProcessVersion: 4,
					State:          process.StateActive,
					StartDate:      "2026-03-23T18:05:00Z",
					Incident:       true,
				},
				Incidents: []incident.ProcessInstanceIncidentDetail{{
					IncidentKey:        "incident-124",
					ProcessInstanceKey: "124",
					State:              "ACTIVE",
					ErrorMessage:       "Second key failed",
				}},
			},
		},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 ACTIVE j:n/a m:First key failed")
	require.Contains(t, output, "124 tenant demo-b v4 ACTIVE")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-124 ACTIVE j:n/a m:Second key failed")
	require.Contains(t, output, "found: 2")
	require.Less(t, strings.Index(output, "123 tenant demo-a"), strings.Index(output, "incident-123"))
	require.Less(t, strings.Index(output, "incident-123"), strings.Index(output, "124 tenant demo-b"))
	require.Less(t, strings.Index(output, "124 tenant demo-b"), strings.Index(output, "incident-124"))
}

func TestVariableEnrichedProcessInstancesView_HumanRowsRenderIndentedSortedVariables(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := variableEnrichedProcessInstancesView(cmd, process.VariableEnrichedProcessInstances{
		Total: 1,
		Items: []process.VariableEnrichedProcessInstance{{
			Item: process.ProcessInstance{
				Key:            "123",
				TenantId:       "tenant",
				BpmnProcessId:  "demo",
				ProcessVersion: 3,
				State:          process.StateActive,
				StartDate:      "2026-03-23T18:00:00Z",
				Incident:       true,
			},
			Variables: []process.ProcessInstanceVariable{
				{Name: "alpha", Value: "1", ProcessInstanceKey: "123", ScopeKey: "123"},
				{Name: "zeta", Value: "2", ProcessInstanceKey: "123", ScopeKey: "123"},
			},
		}},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "└─ vars:")
	require.Contains(t, output, "├─ alpha=1")
	require.Contains(t, output, "└─ zeta=2")
	require.NotContains(t, output, "var alpha")
	require.NotContains(t, output, "incidents:")
	require.Contains(t, output, "found: 1")
	require.Less(t, strings.Index(output, "123 tenant demo"), strings.Index(output, "└─ vars:"))
	require.Less(t, strings.Index(output, "└─ vars:"), strings.Index(output, "alpha=1"))
	require.Less(t, strings.Index(output, "alpha=1"), strings.Index(output, "zeta=2"))
	require.Less(t, strings.Index(output, "zeta=2"), strings.Index(output, "found: 1"))
}

func TestProcessInstanceActivityInstancesView_HumanRowsGroupVarsBeforeIncidents(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := processInstanceActivityInstancesView(cmd, processInstanceActivityInstances{
		Total: 1,
		Items: []processInstanceActivityItem{{
			Item: process.ProcessInstance{
				Key:            "123",
				TenantId:       "tenant",
				BpmnProcessId:  "demo",
				ProcessVersion: 3,
				State:          process.StateActive,
				StartDate:      "2026-03-23T18:00:00Z",
				Incident:       true,
			},
			Variables: []process.ProcessInstanceVariable{{
				Name:               "businessKey",
				Value:              "2234809392328",
				ProcessInstanceKey: "123",
				ScopeKey:           "123",
			}},
			Incidents: []incident.ProcessInstanceIncidentDetail{{
				IncidentKey:         "incident-123",
				ProcessInstanceKey:  "123",
				FlowNodeId:          "task-a",
				FlowNodeInstanceKey: "element-123",
				State:               "ACTIVE",
				ErrorType:           "IO_MAPPING_ERROR",
				ErrorMessage:        "failed",
			}},
			ShowIncidents: true,
		}},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ vars:\n│  └─ businessKey=2234809392328")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 IO_MAPPING_ERROR ACTIVE j:n/a fn:task-a fni:element-123 m:failed")
	require.Less(t, strings.Index(output, "├─ vars:"), strings.Index(output, "└─ incidents:"))
}

func TestVariableEnrichedProcessInstancesView_JSONUsesSharedEnvelopeAndAgeMeta(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := variableEnrichedProcessInstancesView(cmd, process.VariableEnrichedProcessInstances{
		Total: 1,
		Items: []process.VariableEnrichedProcessInstance{{
			Item: process.ProcessInstance{
				Key:       "2251799813758959",
				StartDate: "2026-01-28T12:27:33.233Z",
			},
			Variables: []process.ProcessInstanceVariable{{
				Name:               "customerId",
				Value:              `"C-123"`,
				VariableKey:        "901",
				ProcessInstanceKey: "2251799813758959",
				ScopeKey:           "2251799813758959",
				TenantId:           "tenant-a",
				APITruncated:       true,
			}},
		}},
	})

	require.NoError(t, err)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	meta := requireJSONObject(t, payload["meta"])
	require.Equal(t, true, meta["withAge"])
	ageDaysBy := requireJSONObject(t, meta["ageDaysByKey"])
	require.Equal(t, float64(4), ageDaysBy["2251799813758959"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	variables := requireJSONItems(t, first["variables"], 1)
	variable := requireJSONObject(t, variables[0])
	require.Equal(t, `"C-123"`, variable["value"])
	require.Equal(t, true, variable["apiTruncated"])
}

func TestProcessInstanceVariableHumanLine_CompactsJSONLikeObjectsAndArrays(t *testing.T) {
	prevLimit := flagGetPIVarValueLimit
	flagGetPIVarValueLimit = 0
	t.Cleanup(func() {
		flagGetPIVarValueLimit = prevLimit
	})

	require.Equal(t, `order={"id":"O-9","amount":42}`, processInstanceVariableHumanLine(process.ProcessInstanceVariable{
		Name:  "order",
		Value: "{\n  \"id\": \"O-9\",\n  \"amount\": 42\n}",
	}))
	require.Equal(t, `items=[{"sku":"A"},{"sku":"B"}]`, processInstanceVariableHumanLine(process.ProcessInstanceVariable{
		Name:  "items",
		Value: "[\n  {\"sku\": \"A\"},\n  {\"sku\": \"B\"}\n]",
	}))
}

func TestProcessInstanceVariableHumanLine_DoesNotShortenWhenLimitUnsetOrZero(t *testing.T) {
	prevLimit := flagGetPIVarValueLimit
	t.Cleanup(func() {
		flagGetPIVarValueLimit = prevLimit
	})

	longValue := strings.Repeat("a", 120)
	flagGetPIVarValueLimit = 0
	require.Equal(t, "payload="+longValue, processInstanceVariableHumanLine(process.ProcessInstanceVariable{
		Name:  "payload",
		Value: longValue,
	}))

	flagGetPIVarValueLimit = -1
	require.Equal(t, "payload="+longValue, processInstanceVariableHumanLine(process.ProcessInstanceVariable{
		Name:  "payload",
		Value: longValue,
	}))
}

func TestProcessInstanceVariableHumanLine_AppliesCharacterSafeLimitAndCliTruncatedLabel(t *testing.T) {
	prevLimit := flagGetPIVarValueLimit
	flagGetPIVarValueLimit = 3
	t.Cleanup(func() {
		flagGetPIVarValueLimit = prevLimit
	})

	got := processInstanceVariableHumanLine(process.ProcessInstanceVariable{
		Name:  "payload",
		Value: "äöüabc",
	})

	require.Equal(t, "payload=äöü... [cli-truncated]", got)
}

func TestProcessInstanceVariableHumanLine_RendersAPIAndCombinedTruncationLabels(t *testing.T) {
	prevLimit := flagGetPIVarValueLimit
	t.Cleanup(func() {
		flagGetPIVarValueLimit = prevLimit
	})

	flagGetPIVarValueLimit = 0
	require.Equal(t, "payload=abc [api-truncated]", processInstanceVariableHumanLine(process.ProcessInstanceVariable{
		Name:         "payload",
		Value:        "abc",
		APITruncated: true,
	}))

	flagGetPIVarValueLimit = 3
	require.Equal(t, "payload=abc... [api-truncated,cli-truncated]", processInstanceVariableHumanLine(process.ProcessInstanceVariable{
		Name:         "payload",
		Value:        "abcdef",
		APITruncated: true,
	}))
}

func TestIncidentHumanLine_RendersDetailsForIncidentGroup(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	}
	prevLimit := flagGetPIIncidentMessageLimit
	flagGetPIIncidentMessageLimit = 0
	t.Cleanup(func() {
		relativeDayNow = prevNow
		flagGetPIIncidentMessageLimit = prevLimit
	})

	got := incidentHumanLine(incident.ProcessInstanceIncidentDetail{
		IncidentKey:         "incident-123",
		CreationTime:        "2026-05-06T09:29:42.711Z",
		ErrorMessage:        "No retries left",
		FlowNodeId:          "task-a",
		FlowNodeInstanceKey: "element-123",
		State:               "ACTIVE",
		ErrorType:           "JOB_NO_RETRIES",
		JobKey:              "job-123",
	})

	require.Equal(t, "incident-123 JOB_NO_RETRIES ACTIVE j:job-123 2026-05-06T09:29:42+00:00 (4 days ago) fn:task-a fni:element-123 m:No retries left", got)
	require.NotContains(t, got, "incident incident-123:")
}

func TestIncidentHumanLine_RendersUnavailableJobKeyWhenMissing(t *testing.T) {
	prevLimit := flagGetPIIncidentMessageLimit
	flagGetPIIncidentMessageLimit = 0
	t.Cleanup(func() {
		flagGetPIIncidentMessageLimit = prevLimit
	})

	got := incidentHumanLine(incident.ProcessInstanceIncidentDetail{
		IncidentKey:         "incident-123",
		ErrorMessage:        "Mapping failed",
		FlowNodeId:          "task-a",
		FlowNodeInstanceKey: "element-123",
		State:               "ACTIVE",
		ErrorType:           "IO_MAPPING_ERROR",
	})

	require.Equal(t, "incident-123 IO_MAPPING_ERROR ACTIVE j:n/a fn:task-a fni:element-123 m:Mapping failed", got)
}

func TestIncidentHumanLineWithMessageLimit_ReusesSharedIncidentRowFormatter(t *testing.T) {
	got := incidentHumanLineWithMessageLimit(incident.ProcessInstanceIncidentDetail{
		IncidentKey:  "incident-123",
		ErrorMessage: "Mapping failed in worker",
		State:        "ACTIVE",
	}, 7)

	require.Equal(t, "incident-123 ACTIVE j:n/a m:Mapping...", got)
}

func TestIncidentEnrichedProcessInstancesView_HumanIndirectMarkerRendersRowNote(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := incidentEnrichedProcessInstancesView(cmd, process.IncidentEnrichedProcessInstances{
		Total: 1,
		Items: []process.IncidentEnrichedProcessInstance{{
			Item: process.ProcessInstance{
				Key:            "123",
				TenantId:       "tenant",
				BpmnProcessId:  "demo",
				ProcessVersion: 3,
				State:          process.StateActive,
				StartDate:      "2026-03-23T18:00:00Z",
				Incident:       true,
			},
		}},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "└─ "+indirectProcessTreeIncidentNote)
	require.Contains(t, output, indirectProcessTreeIncidentWarning)
	require.Contains(t, output, "found: 1")
	require.Less(t, strings.Index(output, "123 tenant demo"), strings.Index(output, "└─ "+indirectProcessTreeIncidentNote))
	require.Less(t, strings.Index(output, "└─ "+indirectProcessTreeIncidentNote), strings.Index(output, indirectProcessTreeIncidentWarning))
	require.Less(t, strings.Index(output, indirectProcessTreeIncidentWarning), strings.Index(output, "found: 1"))
}

func TestIncidentEnrichedProcessInstancesView_HumanIndirectMarkersRenderMultipleNotesAndOneWarning(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := incidentEnrichedProcessInstancesView(cmd, process.IncidentEnrichedProcessInstances{
		Total: 2,
		Items: []process.IncidentEnrichedProcessInstance{
			{
				Item: process.ProcessInstance{
					Key:            "123",
					TenantId:       "tenant",
					BpmnProcessId:  "demo-a",
					ProcessVersion: 3,
					State:          process.StateActive,
					StartDate:      "2026-03-23T18:00:00Z",
					Incident:       true,
				},
			},
			{
				Item: process.ProcessInstance{
					Key:            "124",
					TenantId:       "tenant",
					BpmnProcessId:  "demo-b",
					ProcessVersion: 4,
					State:          process.StateActive,
					StartDate:      "2026-03-23T18:05:00Z",
					Incident:       true,
				},
			},
		},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, output, "124 tenant demo-b v4 ACTIVE")
	require.Equal(t, 2, strings.Count(output, "└─ "+indirectProcessTreeIncidentNote))
	require.Equal(t, 1, strings.Count(output, indirectProcessTreeIncidentWarning))
	require.Less(t, strings.Index(output, "123 tenant demo-a"), strings.Index(output, "└─ "+indirectProcessTreeIncidentNote))
	require.Less(t, strings.Index(output, "124 tenant demo-b"), strings.LastIndex(output, "└─ "+indirectProcessTreeIncidentNote))
	require.Less(t, strings.LastIndex(output, "└─ "+indirectProcessTreeIncidentNote), strings.Index(output, indirectProcessTreeIncidentWarning))
	require.Less(t, strings.Index(output, indirectProcessTreeIncidentWarning), strings.Index(output, "found: 2"))
}

func TestTruncateIncidentHumanMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		limit   int
		want    string
	}{
		{
			name:    "unlimited",
			message: "No retries left",
			limit:   0,
			want:    "No retries left",
		},
		{
			name:    "exact limit",
			message: "No retries left",
			limit:   15,
			want:    "No retries left",
		},
		{
			name:    "truncated",
			message: "No retries left",
			limit:   2,
			want:    "No...",
		},
		{
			name:    "multi-byte",
			message: "äöü failed",
			limit:   2,
			want:    "äö...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, truncateIncidentHumanMessage(tt.message, tt.limit))
		})
	}
}

func TestListIncidentsView_HumanJSONAndKeysOnly(t *testing.T) {
	resp := incident.Incidents{
		Total: 2,
		Items: []incident.ProcessInstanceIncidentDetail{
			{
				IncidentKey:         "incident-123",
				CreationTime:        "2026-05-06T09:29:42.711Z",
				ProcessInstanceKey:  "pi-123",
				TenantId:            "tenant-a",
				State:               "ACTIVE",
				ErrorType:           "JOB_NO_RETRIES",
				ErrorMessage:        "No retries left",
				FlowNodeId:          "task-a",
				FlowNodeInstanceKey: "element-123",
				JobKey:              "job-123",
			},
			{
				IncidentKey:  "incident-124",
				State:        "RESOLVED",
				ErrorMessage: "Mapping failed",
			},
		},
	}

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("incident")

		require.NoError(t, listIncidentsView(cmd, resp, 10, false))
		output := cmd.OutOrStdout().(*bytes.Buffer).String()

		require.Contains(t, output, "incident-123")
		require.Contains(t, output, "2026-05-06T09:29:42+00:00")
		require.Contains(t, output, "fn:task-a")
		require.Contains(t, output, "JOB_NO_RETRIES")
		require.Contains(t, output, "j:job-123")
		require.Contains(t, output, "m:No retries...")
		require.Contains(t, output, "incident-124")
		require.Contains(t, output, "j:n/a")
		require.Contains(t, output, "found: 2")
	})

	t.Run("default without messages", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("incident")

		require.NoError(t, listIncidentsView(cmd, resp, 10, true))
		output := cmd.OutOrStdout().(*bytes.Buffer).String()

		require.Contains(t, output, "incident-123")
		require.NotContains(t, output, "m:")
		require.NotContains(t, output, "No retries")
		require.Contains(t, output, "found: 2")
	})

	t.Run("json", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewAsJson = true
		cmd := newGetViewTestCommand("incident")
		setContractSupport(cmd, ContractSupportFull)

		require.NoError(t, listIncidentsView(cmd, resp, 4, false))
		output := cmd.OutOrStdout().(*bytes.Buffer).String()

		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &envelope))
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		payload := requireJSONObject(t, envelope["payload"])
		items, ok := payload["items"].([]any)
		require.True(t, ok)
		first := requireJSONObject(t, items[0])
		require.Equal(t, "No retries left", first["errorMessage"])
		require.Equal(t, "2026-05-06T09:29:42.711Z", first["creationTime"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("incident")

		require.NoError(t, listIncidentsView(cmd, resp, 0, false))

		require.Equal(t, "incident-123\nincident-124\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})
}

func TestListIncidentsView_PIKeysOnlySkipsMissingProcessInstanceKeys(t *testing.T) {
	resetViewModeFlags(t)
	flagGetIncidentPIKeysOnly = true
	cmd := newGetViewTestCommand("incident")

	err := listIncidentsView(cmd, incident.Incidents{
		Items: []incident.ProcessInstanceIncidentDetail{
			{IncidentKey: "incident-123", ProcessInstanceKey: "pi-123"},
			{IncidentKey: "incident-124"},
		},
	}, 0, false)

	require.NoError(t, err)
	require.Equal(t, "pi-123\n", cmd.OutOrStdout().(*bytes.Buffer).String())
}

func TestRenderIncidentProcessInstanceKeys_PreservesDuplicatesAndSkipsMissing(t *testing.T) {
	resetViewModeFlags(t)
	cmd := newGetViewTestCommand("incident")

	err := renderIncidentProcessInstanceKeys(cmd, []incident.ProcessInstanceIncidentDetail{
		{IncidentKey: "incident-123", ProcessInstanceKey: "pi-123"},
		{IncidentKey: "incident-124", ProcessInstanceKey: "pi-123"},
		{IncidentKey: "incident-125"},
		{IncidentKey: "incident-126", ProcessInstanceKey: "pi-126"},
	})

	require.NoError(t, err)
	require.Equal(t, "pi-123\npi-123\npi-126\n", cmd.OutOrStdout().(*bytes.Buffer).String())
}

func newGetViewTestCommand(use string) *cobra.Command {
	cmd := &cobra.Command{Use: use}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	parent := &cobra.Command{Use: "get"}
	parent.AddCommand(cmd)
	return cmd
}

func resetViewModeFlags(t *testing.T) {
	t.Helper()
	prevJSON := flagViewAsJson
	prevKeysOnly := flagViewKeysOnly
	prevPIKeysOnly := flagGetIncidentPIKeysOnly
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
		flagGetIncidentPIKeysOnly = prevPIKeysOnly
	})
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagGetIncidentPIKeysOnly = false
}
