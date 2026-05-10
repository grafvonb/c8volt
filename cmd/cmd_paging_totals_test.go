// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

// TestFormatProcessInstancePagingProgress verifies process-instance
// continuation prompts distinguish exact totals, lower-bound totals, and totals
// invalidated by command-local filtering.
func TestFormatProcessInstancePagingProgress(t *testing.T) {
	t.Run("exact total", func(t *testing.T) {
		resetProcessInstanceCommandGlobals()
		t.Cleanup(resetProcessInstanceCommandGlobals)

		page := process.ProcessInstancePage{
			ReportedTotal: &process.ProcessInstanceReportedTotal{
				Count: 5323,
				Kind:  process.ProcessInstanceReportedTotalKindExact,
			},
		}

		require.Equal(t, "1000/5323 loaded", formatProcessInstancePagingProgress(page, 1000, "loaded"))
	})

	t.Run("lower-bound total", func(t *testing.T) {
		resetProcessInstanceCommandGlobals()
		t.Cleanup(resetProcessInstanceCommandGlobals)

		page := process.ProcessInstancePage{
			ReportedTotal: &process.ProcessInstanceReportedTotal{
				Count: 5323,
				Kind:  process.ProcessInstanceReportedTotalKindLowerBound,
			},
		}

		require.Equal(t, "1000/5323+ requested", formatProcessInstancePagingProgress(page, 1000, "requested"))
	})

	t.Run("local filters hide backend total", func(t *testing.T) {
		resetProcessInstanceCommandGlobals()
		t.Cleanup(resetProcessInstanceCommandGlobals)
		flagGetPIChildrenOnly = true
		page := process.ProcessInstancePage{
			ReportedTotal: &process.ProcessInstanceReportedTotal{
				Count: 5323,
				Kind:  process.ProcessInstanceReportedTotalKindExact,
			},
		}

		require.Equal(t, "1000 loaded", formatProcessInstancePagingProgress(page, 1000, "loaded"))
	})
}

// TestFormatIncidentPagingProgress verifies incident continuation prompts share
// the same exact, lower-bound, and unavailable total wording as process-instance
// prompts.
func TestFormatIncidentPagingProgress(t *testing.T) {
	t.Run("exact total", func(t *testing.T) {
		page := incident.Page{
			ReportedTotal: &incident.ReportedTotal{
				Count: 5323,
				Kind:  incident.ReportedTotalKindExact,
			},
		}

		require.Equal(t, "1000/5323 loaded", formatIncidentPagingProgress(page, 1000, "loaded"))
	})

	t.Run("lower-bound total", func(t *testing.T) {
		page := incident.Page{
			ReportedTotal: &incident.ReportedTotal{
				Count: 5323,
				Kind:  incident.ReportedTotalKindLowerBound,
			},
		}

		require.Equal(t, "1000/5323+ loaded", formatIncidentPagingProgress(page, 1000, "loaded"))
	})

	t.Run("no reported total", func(t *testing.T) {
		require.Equal(t, "1000 loaded", formatIncidentPagingProgress(incident.Page{}, 1000, "loaded"))
	})
}
