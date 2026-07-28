// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
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

func TestFormatPITotalActivityProgress(t *testing.T) {
	require.Equal(t,
		"counting process instances: 16668 counted, fetching next page",
		formatPITotalActivityProgress(processInstanceProgressSummary{
			CumulativeCount:   16668,
			ContinuationState: processInstanceContinuationAutoContinue,
		}),
	)
	require.Equal(t,
		"counting process instances: 16668 counted, complete",
		formatPITotalActivityProgress(processInstanceProgressSummary{
			CumulativeCount:   16668,
			ContinuationState: processInstanceContinuationCompleted,
		}),
	)
}

// TestSearchProcessInstancesTotalUsesWorkflowActivity verifies fallback page-counting progress outranks nested lookups.
func TestSearchProcessInstancesTotalUsesWorkflowActivity(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	cli := stubProcessAPI{
		searchProcessInstancesTotal: func(_ context.Context, _ process.ProcessInstanceSearchRequest, visitor process.ProcessInstanceSearchTotalVisitor, _ ...options.FacadeOption) (int64, error) {
			err := visitor(process.ProcessInstanceSearchTotalStep{
				Page: process.ProcessInstancePage{
					Items:         []process.ProcessInstance{{Key: "1"}, {Key: "2"}},
					OverflowState: process.ProcessInstanceOverflowStateNoMore,
				},
				TotalAfter: 2,
			})
			return 2, err
		},
	}

	got, err := searchProcessInstancesTotal(cmd, nil, cli, nil, process.ProcessInstanceFilter{})

	require.NoError(t, err)
	require.EqualValues(t, 2, got)
	require.Equal(t, []activitysink.Start{{
		Message:    "counting process instances page by page",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.Starts())
	require.Equal(t, []activitysink.Update{{
		Message:    "counting process instances: 2 counted, complete",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}
