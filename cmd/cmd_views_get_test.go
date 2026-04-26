// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
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

func TestOneLinePI_WithAge(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	prevWithAge := flagGetPIWithAge
	flagGetPIWithAge = true
	t.Cleanup(func() {
		relativeDayNow = prevNow
		flagGetPIWithAge = prevWithAge
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

func TestOneLinePI_WithAgeToday(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	}
	prevWithAge := flagGetPIWithAge
	flagGetPIWithAge = true
	t.Cleanup(func() {
		relativeDayNow = prevNow
		flagGetPIWithAge = prevWithAge
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
