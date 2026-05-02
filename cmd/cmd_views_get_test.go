// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
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
