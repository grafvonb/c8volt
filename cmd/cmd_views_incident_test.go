// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/stretchr/testify/require"
)

// TestIncidentHumanLineWithMessageLimit_RendersAlignedIncidentListFieldsAndAge
// protects plain incident row alignment, field order, message truncation, and age tags.
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
			ElementId:              "task-a",
			ElementInstanceKey:     "2251799813685300",
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
	require.Contains(t, lines[0], "j:n/a 2026-05-05T10:15:00.000 (4 days ago) demo-process pi:2251799813711967 root:2251799813711960")
	require.Contains(t, lines[0], "e:task-a ei:2251799813685300 m:No retries left...")
	require.NotContains(t, lines[0], "fn:")
	require.NotContains(t, lines[0], "fni:")
	require.Contains(t, lines[1], "9                <default> IO_MAPPING_ERROR RESOLVED j:n/a")
	require.Contains(t, lines[1], "j:n/a 2026-05-08T10:15:00.000 (1 days ago) tiny-demo    pi:1")
	require.Contains(t, lines[1], "m:short")
	require.Less(t, strings.Index(lines[0], "ACTIVE"), strings.Index(lines[0], "j:n/a"))
	require.Less(t, strings.Index(lines[0], "j:n/a"), strings.Index(lines[0], "2026-05-05T10:15:00.000"))
	require.Less(t, strings.Index(lines[0], "2026-05-05T10:15:00.000"), strings.Index(lines[0], "demo-process"))
	require.Less(t, strings.Index(lines[0], "demo-process"), strings.Index(lines[0], "pi:2251799813711967"))
	require.Less(t, strings.Index(lines[0], "pi:2251799813711967"), strings.Index(lines[0], "root:2251799813711960"))
	require.Less(t, strings.Index(lines[0], "root:2251799813711960"), strings.Index(lines[0], "e:task-a"))
	require.Less(t, strings.Index(lines[0], "e:task-a"), strings.Index(lines[0], "ei:2251799813685300"))
	require.Less(t, strings.Index(lines[0], "ei:2251799813685300"), strings.Index(lines[0], "m:No"))
	require.Equal(t, strings.Index(lines[0], "2026-05-05T10:15:00.000"), strings.Index(lines[1], "2026-05-08T10:15:00.000"))
	require.NotContains(t, lines[0], "2251799813685200")
	require.NotContains(t, lines[0], "err:")
	require.Contains(t, lines[0], "m:No")
	require.NotContains(t, lines[0], "m: ")
}

// TestIncidentHumanLineWithMessageLimit_SkipsAgeForMissingOrInvalidCreationTime
// keeps invalid incident creation times visible without adding misleading age tags.
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

// TestTruncateIncidentHumanMessage preserves the incident message truncation
// contract, including unlimited, exact-length, truncated, and rune-aware cases.
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

// TestListIncidentsView_HumanJSONAndKeysOnly keeps plain incident collection
// rendering compatible for terminal output and script-oriented modes.
func TestListIncidentsView_HumanJSONAndKeysOnly(t *testing.T) {
	resp := incidentRendererFixture()

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("incident")

		require.NoError(t, listIncidentsView(cmd, resp, 10, false))
		output := cmd.OutOrStdout().(*bytes.Buffer).String()

		require.Contains(t, output, "incident-123")
		require.Contains(t, output, "2026-05-06T09:29:42.711")
		require.Contains(t, output, "e:task-a")
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

// TestListIncidentsView_PIKeysOnlySkipsMissingProcessInstanceKeys verifies the
// incident-specific process-instance-key mode stays one key per line and omits blanks.
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

// TestRenderIncidentProcessInstanceKeys_PreservesDuplicatesAndSkipsMissing
// keeps process-instance-key rendering faithful to incident result order.
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

// incidentRendererFixture keeps incident renderer tests on one representative
// payload that exercises message, job, timestamp, element, and fallback fields.
func incidentRendererFixture() incident.Incidents {
	return incident.Incidents{
		Total: 2,
		Items: []incident.ProcessInstanceIncidentDetail{
			{
				IncidentKey:        "incident-123",
				CreationTime:       "2026-05-06T09:29:42.711Z",
				ProcessInstanceKey: "pi-123",
				TenantId:           "tenant-a",
				State:              "ACTIVE",
				ErrorType:          "JOB_NO_RETRIES",
				ErrorMessage:       "No retries left",
				ElementId:          "task-a",
				ElementInstanceKey: "element-123",
				JobKey:             "job-123",
			},
			{
				IncidentKey:  "incident-124",
				State:        "RESOLVED",
				ErrorMessage: "Mapping failed",
			},
		},
	}
}
