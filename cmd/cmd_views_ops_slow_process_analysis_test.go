// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
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

// TestRenderOpsSlowProcessAnalysisResultKeysOnlyRendersRootKeys verifies keyed output remains pipeline-safe.
func TestRenderOpsSlowProcessAnalysisResultKeysOnlyRendersRootKeys(t *testing.T) {
	cmd, buf := newOpsSlowProcessAnalysisRenderTestCommand()
	flagViewKeysOnly = true
	t.Cleanup(func() { flagViewKeysOnly = false })
	result := ops.SlowProcessAnalysisResult{
		Items: []ops.SlowProcessAnalysisProcessInstance{
			{Key: "2251799813685249"},
			{Key: "2251799813685250"},
		},
		Count: 2,
	}

	err := renderOpsSlowProcessAnalysisResult(cmd, result)

	require.NoError(t, err)
	require.Equal(t, "2251799813685249\n2251799813685250\n", buf.String())
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
