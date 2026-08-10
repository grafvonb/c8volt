// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestGetProcessDefinitionWatchSnapshotRequestBroadMode verifies unselected
// watch mode keeps using the broad discovery request shape.
func TestGetProcessDefinitionWatchSnapshotRequestBroadMode(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	request := newGetProcessDefinitionWatchSnapshotRequest(process.ProcessDefinitionFilter{})

	require.True(t, request.WatchAllWhenUnselected)
	require.Empty(t, request.Key)
	require.Equal(t, process.ProcessDefinitionFilter{}, request.Filter)
	require.Equal(t, consts.MaxPISearchSize, request.Page.Size)
	require.False(t, request.Latest)
}

// TestGetProcessDefinitionWatchSnapshotRequestPreservesSelectors verifies
// selector-backed watch refreshes keep their selector and do not broaden scope.
func TestGetProcessDefinitionWatchSnapshotRequestPreservesSelectors(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	flagGetPDLatest = true
	flagGetPDBatchSize = 25
	filter := process.ProcessDefinitionFilter{
		BpmnProcessId:     "invoice",
		ProcessVersion:    3,
		ProcessVersionTag: "stable",
	}

	request := newGetProcessDefinitionWatchSnapshotRequest(filter)

	require.False(t, request.WatchAllWhenUnselected)
	require.Empty(t, request.Key)
	require.Equal(t, filter, request.Filter)
	require.Equal(t, int32(25), request.Page.Size)
	require.True(t, request.Latest)
}

// TestGetProcessDefinitionWatchOutputParityAssertions pins the watch-specific
// output contract before the lifecycle declarations move into a focused file.
func TestGetProcessDefinitionWatchOutputParityAssertions(t *testing.T) {
	t.Run("human and verbose refresh bodies match normal process-definition rows", func(t *testing.T) {
		tests := []struct {
			name       string
			verbose    bool
			wantStderr string
		}{
			{name: "human"},
			{name: "verbose", verbose: true, wantStderr: "process-definition watch refresh 1 completed in 1ms (interval 1s, status: on-time)\n"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resetGetProcessDefinitionCommandGlobals()
				t.Cleanup(resetGetProcessDefinitionCommandGlobals)
				flagVerbose = tt.verbose

				cli := processDefinitionWatchTestAPI{
					collect: func(_ context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
						return process.ProcessDefinitionWatchSnapshot{
							Items: []process.ProcessDefinition{{
								Key:            "2251799813685255",
								TenantId:       "tenant",
								BpmnProcessId:  "invoice",
								ProcessVersion: 3,
							}},
							Total: 1,
						}, nil
					},
				}

				result := executeGetProcessDefinitionWatchHarnessForTest(t, processDefinitionWatchHarness{
					cli:        cli,
					filter:     process.ProcessDefinitionFilter{},
					maxRetries: defaultBackoffMaxRetries,
					now: newProcessDefinitionWatchClockForTest(time.Unix(0, 0),
						0, time.Millisecond,
					),
					sleep: func(context.Context, time.Duration) error {
						return context.Canceled
					},
				})

				require.NoError(t, result.err)
				requireProcessDefinitionWatchRepaintCount(t, result, 1)
				require.Equal(t, "2251799813685255 tenant invoice v3\nfound: 1\n", result.stdoutWithoutRepaintControls())
				require.NotContains(t, result.stdoutWithoutRepaintControls(), "snapshot")
				require.Equal(t, tt.wantStderr, result.stderr)
			})
		}
	})

	t.Run("machine and noninteractive modes are rejected before watch refresh", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(*cobra.Command)
			want  string
		}{
			{
				name: "json rejection",
				setup: func(*cobra.Command) {
					flagViewAsJson = true
				},
				want: "--json",
			},
			{
				name: "keys-only rejection",
				setup: func(*cobra.Command) {
					flagViewKeysOnly = true
				},
				want: "--keys-only",
			},
			{
				name: "quiet rejection",
				setup: func(*cobra.Command) {
					flagQuiet = true
				},
				want: "--quiet",
			},
			{
				name: "automation rejection",
				setup: func(*cobra.Command) {
					flagCmdAutomation = true
				},
				want: "--automation",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resetGetProcessDefinitionCommandGlobals()
				t.Cleanup(resetGetProcessDefinitionCommandGlobals)
				cmd := &cobra.Command{Use: "process-definition"}
				cmd.SetContext(context.Background())
				flagGetPDWatch = true
				tt.setup(cmd)

				err := validateGetProcessDefinitionFlags(cmd)

				require.Error(t, err)
				require.Contains(t, err.Error(), "--watch cannot be combined")
				require.Contains(t, err.Error(), tt.want)
				require.Contains(t, err.Error(), "watch repaints terminal output")
			})
		}
	})
}
