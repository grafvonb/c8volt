// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// suspensionActivityAPI isolates planning and execution while retaining real command activity.
type suspensionActivityAPI struct {
	process.API
	plan    func(context.Context, typex.Keys, ...foptions.FacadeOption) (process.ProcessInstanceSuspensionPlan, error)
	suspend func(context.Context, typex.Keys, int, ...foptions.FacadeOption) (process.SuspensionResults, error)
}

// SuspendProcessInstances delegates the isolated collection callback.
func (s suspensionActivityAPI) SuspendProcessInstances(ctx context.Context, keys typex.Keys, workers int, opts ...foptions.FacadeOption) (process.SuspensionResults, error) {
	return s.suspend(ctx, keys, workers, opts...)
}

// PlanProcessInstanceSuspension delegates the isolated planner callback.
func (s suspensionActivityAPI) PlanProcessInstanceSuspension(ctx context.Context, keys typex.Keys, opts ...foptions.FacadeOption) (process.ProcessInstanceSuspensionPlan, error) {
	return s.plan(ctx, keys, opts...)
}

// suspensionActivityWriter asserts activity cleanup at the logger destination.
type suspensionActivityWriter func([]byte) (int, error)

// Write checks activity before forwarding output.
func (w suspensionActivityWriter) Write(p []byte) (int, error) { return w(p) }

// The command owns one activity across collection execution, after approval and
// before final output. The shared writer still owns animation and terminal checks.
func TestSuspendProcessInstanceActivity(t *testing.T) {
	t.Run("exact", func(t *testing.T) { testSuspendProcessInstanceActivity(t, false) })
	t.Run("tree", func(t *testing.T) { testSuspendProcessInstanceActivity(t, true) })
}

// testSuspendProcessInstanceActivity applies lifecycle checks to both runners.
func testSuspendProcessInstanceActivity(t *testing.T, tree bool) {
	for _, tc := range []struct {
		name                             string
		setup                            func(*config.Config)
		active, preview, panicRun, empty bool
	}{
		{name: "waiting", active: true},
		{name: "no active targets", empty: true},
		{name: "no active no-wait", empty: true, setup: func(*config.Config) { flagNoWait = true }},
		{name: "no wait", active: true, setup: func(*config.Config) { flagNoWait = true }},
		{name: "execution unwinds", active: true, panicRun: true},
		{name: "dry run", preview: true, setup: func(*config.Config) { flagDryRun = true }},
		{name: "dry run no-wait", preview: true, setup: func(*config.Config) { flagDryRun = true; flagNoWait = true }},
		{name: "json", setup: func(*config.Config) { flagViewAsJson = true }},
		{name: "keys", setup: func(*config.Config) { flagViewKeysOnly = true }},
		{name: "quiet", setup: func(*config.Config) { flagQuiet = true }},
		{name: "no indicator", setup: func(*config.Config) { flagNoIndicator = true }},
		{name: "automation", setup: func(cfg *config.Config) { cfg.App.Automation = true }},
		{name: "structured logging", setup: func(cfg *config.Config) { cfg.Log.Format = "json" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.empty && !tree {
				return
			}
			resetCommandTreeFlags(Root())
			t.Cleanup(func() { resetCommandTreeFlags(Root()) })
			cfg := config.New()
			if tc.setup != nil {
				tc.setup(cfg)
			}
			sink := &activitysink.Sink{}
			var stdout, stderr bytes.Buffer
			log := logging.New(logging.LoggerConfig{Level: "info", Writer: suspensionActivityWriter(func(p []byte) (int, error) {
				require.Equal(t, sink.Started(), sink.Stopped(), "activity must stop before final logging")
				return stderr.Write(p)
			})})
			cmd := &cobra.Command{Use: "process-instance"}
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetContext(logging.ToActivityContext(logging.ToContext(context.Background(), log), sink))
			previousConfirm := confirmCmdOrAbortFn
			t.Cleanup(func() { confirmCmdOrAbortFn = previousConfirm })
			confirmed := false
			confirmCmdOrAbortFn = func(io.Writer, bool, string) error {
				require.Zero(t, sink.Started(), "activity must not cover the confirmation question")
				confirmed = true
				return nil
			}
			calls, plans := 0, 0
			plan := process.ProcessInstanceSuspensionPlan{
				RequestedKeys: typex.Keys{suspendKeyA},
				TargetKeys:    typex.Keys{suspendKeyA, suspendKeyB},
				Instances: []process.ProcessInstanceSuspensionPlanItem{
					{Key: suspendKeyA, ObservedState: process.StateActive, Classification: process.ProcessInstanceSuspensionPlanEligible},
					{Key: suspendKeyB, ObservedState: process.StateActive, Classification: process.ProcessInstanceSuspensionPlanEligible},
				},
			}
			if tc.empty {
				plan.TargetKeys = typex.Keys{}
				for i := range plan.Instances {
					plan.Instances[i].ObservedState = process.StateSuspended
					plan.Instances[i].Classification = process.ProcessInstanceSuspensionPlanAlreadySuspended
				}
			}
			api := suspensionActivityAPI{plan: func(ctx context.Context, keys typex.Keys, opts ...foptions.FacadeOption) (process.ProcessInstanceSuspensionPlan, error) {
				plans++
				require.Zero(t, sink.Started(), "planning has no execution activity")
				require.False(t, confirmed)
				require.True(t, foptions.ApplyFacadeOptions(opts).IgnoreTenant)
				return plan, nil
			}, suspend: func(ctx context.Context, keys typex.Keys, workers int, opts ...foptions.FacadeOption) (process.SuspensionResults, error) {
				calls++
				require.False(t, tc.empty, "empty plans must not call the mutation facade")
				preview := foptions.ApplyFacadeOptions(opts).DryRun
				if preview {
					require.Zero(t, sink.Started())
				} else {
					require.True(t, confirmed)
					if tc.active {
						require.Equal(t, []activitysink.Start{{Message: "suspending 2 process instance(s)", Importance: logging.ActivityImportanceWorkflow}}, sink.Starts())
					} else {
						require.Zero(t, sink.Started())
					}
					require.Zero(t, sink.Stopped(), "activity must cover the whole facade call")
					if tc.panicRun {
						panic("execution interrupted")
					}
				}
				items := make([]process.SuspensionResult, len(keys))
				for i, key := range keys {
					items[i] = process.SuspensionResult{Key: key, Status: process.SuspensionStatusConfirmed, Submission: process.SuspensionSubmissionAccepted}
					if preview {
						items[i].Status = process.SuspensionStatusPreview
					}
				}
				if tree {
					keys[0] = suspendKeyC
				}
				return process.SuspensionResults{Items: items}, nil
			}}
			run := func() {
				if tree {
					runSuspendProcessInstanceTree(cmd, api, log, cfg, typex.Keys{suspendKeyA})
					return
				}
				runSuspendProcessInstances(cmd, api, log, cfg, typex.Keys{suspendKeyA, suspendKeyB})
			}
			if tc.panicRun {
				require.PanicsWithValue(t, "execution interrupted", run)
			} else {
				run()
			}
			require.Equal(t, sink.Started(), sink.Stopped(), "activity must be closed before returning or unwinding")
			if tree {
				require.Equal(t, 1, plans)
				want := 1
				if tc.preview || tc.empty {
					want = 0
					require.False(t, confirmed)
				}
				require.Equal(t, want, calls)
				if !tc.empty {
					require.Equal(t, typex.Keys{suspendKeyA, suspendKeyB}, plan.TargetKeys, "executor cannot mutate frozen report")
				}
			} else if tc.preview {
				require.Equal(t, 1, calls)
				require.False(t, confirmed)
			} else {
				require.Equal(t, 2, calls)
			}
			if !flagViewAsJson && !flagViewKeysOnly {
				require.Empty(t, stdout.String())
			}
		})
	}
}
