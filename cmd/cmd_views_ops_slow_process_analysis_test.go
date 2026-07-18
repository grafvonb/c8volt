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
						Kind:               ops.SlowProcessAnalysisTimelineEntryKindElement,
						ElementInstanceKey: "2251799813685251",
						ElementID:          "Work",
						Type:               "SERVICE_TASK",
						State:              "COMPLETED",
						StartDate:          "2026-07-18T10:00:00Z",
						EndDate:            "2026-07-18T10:00:05Z",
						Duration:           "5s",
						DurationMillis:     5000,
						DurationAvailable:  true,
						RelativePercentile: 83,
						RelativeBar:        "[########--]",
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
	require.Contains(t, output, "└─ elements:\n")
	require.Contains(t, output, "   ├─ 2251799813685250 SERVICE_TASK ReserveStock COMPLETED s:10:00:00.000 e:10:00:00.000 dur:0s")
	require.NotContains(t, output, "dur:0s [")
	require.Contains(t, output, "   └─ 2251799813685251 SERVICE_TASK Work COMPLETED s:10:00:00.000 e:10:00:05.000 dur:5s [█████░░░░░] 50%")
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
	require.Contains(t, buf.String(), "└─ elements:\n   ├─ 4108 SERVICE_TASK ReserveStock COMPLETED s:08:10:00.300 e:08:10:04.500 dur:4.2s [░░░░░░░░░░] <1%")
	require.Contains(t, buf.String(), "   ├─ ReserveStock -> OrderFinished: 14m20s [██████████] 99%")
	require.Contains(t, buf.String(), "   └─ 4122 END_EVENT OrderFinished COMPLETED s:08:24:24.500 e:08:24:24.508 dur:8ms [░░░░░░░░░░] <1%")
	require.NotContains(t, buf.String(), "PI:")
}

// TestRenderOpsSlowProcessAnalysisResultHumanRendersTimelineDetails verifies compact element and transition detail rows.
func TestRenderOpsSlowProcessAnalysisResultHumanRendersTimelineDetails(t *testing.T) {
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
	require.Contains(t, output, "└─ elements:\n")
	require.Contains(t, output, "   ├─ 2251799813685250 SERVICE_TASK ReserveStock COMPLETED s:10:00:00.000 e:10:00:04.000 dur:4s [░░░░░░░░░░] 1% inc!:2251799813687777")
	require.Contains(t, output, "   └─ ReserveStock -> OrderFinished: 4m56s [██████████] 99%")
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

// newOpsSlowProcessAnalysisRenderTestCommand captures command output without mutating the root command.
func newOpsSlowProcessAnalysisRenderTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{Use: "slow-process-instances"}
	cmd.SetOut(buf)
	return cmd, buf
}
