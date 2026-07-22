// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestRenderOpsSlowProcessAnalysisResultHumanRendersKeyedRoots verifies root rows include durations and the final count.
func TestRenderOpsSlowProcessAnalysisResultHumanRendersKeyedRoots(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	result := ops.SlowProcessAnalysisResult{
		Items: []ops.SlowProcessAnalysisProcessInstance{
			{
				Key:                    "2251799813685249",
				TenantID:               "tenant-a",
				BpmnProcessID:          "OrderProcess",
				ProcessDefinitionKey:   "2251799813687001",
				ProcessVersion:         7,
				State:                  process.StateCompleted,
				StartDate:              "2026-07-18T10:00:00Z",
				EndDate:                "2026-07-18T10:05:00Z",
				RootProcessInstanceKey: "2251799813685249",
				Duration:               "5m0s",
				DurationMillis:         300000,
				DurationAvailable:      true,
			},
			{
				Key:                    "2251799813685250",
				TenantID:               "tenant-b",
				BpmnProcessID:          "OrderProcess",
				ProcessDefinitionKey:   "2251799813687001",
				ProcessVersion:         7,
				State:                  process.StateCompleted,
				StartDate:              "2026-07-18T10:00:00Z",
				RootProcessInstanceKey: "2251799813685250",
				DurationAvailable:      false,
			},
		},
		Count: 2,
	}

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "2251799813685249")
	require.Contains(t, output, "tenant-a")
	require.Contains(t, output, "OrderProcess")
	require.Contains(t, output, "dur:5m0s")
	require.Contains(t, output, "2251799813685250")
	require.Contains(t, output, "dur:-")
	require.True(t, strings.HasSuffix(output, "process instances: 2\n"))
}

// TestRenderOpsSlowProcessAnalysisResultHumanRendersDurationShareBars verifies visual indicators show duration shares without labels.
func TestRenderOpsSlowProcessAnalysisResultHumanRendersDurationShareBars(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	result := ops.SlowProcessAnalysisResult{
		Items: []ops.SlowProcessAnalysisProcessInstance{
			{
				Key:                    "root-100",
				ProcessDefinitionKey:   "2251799813687001",
				State:                  process.StateCompleted,
				StartDate:              "2026-07-18T10:00:00Z",
				EndDate:                "2026-07-18T10:01:40Z",
				RootProcessInstanceKey: "root-100",
				Duration:               "1m40s",
				DurationMillis:         100000,
				DurationAvailable:      true,
				RelativePercentile:     17,
				RelativeBar:            "[##--------]",
			},
			{
				Key:                    "root-50",
				ProcessDefinitionKey:   "2251799813687001",
				State:                  process.StateCompleted,
				StartDate:              "2026-07-18T10:00:00Z",
				EndDate:                "2026-07-18T10:00:50Z",
				RootProcessInstanceKey: "root-50",
				Duration:               "50s",
				DurationMillis:         50000,
				DurationAvailable:      true,
				RelativePercentile:     50,
				RelativeBar:            "[#####-----]",
			},
			{
				Key:                    "root-10",
				ProcessDefinitionKey:   "2251799813687001",
				State:                  process.StateCompleted,
				StartDate:              "2026-07-18T10:00:00Z",
				EndDate:                "2026-07-18T10:00:10Z",
				RootProcessInstanceKey: "root-10",
				Duration:               "10s",
				DurationMillis:         10000,
				DurationAvailable:      true,
				RelativePercentile:     83,
				RelativeBar:            "[########--]",
				Timeline: []ops.SlowProcessAnalysisTimelineEntry{
					{
						Kind:               ops.SlowProcessAnalysisTimelineEntryKindElement,
						ElementInstanceKey: "2251799813685250",
						ElementID:          "ReserveStock",
						Type:               "SERVICE_TASK",
						State:              "COMPLETED",
						StartDate:          "2026-07-18T10:00:00Z",
						EndDate:            "2026-07-18T10:00:00Z",
						Duration:           "0s",
						DurationAvailable:  true,
						RelativePercentile: 50,
						RelativeBar:        "[#####-----]",
					},
					{
						Kind:                 ops.SlowProcessAnalysisTimelineEntryKindElement,
						ElementInstanceKey:   "2251799813685251",
						ElementID:            "Work",
						Type:                 "SERVICE_TASK",
						State:                "COMPLETED",
						StartDate:            "2026-07-18T10:00:00Z",
						EndDate:              "2026-07-18T10:00:05Z",
						Duration:             "5s",
						DurationMillis:       5000,
						DurationAvailable:    true,
						ProcessDurationShare: 50,
						RelativePercentile:   83,
						RelativeBar:          "[########--]",
					},
				},
			},
		},
		Count: 3,
	}

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "root-100")
	require.Contains(t, output, "dur:1m40s [██████████] 100%")
	require.Contains(t, output, "root-50")
	require.Contains(t, output, "dur:50s [█████░░░░░] 50%")
	require.Contains(t, output, "root-10")
	require.Contains(t, output, "dur:10s [█░░░░░░░░░] 10%")
	require.Contains(t, output, "└─ slowest elements:\n")
	require.NotContains(t, output, "ReserveStock")
	require.NotContains(t, output, "dur:0s [")
	require.Contains(t, output, "   ├─ SERVICE_TASK Work COMPLETED s:10:00:00.000 e:10:00:05.000 dur:5s [█████░░░░░] 50%")
	require.Contains(t, output, "   └─ hidden: 1 instant/fast timeline row; use --with-full-timeline")
	require.NotContains(t, output, "cohort")
	require.NotContains(t, output, "peer")
	require.NotContains(t, output, "compared")
	require.NotContains(t, output, "rank")
	require.NotContains(t, output, "share")
}

// TestRenderOpsSlowProcessAnalysisResultHumanRendersSubPercentDurationShare verifies issue #244's sub-percent shape.
func TestRenderOpsSlowProcessAnalysisResultHumanRendersSubPercentProcessShare(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	result := ops.SlowProcessAnalysisResult{
		Items: []ops.SlowProcessAnalysisProcessInstance{{
			Key:                    "2251799813694100",
			TenantID:               "tenant-a",
			BpmnProcessID:          "OrderProcess",
			ProcessDefinitionKey:   "2251799813687001",
			ProcessVersion:         7,
			State:                  process.StateCompleted,
			StartDate:              "2026-07-18T08:10:00Z",
			EndDate:                "2026-07-18T08:24:32Z",
			RootProcessInstanceKey: "2251799813694100",
			Duration:               "14m32s",
			DurationMillis:         872000,
			DurationAvailable:      true,
			RelativePercentile:     93,
			ComparisonSampleCount:  12,
			RelativeBar:            "[#########-]",
			Timeline: []ops.SlowProcessAnalysisTimelineEntry{
				{
					Kind:                  ops.SlowProcessAnalysisTimelineEntryKindElement,
					ElementInstanceKey:    "4108",
					ElementID:             "ReserveStock",
					Type:                  "SERVICE_TASK",
					State:                 "COMPLETED",
					StartDate:             "2026-07-18T08:10:00.300Z",
					EndDate:               "2026-07-18T08:10:04.500Z",
					Duration:              "4.2s",
					DurationMillis:        4200,
					DurationAvailable:     true,
					RelativePercentile:    64,
					ComparisonSampleCount: 8,
					RelativeBar:           "[######----]",
				},
				{
					Kind:                   ops.SlowProcessAnalysisTimelineEntryKindTransition,
					FromElementID:          "ReserveStock",
					FromElementType:        "SERVICE_TASK",
					FromElementInstanceKey: "4108",
					ToElementID:            "OrderFinished",
					ToElementType:          "END_EVENT",
					ToElementInstanceKey:   "4122",
					Duration:               "14m20s",
					DurationMillis:         860000,
					DurationAvailable:      true,
					RelativePercentile:     99,
					ComparisonSampleCount:  7,
					RelativeBar:            "[##########]",
				},
				{
					Kind:                  ops.SlowProcessAnalysisTimelineEntryKindElement,
					ElementInstanceKey:    "4122",
					ElementID:             "OrderFinished",
					Type:                  "END_EVENT",
					State:                 "COMPLETED",
					StartDate:             "2026-07-18T08:24:24.500Z",
					EndDate:               "2026-07-18T08:24:24.508Z",
					Duration:              "8ms",
					DurationMillis:        8,
					DurationAvailable:     true,
					RelativePercentile:    31,
					ComparisonSampleCount: 8,
					RelativeBar:           "[###-------]",
				},
			},
		}},
		Count: 1,
	}

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	require.Contains(t, buf.String(), "2251799813694100 tenant-a OrderProcess v7 COMPLETED s:2026-07-18T08:10:00.000 e:2026-07-18T08:24:32.000 p:<root>")
	require.Contains(t, buf.String(), "dur:14m32s")
	require.NotContains(t, buf.String(), "dur:14m32s [")
	require.Contains(t, buf.String(), "└─ slowest elements:\n   └─ hidden: 3 instant/fast timeline rows; use --with-full-timeline")
	require.NotContains(t, buf.String(), "4108 SERVICE_TASK ReserveStock")
	require.NotContains(t, buf.String(), "ReserveStock -> OrderFinished")
	require.NotContains(t, buf.String(), "4122 END_EVENT OrderFinished")
	require.NotContains(t, buf.String(), "PI:")
}

// TestRenderOpsSlowProcessAnalysisResultHumanRendersHotspotSummaryDetails verifies default detail rows stay compact and omit noisy transitions.
func TestRenderOpsSlowProcessAnalysisResultHumanRendersHotspotSummaryDetails(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	result := ops.SlowProcessAnalysisResult{
		Items: []ops.SlowProcessAnalysisProcessInstance{
			{
				Key:                    "2251799813685249",
				TenantID:               "tenant-a",
				BpmnProcessID:          "OrderProcess",
				ProcessDefinitionKey:   "2251799813687001",
				ProcessVersion:         7,
				State:                  process.StateCompleted,
				StartDate:              "2026-07-18T10:00:00Z",
				EndDate:                "2026-07-18T10:05:00Z",
				RootProcessInstanceKey: "2251799813685249",
				Duration:               "5m0s",
				DurationMillis:         300000,
				DurationAvailable:      true,
				Timeline: []ops.SlowProcessAnalysisTimelineEntry{
					{
						Kind:                 ops.SlowProcessAnalysisTimelineEntryKindElement,
						ElementInstanceKey:   "2251799813685250",
						ElementID:            "ReserveStock",
						Type:                 "SERVICE_TASK",
						State:                "COMPLETED",
						StartDate:            "2026-07-18T10:00:00Z",
						EndDate:              "2026-07-18T10:00:04Z",
						Duration:             "4s",
						DurationMillis:       4000,
						DurationAvailable:    true,
						ProcessDurationShare: 1,
						HasIncident:          true,
						IncidentKey:          "2251799813687777",
					},
					{
						Kind:                   ops.SlowProcessAnalysisTimelineEntryKindTransition,
						FromElementID:          "ReserveStock",
						FromElementType:        "SERVICE_TASK",
						FromElementInstanceKey: "2251799813685250",
						ToElementID:            "OrderFinished",
						ToElementType:          "END_EVENT",
						ToElementInstanceKey:   "2251799813685251",
						Duration:               "4m56s",
						DurationMillis:         296000,
						DurationAvailable:      true,
						ProcessDurationShare:   99,
					},
				},
			},
		},
		Count: 1,
	}

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "└─ slowest elements:\n")
	require.Contains(t, output, "   ├─ SERVICE_TASK ReserveStock COMPLETED s:10:00:00.000 e:10:00:04.000 dur:4s [░░░░░░░░░░] 1% inc!:2251799813687777")
	require.Contains(t, output, "   └─ hidden: 1 instant/fast timeline row; use --with-full-timeline")
	require.NotContains(t, output, "2251799813685250 SERVICE_TASK")
	require.NotContains(t, output, "ReserveStock -> OrderFinished")
	require.NotContains(t, output, "between:")
	require.NotContains(t, output, "transition:")
	require.NotContains(t, output, "PI:")
	require.True(t, strings.HasSuffix(output, "process instances: 1\n"))
}

// TestRenderOpsSlowProcessAnalysisResultKeysOnlyRendersRootKeys verifies keyed output remains pipeline-safe.
func TestRenderOpsSlowProcessAnalysisResultKeysOnlyRendersUniqueRootKeys(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	flagViewKeysOnly = true
	t.Cleanup(func() { flagViewKeysOnly = false })
	result := ops.SlowProcessAnalysisResult{
		Items: []ops.SlowProcessAnalysisProcessInstance{
			{Key: "2251799813685249"},
			{Key: "2251799813685250"},
			{Key: "2251799813685249"},
		},
		Count: 2,
	}

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	require.Equal(t, "2251799813685249\n2251799813685250\n", buf.String())
}

// TestRenderOpsSlowProcessAnalysisResultJSONIncludesStableAnalysisFields verifies automation-visible payload fields.
func TestRenderOpsSlowProcessAnalysisResultJSONIncludesStableAnalysisFields(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	flagViewAsJson = true
	t.Cleanup(func() { flagViewAsJson = false })
	captured := time.Date(2026, 7, 18, 10, 30, 0, 0, time.UTC)
	result := ops.SlowProcessAnalysisResult{
		Request: ops.SlowProcessAnalysisRequest{
			CommandName:   "ops analyse slow-process-instances",
			SelectionMode: ops.SlowProcessAnalysisSelectionModeExplicitKeys,
			InputKeys:     typex.Keys{"2251799813685249"},
			OutputMode:    "json",
		},
		CapturedAt: captured,
		Items: []ops.SlowProcessAnalysisProcessInstance{{
			Key:                   "2251799813685249",
			TenantID:              "tenant-a",
			BpmnProcessID:         "OrderProcess",
			ProcessDefinitionKey:  "2251799813687001",
			ProcessVersion:        7,
			State:                 process.StateCompleted,
			StartDate:             "2026-07-18T10:00:00Z",
			EndDate:               "2026-07-18T10:05:00Z",
			Duration:              "5m0s",
			DurationMillis:        300000,
			DurationAvailable:     true,
			RelativePercentile:    88,
			ComparisonSampleCount: 4,
			RelativeBar:           "[#########-]",
			Timeline: []ops.SlowProcessAnalysisTimelineEntry{
				{
					Kind:                  ops.SlowProcessAnalysisTimelineEntryKindElement,
					ElementInstanceKey:    "2251799813685250",
					ElementID:             "ReserveStock",
					Type:                  "SERVICE_TASK",
					State:                 "COMPLETED",
					StartDate:             "2026-07-18T10:00:00Z",
					EndDate:               "2026-07-18T10:00:04Z",
					Duration:              "4s",
					DurationMillis:        4000,
					DurationAvailable:     true,
					RelativePercentile:    50,
					ComparisonSampleCount: 4,
					RelativeBar:           "[#####-----]",
					ProcessDurationShare:  1,
				},
				{
					Kind:                   ops.SlowProcessAnalysisTimelineEntryKindTransition,
					FromElementInstanceKey: "2251799813685250",
					FromElementID:          "ReserveStock",
					FromElementType:        "SERVICE_TASK",
					FromEndDate:            "2026-07-18T10:00:04Z",
					ToElementInstanceKey:   "2251799813685251",
					ToElementID:            "OrderFinished",
					ToElementType:          "END_EVENT",
					ToStartDate:            "2026-07-18T10:04:00Z",
					Duration:               "3m56s",
					DurationMillis:         236000,
					DurationAvailable:      true,
					RelativePercentile:     83,
					ComparisonSampleCount:  3,
					RelativeBar:            "[########--]",
					ProcessDurationShare:   79,
				},
			},
		}},
		Count: 1,
		Empty: false,
	}

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	var payload ops.SlowProcessAnalysisResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload))
	require.Equal(t, captured, payload.CapturedAt)
	require.Equal(t, "ops analyse slow-process-instances", payload.Request.CommandName)
	require.Equal(t, "2251799813685249", payload.Items[0].Key)
	require.Equal(t, int64(300000), payload.Items[0].DurationMillis)
	require.Equal(t, 88, payload.Items[0].RelativePercentile)
	require.Equal(t, 4, payload.Items[0].ComparisonSampleCount)
	require.Equal(t, "[#########-]", payload.Items[0].RelativeBar)
	require.Equal(t, ops.SlowProcessAnalysisTimelineEntryKindElement, payload.Items[0].Timeline[0].Kind)
	require.Equal(t, int64(4000), payload.Items[0].Timeline[0].DurationMillis)
	require.Equal(t, 1, payload.Items[0].Timeline[0].ProcessDurationShare)
	require.Equal(t, ops.SlowProcessAnalysisTimelineEntryKindTransition, payload.Items[0].Timeline[1].Kind)
	require.Equal(t, "ReserveStock", payload.Items[0].Timeline[1].FromElementID)
	require.Equal(t, "OrderFinished", payload.Items[0].Timeline[1].ToElementID)
	require.Equal(t, int64(236000), payload.Items[0].Timeline[1].DurationMillis)
	require.Equal(t, 83, payload.Items[0].Timeline[1].RelativePercentile)
	require.Contains(t, buf.String(), `"capturedAt"`)
	require.Contains(t, buf.String(), `"comparisonSampleCount"`)
	require.Contains(t, buf.String(), `"processDurationShare"`)
}

// TestRenderOpsSlowProcessAnalysisResultEmptySearchOutputs verifies all output modes represent no-match discovery cleanly.
func TestRenderOpsSlowProcessAnalysisResultEmptySearchOutputs(t *testing.T) {
	t.Run("human count only", func(t *testing.T) {
		cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()

		err := renderOpsSlowProcessAnalysisResult(cmd, ops.SlowProcessAnalysisResult{Items: []ops.SlowProcessAnalysisProcessInstance{}, Count: 0, Empty: true})

		require.NoError(t, err)
		require.Equal(t, "process instances: 0\n", buf.String())
	})

	t.Run("json empty items", func(t *testing.T) {
		cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
		flagViewAsJson = true
		t.Cleanup(func() { flagViewAsJson = false })

		err := renderOpsSlowProcessAnalysisResult(cmd, ops.SlowProcessAnalysisResult{Items: []ops.SlowProcessAnalysisProcessInstance{}, Count: 0, Empty: true})

		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &payload))
		require.Equal(t, float64(0), payload["count"])
		require.Equal(t, true, payload["empty"])
		require.Equal(t, []any{}, payload["items"])
	})

	t.Run("keys-only silence", func(t *testing.T) {
		cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
		flagViewKeysOnly = true
		t.Cleanup(func() { flagViewKeysOnly = false })

		err := renderOpsSlowProcessAnalysisResult(cmd, ops.SlowProcessAnalysisResult{Items: []ops.SlowProcessAnalysisProcessInstance{}, Count: 0, Empty: true})

		require.NoError(t, err)
		require.Empty(t, buf.String())
	})
}

// TestOpsSlowProcessAnalysisDefaultHotspotSummarySelectsCompletedContributors verifies the default summary picks only actionable completed rows.
func TestOpsSlowProcessAnalysisDefaultHotspotSummarySelectsCompletedContributors(t *testing.T) {
	result := opsSlowProcessAnalysisRenderTestResult(
		opsSlowProcessAnalysisRenderTestElement("4109", "PackOrder", "SERVICE_TASK", "COMPLETED", "2026-07-18T10:00:04Z", "2026-07-18T10:00:12Z", "8s", 8000, 3),
		opsSlowProcessAnalysisRenderTestElement("4108", "ReserveStock", "SERVICE_TASK", "COMPLETED", "2026-07-18T10:00:00Z", "2026-07-18T10:00:04Z", "4s", 4000, 1),
		opsSlowProcessAnalysisRenderTestElement("4110", "AuditTrail", "SERVICE_TASK", "COMPLETED", "2026-07-18T10:00:12Z", "2026-07-18T10:00:13Z", "1s", 1000, 0),
		opsSlowProcessAnalysisRenderTestTransition("ReserveStock", "SERVICE_TASK", "OrderFinished", "END_EVENT", "2026-07-18T10:00:04Z", "2026-07-18T10:05:00Z", "4m56s", 296000, 99),
	)

	summary := opsSlowProcessAnalysisDefaultHotspotSummary(result.Items[0])

	require.Len(t, summary.Rows, 2)
	require.Equal(t, "PackOrder", summary.Rows[0].ElementID)
	require.Equal(t, "ReserveStock", summary.Rows[1].ElementID)
	require.Equal(t, 2, summary.HiddenRowCount)
	require.Equal(t, opsSlowProcessAnalysisHotspotMinimumProcessShare, 1)
	summary.Rows[0].ElementID = "changed"
	require.Equal(t, "PackOrder", result.Items[0].Timeline[0].ElementID)
}

// TestOpsSlowProcessAnalysisDefaultHotspotSummarySelectsActiveAndIncidentRows verifies operational rows are never hidden by duration threshold.
func TestOpsSlowProcessAnalysisDefaultHotspotSummarySelectsActiveAndIncidentRows(t *testing.T) {
	active := opsSlowProcessAnalysisRenderTestElement("4110", "WaitForCallback", "RECEIVE_TASK", "ACTIVE", "2026-07-18T10:04:59Z", "", "", 0, 0)
	incident := opsSlowProcessAnalysisRenderTestElement("4111", "NotifyCustomer", "SERVICE_TASK", "COMPLETED", "2026-07-18T10:00:12Z", "2026-07-18T10:00:13Z", "1s", 1000, 0)
	incident.HasIncident = true
	incident.IncidentKey = "2251799813687777"
	result := opsSlowProcessAnalysisRenderTestResult(
		active,
		incident,
		opsSlowProcessAnalysisRenderTestElement("4112", "FastAudit", "SERVICE_TASK", "COMPLETED", "2026-07-18T10:00:13Z", "2026-07-18T10:00:13Z", "0s", 0, 0),
	)

	summary := opsSlowProcessAnalysisDefaultHotspotSummary(result.Items[0])

	require.Len(t, summary.Rows, 2)
	require.ElementsMatch(t, []string{"WaitForCallback", "NotifyCustomer"}, []string{summary.Rows[0].ElementID, summary.Rows[1].ElementID})
	require.Equal(t, 1, summary.HiddenRowCount)
}

// TestRenderOpsSlowProcessAnalysisResultHumanIncludesActiveAndIncidentRows verifies default output shows active and incident rows once.
func TestRenderOpsSlowProcessAnalysisResultHumanIncludesActiveAndIncidentRows(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	active := opsSlowProcessAnalysisRenderTestElement("4110", "WaitForCallback", "RECEIVE_TASK", "ACTIVE", "2026-07-18T10:04:59Z", "", "", 0, 0)
	incident := opsSlowProcessAnalysisRenderTestElement("4111", "NotifyCustomer", "SERVICE_TASK", "COMPLETED", "2026-07-18T10:00:12Z", "2026-07-18T10:00:13Z", "1s", 1000, 0)
	incident.HasIncident = true
	incident.IncidentKey = "2251799813687777"
	alsoSlowIncident := opsSlowProcessAnalysisRenderTestElement("4112", "ReviewOrder", "USER_TASK", "COMPLETED", "2026-07-18T10:00:13Z", "2026-07-18T10:01:13Z", "1m0s", 60000, 20)
	alsoSlowIncident.HasIncident = true
	alsoSlowIncident.IncidentKey = "2251799813687778"
	result := opsSlowProcessAnalysisRenderTestResult(active, incident, alsoSlowIncident)

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "└─ slowest elements:\n")
	require.Contains(t, output, "USER_TASK ReviewOrder COMPLETED s:10:00:13.000 e:10:01:13.000 dur:1m0s [██░░░░░░░░] 20% inc!:2251799813687778")
	require.Contains(t, output, "SERVICE_TASK NotifyCustomer COMPLETED s:10:00:12.000 e:10:00:13.000 dur:1s [░░░░░░░░░░] <1% inc!:2251799813687777")
	require.Contains(t, output, "RECEIVE_TASK WaitForCallback ACTIVE s:10:04:59.000 dur:-")
	require.Equal(t, 1, strings.Count(output, "ReviewOrder"))
	require.NotContains(t, output, "hidden:")
}

// TestRenderOpsSlowProcessAnalysisResultHumanWithFullTimelineRestoresChronologicalDetails verifies the explicit audit view.
func TestRenderOpsSlowProcessAnalysisResultHumanWithFullTimelineRestoresChronologicalDetails(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	flagOpsAnalyseSlowProcessInstanceWithFullTimeline = true
	t.Cleanup(func() { flagOpsAnalyseSlowProcessInstanceWithFullTimeline = false })
	result := opsSlowProcessAnalysisRenderTestResult(
		opsSlowProcessAnalysisRenderTestElement("4108", "ReserveStock", "SERVICE_TASK", "COMPLETED", "2026-07-18T10:00:00Z", "2026-07-18T10:00:00Z", "0s", 0, 0),
		opsSlowProcessAnalysisRenderTestTransition("ReserveStock", "SERVICE_TASK", "PackOrder", "SERVICE_TASK", "2026-07-18T10:00:00Z", "2026-07-18T10:00:04Z", "4s", 4000, 1),
		opsSlowProcessAnalysisRenderTestElement("4109", "PackOrder", "SERVICE_TASK", "COMPLETED", "2026-07-18T10:00:04Z", "2026-07-18T10:00:12Z", "8s", 8000, 3),
	)

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "└─ elements:\n")
	require.Contains(t, output, "   ├─ 4108 SERVICE_TASK ReserveStock COMPLETED s:10:00:00.000 e:10:00:00.000 dur:0s")
	require.Contains(t, output, "   ├─ ReserveStock -> PackOrder: 4s [░░░░░░░░░░] 1%")
	require.Contains(t, output, "   └─ 4109 SERVICE_TASK PackOrder COMPLETED s:10:00:04.000 e:10:00:12.000 dur:8s [░░░░░░░░░░] 3%")
	require.Less(t, strings.Index(output, "4108 SERVICE_TASK ReserveStock"), strings.Index(output, "ReserveStock -> PackOrder"))
	require.Less(t, strings.Index(output, "ReserveStock -> PackOrder"), strings.Index(output, "4109 SERVICE_TASK PackOrder"))
	require.NotContains(t, output, "slowest elements:")
	require.NotContains(t, output, "hidden:")
	require.NotContains(t, output, "PackOrder -> ReserveStock")
	require.True(t, strings.HasSuffix(output, "process instances: 1\n"))
}

// TestRenderOpsSlowProcessAnalysisResultHumanOmitsHiddenSummaryForEmptyTimeline verifies roots without details do not imply hidden rows.
func TestRenderOpsSlowProcessAnalysisResultHumanOmitsHiddenSummaryForEmptyTimeline(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	result := opsSlowProcessAnalysisRenderTestResult()

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	output := buf.String()
	require.NotContains(t, output, "slowest elements:")
	require.NotContains(t, output, "hidden:")
}

// newOpsSlowProcessAnalysisRenderTestCommand captures command output without mutating the root command.
func newOpsSlowProcessAnalysisRenderTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{Use: "slow-process-instances"}
	cmd.SetOut(buf)
	return cmd, buf
}

// opsSlowProcessAnalysisRenderTestResult builds one complete root with caller-supplied detail rows.
func opsSlowProcessAnalysisRenderTestResult(entries ...ops.SlowProcessAnalysisTimelineEntry) ops.SlowProcessAnalysisResult {
	return ops.SlowProcessAnalysisResult{
		Items: []ops.SlowProcessAnalysisProcessInstance{
			opsSlowProcessAnalysisRenderTestRoot(entries...),
		},
		Count: 1,
	}
}

// opsSlowProcessAnalysisRenderTestRoot keeps fixture roots consistent across summary and full-timeline tests.
func opsSlowProcessAnalysisRenderTestRoot(entries ...ops.SlowProcessAnalysisTimelineEntry) ops.SlowProcessAnalysisProcessInstance {
	return ops.SlowProcessAnalysisProcessInstance{
		Key:                    "2251799813685249",
		TenantID:               "tenant-a",
		BpmnProcessID:          "OrderProcess",
		ProcessDefinitionKey:   "2251799813687001",
		ProcessVersion:         7,
		State:                  process.StateCompleted,
		StartDate:              "2026-07-18T10:00:00Z",
		EndDate:                "2026-07-18T10:05:00Z",
		RootProcessInstanceKey: "2251799813685249",
		Duration:               "5m0s",
		DurationMillis:         300000,
		DurationAvailable:      true,
		Timeline:               append([]ops.SlowProcessAnalysisTimelineEntry(nil), entries...),
	}
}

// opsSlowProcessAnalysisRenderTestElement builds an element timeline row with only renderer-relevant fields.
func opsSlowProcessAnalysisRenderTestElement(key string, id string, typ string, state string, start string, end string, duration string, millis int64, share int) ops.SlowProcessAnalysisTimelineEntry {
	return ops.SlowProcessAnalysisTimelineEntry{
		Kind:                 ops.SlowProcessAnalysisTimelineEntryKindElement,
		ElementInstanceKey:   key,
		ElementID:            id,
		Type:                 typ,
		State:                state,
		StartDate:            start,
		EndDate:              end,
		Duration:             duration,
		DurationMillis:       millis,
		DurationAvailable:    duration != "",
		ProcessDurationShare: share,
	}
}

// opsSlowProcessAnalysisRenderTestTransition builds an adjacent timing row with only renderer-relevant fields.
func opsSlowProcessAnalysisRenderTestTransition(fromID string, fromType string, toID string, toType string, fromEnd string, toStart string, duration string, millis int64, share int) ops.SlowProcessAnalysisTimelineEntry {
	return ops.SlowProcessAnalysisTimelineEntry{
		Kind:                 ops.SlowProcessAnalysisTimelineEntryKindTransition,
		FromElementID:        fromID,
		FromElementType:      fromType,
		FromEndDate:          fromEnd,
		ToElementID:          toID,
		ToElementType:        toType,
		ToStartDate:          toStart,
		Duration:             duration,
		DurationMillis:       millis,
		DurationAvailable:    duration != "",
		ProcessDurationShare: share,
	}
}
