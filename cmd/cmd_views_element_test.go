// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/stretchr/testify/require"
)

// TestElementFlatRowsFormatTimestampsAndIncidents verifies the compact element
// row grammar for active, completed, and incident-bearing runtime elements.
func TestElementFlatRowsFormatTimestampsAndIncidents(t *testing.T) {
	capturedNow := time.Date(2026, 7, 15, 10, 15, 1, 0, time.UTC)
	active := oneLineElementWithTimezone(element.Element{
		ElementInstanceKey:   "2251799813689002",
		ElementId:            "ship-order",
		Type:                 "SERVICE_TASK",
		State:                "ACTIVE",
		StartDate:            "2026-07-15T10:12:01Z",
		ProcessInstanceKey:   "2251799813688001",
		ProcessDefinitionKey: "2251799813687001",
		TenantId:             "tenant-a",
	}, false, capturedNow)
	require.Equal(t, "2251799813689002 tenant-a SERVICE_TASK ship-order ACTIVE s:2026-07-15T10:12:01.000 dur:3m0s pi:2251799813688001 pd:2251799813687001", active)
	require.NotContains(t, active, " e:")
	require.NotContains(t, active, "element:")
	require.NotContains(t, active, "inc!")

	completed := oneLineElementWithTimezone(element.Element{
		ElementInstanceKey:   "2251799813689003",
		ElementId:            "finish-order",
		Type:                 "END_EVENT",
		State:                "COMPLETED",
		StartDate:            "2026-07-15T10:12:01.21Z",
		EndDate:              "2026-07-15T10:12:02Z",
		ProcessInstanceKey:   "2251799813688001",
		ProcessDefinitionKey: "2251799813687001",
		TenantId:             "tenant-a",
		HasIncident:          true,
	}, false, capturedNow)
	require.Equal(t, "2251799813689003 tenant-a END_EVENT finish-order COMPLETED s:2026-07-15T10:12:01.210 e:2026-07-15T10:12:02.000 dur:790ms pi:2251799813688001 pd:2251799813687001 inc!", completed)
	require.Equal(t, 1, strings.Count(completed, "inc!"))

	incidentWithKey := oneLineElementWithTimezone(element.Element{
		ElementInstanceKey:   "2251799813689004",
		ElementId:            "charge-card",
		Type:                 "SERVICE_TASK",
		State:                "ACTIVE",
		StartDate:            "not-a-date",
		ProcessInstanceKey:   "2251799813688001",
		ProcessDefinitionKey: "2251799813687001",
		TenantId:             "tenant-a",
		HasIncident:          true,
		IncidentKey:          "2251799813687777",
	}, false, capturedNow)
	require.Contains(t, incidentWithKey, "s:not-a-date")
	require.NotContains(t, incidentWithKey, "dur:")
	require.Contains(t, incidentWithKey, "inc!:2251799813687777")
	require.Equal(t, 1, strings.Count(incidentWithKey, "inc!"))
	require.NotContains(t, incidentWithKey, " inc! inc!:")
	require.NotContains(t, incidentWithKey, "element:")
}

// TestRuntimeElementDurationOmitsUnprovenDurations verifies display-only duration calculation is conservative.
func TestRuntimeElementDurationOmitsUnprovenDurations(t *testing.T) {
	capturedNow := time.Date(2026, 7, 15, 10, 15, 1, 0, time.UTC)

	require.Equal(t, "3m0s", runtimeElementDuration("2026-07-15T10:12:01Z", "", "ACTIVE", capturedNow))
	require.Equal(t, "790ms", runtimeElementDuration("2026-07-15T10:12:01.21Z", "2026-07-15T10:12:02Z", "COMPLETED", capturedNow))
	require.Empty(t, runtimeElementDuration("2026-07-15T10:12:01Z", "", "COMPLETED", capturedNow))
	require.Empty(t, runtimeElementDuration("2026-07-15T10:12:02Z", "2026-07-15T10:12:01Z", "COMPLETED", capturedNow))
	require.Empty(t, runtimeElementDuration("not-a-date", "", "ACTIVE", capturedNow))
}

func TestElementFlatRowsAlignElementIDColumn(t *testing.T) {
	capturedNow := time.Date(2026, 7, 15, 10, 15, 1, 0, time.UTC)
	lines := formatFlatRows([]flatRow{
		flatRowElementWithTimezone(element.Element{
			ElementInstanceKey:   "1",
			ElementId:            "short",
			Type:                 "SERVICE_TASK",
			State:                "ACTIVE",
			StartDate:            "2026-07-15T10:12:01Z",
			ProcessInstanceKey:   "10",
			ProcessDefinitionKey: "20",
			TenantId:             "tenant-a",
		}, false, capturedNow),
		flatRowElementWithTimezone(element.Element{
			ElementInstanceKey:   "22",
			ElementId:            "SimpleUserTaskRequested_StartEvent",
			Type:                 "START_EVENT",
			State:                "COMPLETED",
			StartDate:            "2026-07-15T10:12:02Z",
			EndDate:              "2026-07-15T10:12:03Z",
			ProcessInstanceKey:   "100",
			ProcessDefinitionKey: "200",
			TenantId:             "<default>",
		}, false, capturedNow),
	})

	require.Len(t, lines, 2)
	require.Equal(t, strings.Index(lines[0], "short"), strings.Index(lines[1], "SimpleUserTaskRequested_StartEvent"))
	require.Equal(t, strings.Index(lines[0], "ACTIVE"), strings.Index(lines[1], "COMPLETED"))
	require.Equal(t, strings.Index(lines[0], "s:"), strings.Index(lines[1], "s:"))
	require.NotContains(t, lines[0], "element:")
	require.NotContains(t, lines[1], "element:")
}
