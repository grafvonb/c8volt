// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// slowProcessAnalysisFixtureTime centralizes timestamp parsing for service analysis fixtures.
func slowProcessAnalysisFixtureTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	require.NoError(t, err)
	return parsed
}

// slowProcessAnalysisFixtureProcessInstance returns a minimal process-instance root for timing tests.
func slowProcessAnalysisFixtureProcessInstance(key string, start time.Time, end time.Time) d.ProcessInstance {
	return d.ProcessInstance{
		Key:                    key,
		RootProcessInstanceKey: key,
		ProcessDefinitionKey:   "2251799813687001",
		BpmnProcessId:          "OrderProcess",
		ProcessVersion:         7,
		State:                  d.StateCompleted,
		StartDate:              start.Format(time.RFC3339Nano),
		EndDate:                end.Format(time.RFC3339Nano),
		TenantId:               "tenant-a",
	}
}

// slowProcessAnalysisFixtureElement returns a runtime element row tied to a process instance.
func slowProcessAnalysisFixtureElement(processInstanceKey string, elementInstanceKey string, elementID string, start time.Time, end time.Time) d.Element {
	return d.Element{
		ElementInstanceKey:     elementInstanceKey,
		ElementId:              elementID,
		Type:                   "SERVICE_TASK",
		State:                  "COMPLETED",
		StartDate:              start.Format(time.RFC3339Nano),
		EndDate:                end.Format(time.RFC3339Nano),
		ProcessInstanceKey:     processInstanceKey,
		RootProcessInstanceKey: processInstanceKey,
		ProcessDefinitionKey:   "2251799813687001",
		TenantId:               "tenant-a",
	}
}

// TestSlowProcessAnalysisFixturesBuildConsistentTimingRows protects the shared fixture assumptions.
func TestSlowProcessAnalysisFixturesBuildConsistentTimingRows(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	end := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:05:00Z")

	pi := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, end)
	element := slowProcessAnalysisFixtureElement(pi.Key, "2251799813685250", "ReserveStock", start.Add(time.Second), start.Add(5*time.Second))

	require.Equal(t, pi.Key, pi.RootProcessInstanceKey)
	require.Equal(t, pi.Key, element.ProcessInstanceKey)
	require.Equal(t, pi.ProcessDefinitionKey, element.ProcessDefinitionKey)
	require.Equal(t, "ReserveStock", element.ElementId)
}
