// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
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
		"1  tenant Short             v1         ACTIVE    s:2026-02-01T07:00:00.000 p:<root>      (today)\n"+
		"22 tenant MuchLongerProcess v12/stable COMPLETED s:2026-01-30T07:00:00.000 p:1      inc! (2 days ago)\n"+
		"found: 2\n", buf.String())
}

func TestListProcessInstancesView_UsesConfiguredTimezoneOffset(t *testing.T) {
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
	cfg := config.New()
	cfg.App.ShowTimezoneOffset = true
	cmd.SetContext(cfg.ToContext(context.Background()))

	err := listProcessInstancesView(cmd, process.ProcessInstances{
		Items: []process.ProcessInstance{{
			Key:            "1",
			TenantId:       "tenant",
			BpmnProcessId:  "Short",
			ProcessVersion: 1,
			State:          process.StateActive,
			StartDate:      "2026-02-01T07:00:00Z",
		}},
	})

	require.NoError(t, err)
	require.Contains(t, buf.String(), "s:2026-02-01T07:00:00.000+00:00")
}

// Verifies shared JSON process-instance list rendering preserves lifecycle state fields for command payloads.
func TestListProcessInstancesView_JSONPayloadIncludesState(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := listProcessInstancesView(cmd, process.ProcessInstances{
		Total: 1,
		Items: []process.ProcessInstance{{
			Key:            "2251799813711967",
			TenantId:       "<default>",
			BpmnProcessId:  "order-process",
			ProcessVersion: 3,
			State:          process.StateCompleted,
			StartDate:      "2026-05-23T12:00:00Z",
		}},
	})

	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	items, ok := got["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813711967", item["key"])
	require.Equal(t, "COMPLETED", item["state"])
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

func TestOneLinePI_RendersStartAndEndDateWithMilliseconds(t *testing.T) {
	line := oneLinePI(process.ProcessInstance{
		Key:            "2251799813758959",
		TenantId:       "<default>",
		BpmnProcessId:  "Process_18qgpch",
		ProcessVersion: 6,
		State:          process.StateActive,
		StartDate:      "2026-04-13T18:03:24.36Z",
		EndDate:        "2026-04-13T18:03:24Z",
	})

	require.Contains(t, line, "s:2026-04-13T18:03:24.360")
	require.Contains(t, line, "e:2026-04-13T18:03:24.000")
	require.NotContains(t, line, "Z")
	require.NotContains(t, line, "+00:00")
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
				IncidentKey:        "incident-123",
				ProcessInstanceKey: "123",
				ElementId:          "task-a",
				ElementInstanceKey: "element-123",
				State:              "ACTIVE",
				ErrorType:          "IO_MAPPING_ERROR",
				ErrorMessage:       "failed",
			}},
			ShowIncidents: true,
		}},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ vars:\n│  └─ businessKey=2234809392328")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 IO_MAPPING_ERROR ACTIVE j:n/a e:task-a ei:element-123 m:failed")
	require.Less(t, strings.Index(output, "├─ vars:"), strings.Index(output, "└─ incidents:"))
}

// TestProcessInstanceActivityInstancesView_HumanRowsGroupVarsIncidentsAndElements verifies combined sections stay in contract order.
func TestProcessInstanceActivityInstancesView_HumanRowsGroupVarsIncidentsAndElements(t *testing.T) {
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
				StartDate:      "2026-07-15T10:12:00Z",
				Incident:       true,
			},
			Variables: []process.ProcessInstanceVariable{{
				Name:               "businessKey",
				Value:              "2234809392328",
				ProcessInstanceKey: "123",
				ScopeKey:           "123",
			}},
			Incidents: []incident.ProcessInstanceIncidentDetail{{
				IncidentKey:        "incident-123",
				ProcessInstanceKey: "123",
				ElementId:          "task-a",
				ElementInstanceKey: "element-123",
				State:              "ACTIVE",
				ErrorType:          "IO_MAPPING_ERROR",
				ErrorMessage:       "failed",
			}},
			Elements: []process.ProcessInstanceElement{{
				ElementInstanceKey: "element-1",
				ElementId:          "task-a",
				Type:               "SERVICE_TASK",
				State:              "ACTIVE",
				StartDate:          "2026-07-15T10:12:01Z",
				ProcessInstanceKey: "123",
			}},
			ShowIncidents: true,
		}},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ vars:\n│  └─ businessKey=2234809392328")
	require.Contains(t, output, "├─ incidents:\n│  └─ incident-123 IO_MAPPING_ERROR ACTIVE j:n/a e:task-a ei:element-123 m:failed")
	require.Contains(t, output, "└─ elements:\n   └─ element-1 SERVICE_TASK task-a ACTIVE s:2026-07-15T10:12:01.000")
	require.Equal(t, 1, strings.Count(output, "vars:"))
	require.Equal(t, 1, strings.Count(output, "incidents:"))
	require.Equal(t, 1, strings.Count(output, "elements:"))
	require.Less(t, strings.Index(output, "├─ vars:"), strings.Index(output, "├─ incidents:"))
	require.Less(t, strings.Index(output, "├─ incidents:"), strings.Index(output, "└─ elements:"))
}

func TestProcessInstanceActivityInstancesView_HumanRowsRenderElements(t *testing.T) {
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
				StartDate:      "2026-07-15T10:12:00Z",
			},
			Elements: []process.ProcessInstanceElement{
				{ElementInstanceKey: "element-1", Type: "START_EVENT", ElementId: "start", State: "COMPLETED", StartDate: "2026-07-15T10:12:01Z", EndDate: "2026-07-15T10:12:02Z", ProcessInstanceKey: "123"},
				{ElementInstanceKey: "element-2", Type: "USER_TASK", ElementId: "review", State: "ACTIVE", StartDate: "2026-07-15T10:12:03Z", ProcessInstanceKey: "123", HasIncident: true},
				{ElementInstanceKey: "element-3", Type: "SERVICE_TASK", ElementId: "ship-order", State: "ACTIVE", StartDate: "2026-07-15T10:12:03Z", ProcessInstanceKey: "123", HasIncident: true, IncidentKey: "incident-777"},
			},
		}},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "└─ elements:")
	require.Contains(t, output, "element-1 START_EVENT  start      COMPLETED s:2026-07-15T10:12:01.000 e:2026-07-15T10:12:02.000 dur:1s")
	require.Contains(t, output, "element-2 USER_TASK    review     ACTIVE    s:2026-07-15T10:12:03.000")
	require.Contains(t, output, "element-2 USER_TASK    review     ACTIVE")
	require.Contains(t, output, "dur:")
	require.Contains(t, output, "inc!")
	require.Contains(t, output, "element-3 SERVICE_TASK ship-order ACTIVE")
	require.Contains(t, output, "inc!:incident-777")
	require.NotContains(t, output, "element:ship-order")
	require.Contains(t, output, "found: 1")
}

// TestProcessInstanceActivityInstancesView_HumanRowsRenderRepeatedElementsSeparately verifies looped BPMN executions remain distinct rows.
func TestProcessInstanceActivityInstancesView_HumanRowsRenderRepeatedElementsSeparately(t *testing.T) {
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
			},
			Elements: []process.ProcessInstanceElement{
				{ElementInstanceKey: "element-loop-1", Type: "SERVICE_TASK", ElementId: "retry-task", State: "COMPLETED", StartDate: "2026-07-15T10:12:01Z", EndDate: "2026-07-15T10:12:02Z", ProcessInstanceKey: "123"},
				{ElementInstanceKey: "element-loop-2", Type: "SERVICE_TASK", ElementId: "retry-task", State: "ACTIVE", StartDate: "2026-07-15T10:12:03Z", ProcessInstanceKey: "123"},
			},
		}},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "element-loop-1 SERVICE_TASK retry-task COMPLETED")
	require.Contains(t, output, "element-loop-2 SERVICE_TASK retry-task ACTIVE")
	require.Equal(t, 2, strings.Count(output, "retry-task"))
}

// TestProcessInstanceActivityInstancesView_JSONIncludesCombinedEnrichmentFields verifies one payload item can carry all enrichment arrays.
func TestProcessInstanceActivityInstancesView_JSONIncludesCombinedEnrichmentFields(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := processInstanceActivityInstancesView(cmd, processInstanceActivityInstances{
		Total: 1,
		Items: []processInstanceActivityItem{{
			Item: process.ProcessInstance{
				Key:       "123",
				StartDate: "2026-07-15T10:12:00Z",
			},
			Variables: []process.ProcessInstanceVariable{{
				Name:               "businessKey",
				Value:              `"C-123"`,
				VariableKey:        "901",
				ProcessInstanceKey: "123",
				ScopeKey:           "123",
			}},
			Incidents: []incident.ProcessInstanceIncidentDetail{{
				IncidentKey:        "incident-123",
				ProcessInstanceKey: "123",
				ErrorType:          "IO_MAPPING_ERROR",
				State:              "ACTIVE",
			}},
			Elements: []process.ProcessInstanceElement{{
				ElementInstanceKey: "element-1",
				ElementId:          "task-a",
				ElementName:        "Task A",
				Type:               "SERVICE_TASK",
				State:              "ACTIVE",
				StartDate:          "2026-07-15T10:12:01Z",
				ProcessInstanceKey: "123",
			}},
			ShowIncidents: true,
		}},
	})

	require.NoError(t, err)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	require.Equal(t, "123", requireJSONObject(t, first["item"])["key"])
	require.Equal(t, "businessKey", requireJSONObject(t, requireJSONItems(t, first["variables"], 1)[0])["name"])
	require.Equal(t, "incident-123", requireJSONObject(t, requireJSONItems(t, first["incidents"], 1)[0])["incidentKey"])
	require.Equal(t, "element-1", requireJSONObject(t, requireJSONItems(t, first["elements"], 1)[0])["elementInstanceKey"])
}

func TestProcessInstanceActivityInstancesView_JSONIncludesElements(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
	})

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := elementEnrichedProcessInstancesView(cmd, process.ElementEnrichedProcessInstances{
		Total: 1,
		Items: []process.ElementEnrichedProcessInstance{{
			Item: process.ProcessInstance{
				Key:       "123",
				StartDate: "2026-07-15T10:12:00Z",
			},
			Elements: []process.ProcessInstanceElement{{
				ElementInstanceKey:     "element-1",
				ElementId:              "start",
				ElementName:            "Start",
				Type:                   "START_EVENT",
				State:                  "COMPLETED",
				StartDate:              "2026-07-15T10:12:01Z",
				EndDate:                "2026-07-15T10:12:02Z",
				ProcessInstanceKey:     "123",
				RootProcessInstanceKey: "123",
				ProcessDefinitionId:    "demo",
				ProcessDefinitionKey:   "9001",
				TenantId:               "tenant",
				HasIncident:            true,
				IncidentKey:            "incident-777",
			}},
		}},
	})

	require.NoError(t, err)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	elements := requireJSONItems(t, first["elements"], 1)
	element := requireJSONObject(t, elements[0])
	require.Equal(t, "element-1", element["elementInstanceKey"])
	require.Equal(t, "start", element["elementId"])
	require.Equal(t, "Start", element["elementName"])
	require.Equal(t, "START_EVENT", element["type"])
	require.Equal(t, "COMPLETED", element["state"])
	require.Equal(t, "2026-07-15T10:12:01Z", element["startDate"])
	require.Equal(t, "2026-07-15T10:12:02Z", element["endDate"])
	require.Equal(t, "123", element["processInstanceKey"])
	require.Equal(t, "123", element["rootProcessInstanceKey"])
	require.Equal(t, "demo", element["processDefinitionId"])
	require.Equal(t, "9001", element["processDefinitionKey"])
	require.Equal(t, "tenant", element["tenantId"])
	require.Equal(t, true, element["hasIncident"])
	require.Equal(t, "incident-777", element["incidentKey"])
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
		IncidentKey:        "incident-123",
		CreationTime:       "2026-05-06T09:29:42.711Z",
		ErrorMessage:       "No retries left",
		ElementId:          "task-a",
		ElementInstanceKey: "element-123",
		State:              "ACTIVE",
		ErrorType:          "JOB_NO_RETRIES",
		JobKey:             "job-123",
	})

	require.Equal(t, "incident-123 JOB_NO_RETRIES ACTIVE j:job-123 2026-05-06T09:29:42.711 (4 days ago) e:task-a ei:element-123 m:No retries left", got)
	require.NotContains(t, got, "incident incident-123:")
}

func TestIncidentHumanLine_RendersUnavailableJobKeyWhenMissing(t *testing.T) {
	prevLimit := flagGetPIIncidentMessageLimit
	flagGetPIIncidentMessageLimit = 0
	t.Cleanup(func() {
		flagGetPIIncidentMessageLimit = prevLimit
	})

	got := incidentHumanLine(incident.ProcessInstanceIncidentDetail{
		IncidentKey:        "incident-123",
		ErrorMessage:       "Mapping failed",
		ElementId:          "task-a",
		ElementInstanceKey: "element-123",
		State:              "ACTIVE",
		ErrorType:          "IO_MAPPING_ERROR",
	})

	require.Equal(t, "incident-123 IO_MAPPING_ERROR ACTIVE j:n/a e:task-a ei:element-123 m:Mapping failed", got)
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
