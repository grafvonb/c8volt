// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
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
		StartDate:      "2026-02-01T07:00:00.000Z",
		EndDate:        "2026-02-01T09:00:00.000Z",
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
		StartDate:      "2026-02-01T07:00:00.000Z",
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
				StartDate:      "2026-02-01T07:00:00.000Z",
			},
			{
				Key:               "22",
				TenantId:          "tenant",
				BpmnProcessId:     "MuchLongerProcess",
				ProcessVersion:    12,
				ProcessVersionTag: "stable",
				State:             process.StateCompleted,
				StartDate:         "2026-01-30T07:00:00.000Z",
				ParentKey:         "1",
				Incident:          true,
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, ""+
		"1  tenant Short             v1         ACTIVE    s:2026-02-01T07:00:00.000Z p:<root>      (today)\n"+
		"22 tenant MuchLongerProcess v12/stable COMPLETED s:2026-01-30T07:00:00.000Z p:1      inc! (2 days ago)\n"+
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
		StartDate:      "2026-02-01T07:00:00.000Z",
	}

	line := oneLinePI(base)
	require.NotContains(t, line, "inc!")
	require.NotContains(t, line, "i:false")

	base.Incident = true
	line = oneLinePI(base)
	require.Contains(t, line, " inc!")
	require.NotContains(t, line, "i:true")
}

func TestOneLinePI_RendersStartAndEndDateWithThreeDigitMilliseconds(t *testing.T) {
	line := oneLinePI(process.ProcessInstance{
		Key:            "2251799813758959",
		TenantId:       "<default>",
		BpmnProcessId:  "Process_18qgpch",
		ProcessVersion: 6,
		State:          process.StateActive,
		StartDate:      "2026-04-13T18:03:24.36Z",
		EndDate:        "2026-04-13T18:03:24Z",
	})

	require.Contains(t, line, "s:2026-04-13T18:03:24.360Z")
	require.Contains(t, line, "e:2026-04-13T18:03:24.000Z")
}

func TestProcessInstanceTimestampMillis_LeavesInvalidValuesUnchanged(t *testing.T) {
	require.Equal(t, "not-a-date", processInstanceTimestampMillis("not-a-date"))
	require.Equal(t, "", processInstanceTimestampMillis(""))
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
			Incidents: []process.ProcessInstanceIncidentDetail{{
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
				Incidents: []process.ProcessInstanceIncidentDetail{{
					IncidentKey:        "incident-123",
					ProcessInstanceKey: "123",
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
				Incidents: []process.ProcessInstanceIncidentDetail{{
					IncidentKey:        "incident-124",
					ProcessInstanceKey: "124",
					ErrorMessage:       "Second key failed",
				}},
			},
		},
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, output, "  incident incident-123: First key failed")
	require.Contains(t, output, "124 tenant demo-b v4 ACTIVE")
	require.Contains(t, output, "  incident incident-124: Second key failed")
	require.Contains(t, output, "found: 2")
	require.Less(t, strings.Index(output, "123 tenant demo-a"), strings.Index(output, "  incident incident-123"))
	require.Less(t, strings.Index(output, "  incident incident-123"), strings.Index(output, "124 tenant demo-b"))
	require.Less(t, strings.Index(output, "124 tenant demo-b"), strings.Index(output, "  incident incident-124"))
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
