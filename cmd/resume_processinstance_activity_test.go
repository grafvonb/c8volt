// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/testx"
	"io"
	"os"
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

// resumptionActivityAPI isolates collection execution while retaining real command activity.
type resumptionActivityAPI struct {
	process.API
	resume func(context.Context, typex.Keys, int, ...foptions.FacadeOption) (process.ResumptionResults, error)
}

// ResumeProcessInstances delegates the isolated collection callback.
func (s resumptionActivityAPI) ResumeProcessInstances(ctx context.Context, keys typex.Keys, workers int, opts ...foptions.FacadeOption) (process.ResumptionResults, error) {
	return s.resume(ctx, keys, workers, opts...)
}

// resumptionActivityWriter checks activity cleanup at the configured logger destination.
type resumptionActivityWriter func([]byte) (int, error)

// Write checks activity before forwarding output.
func (w resumptionActivityWriter) Write(p []byte) (int, error) { return w(p) }

// The command owns one activity across collection execution, after approval and
// before final output. The shared writer still owns animation and terminal checks.
func TestResumeProcessInstanceActivity(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		setup                     func(*config.Config)
		active, preview, panicRun bool
	}{
		{name: "waiting", active: true},
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
			resetCommandTreeFlags(Root())
			t.Cleanup(func() { resetCommandTreeFlags(Root()) })
			cfg := config.New()
			if tc.setup != nil {
				tc.setup(cfg)
			}
			sink := &activitysink.Sink{}
			var stdout, stderr bytes.Buffer
			log := logging.New(logging.LoggerConfig{Level: "info", Writer: resumptionActivityWriter(func(p []byte) (int, error) {
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
			calls := 0
			api := resumptionActivityAPI{resume: func(ctx context.Context, keys typex.Keys, workers int, opts ...foptions.FacadeOption) (process.ResumptionResults, error) {
				calls++
				preview := foptions.ApplyFacadeOptions(opts).DryRun
				if preview {
					require.Zero(t, sink.Started())
				} else {
					require.True(t, confirmed)
					if tc.active {
						require.Equal(t, []activitysink.Start{{Message: "resuming 2 process instance(s)", Importance: logging.ActivityImportanceWorkflow}}, sink.Starts())
					} else {
						require.Zero(t, sink.Started())
					}
					require.Zero(t, sink.Stopped(), "activity must cover the whole facade call")
					if tc.panicRun {
						panic("execution interrupted")
					}
				}
				items := make([]process.ResumptionResult, len(keys))
				for i, key := range keys {
					items[i] = process.ResumptionResult{Key: key, Status: process.ResumptionStatusConfirmed, Submission: process.ResumptionSubmissionAccepted}
					if preview {
						items[i].Status = process.ResumptionStatusPreview
					}
				}
				return process.ResumptionResults{Items: items}, nil
			}}
			run := func() { runResumeProcessInstances(cmd, api, log, cfg, typex.Keys{resumeKeyA, resumeKeyB}) }
			if tc.panicRun {
				require.PanicsWithValue(t, "execution interrupted", run)
			} else {
				run()
			}
			require.Equal(t, sink.Started(), sink.Stopped(), "activity must be closed before returning or unwinding")
			if tc.preview {
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

// TestResumeProcessInstanceActivityFailure verifies cleanup before logging and exiting
// for rejection, interrupted confirmation and result-writer failure.
func TestResumeProcessInstanceActivityFailure(t *testing.T) {
	for _, scenario := range []string{"rejected", "interrupted", "writer", "empty preview", "empty preview json"} {
		t.Run(scenario, func(t *testing.T) {
			out, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestResumeProcessInstanceActivityFailureProcess", "", map[string]string{"C8VOLT_RESUME_ACTIVITY_SCENARIO": scenario}, "")
			code := 6
			if scenario != "rejected" {
				code = 1
			}
			assertUserTaskCompletionExit(t, err, code)
			if scenario == "empty preview json" {
				require.Empty(t, stderr)
				env := requireSingleJSONObjectDocument(t, out)
				require.Equal(t, "failed", env["outcome"])
				require.Equal(t, "local_precondition", env["class"])
				require.NotContains(t, env, "payload")
				return
			}
			require.Empty(t, out)
			records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
			require.Len(t, records, 1)
			require.Equal(t, "ERROR", records[0][0])
			if scenario == "empty preview" {
				require.Equal(t, "resume process instances: preview returned no targets", records[0][1])
			} else if scenario == "writer" {
				require.Equal(t, "resume process instances: could not write result", records[0][1])
			} else {
				require.Equal(t, "resume process instances: "+scenario+": 1", records[0][1])
			}
		})
	}
}

// TestResumeProcessInstanceActivityFailureProcess uses isolated facade results and
// the real exit path; writes assert the activity stopped before any output.
func TestResumeProcessInstanceActivityFailureProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	scenario := os.Getenv("C8VOLT_RESUME_ACTIVITY_SCENARIO")
	resetCommandTreeFlags(Root())
	sink := &activitysink.Sink{}
	wantActivity := 1
	if scenario == "empty preview" || scenario == "empty preview json" {
		wantActivity = 0
		flagDryRun = scenario == "empty preview json"
		flagNoWait = true
	}
	if scenario == "empty preview json" {
		flagViewAsJson = true
	}
	log := logging.New(logging.LoggerConfig{Level: "warn", Format: "plain-time", Writer: resumptionActivityWriter(func(p []byte) (int, error) {
		require.Equal(t, wantActivity, sink.Started())
		require.Equal(t, wantActivity, sink.Stopped(), "stop before failure diagnostics")
		return os.Stderr.Write(p)
	})})
	c := &cobra.Command{Use: "process-instance"}
	setContractSupport(c, ContractSupportFull)
	c.SetContext(logging.ToActivityContext(logging.ToContext(context.Background(), log), sink))
	c.SetErr(os.Stderr)
	c.SetOut(resumptionActivityWriter(func(p []byte) (int, error) {
		if scenario == "empty preview json" {
			require.Zero(t, sink.Started())
			return os.Stdout.Write(p)
		}
		require.Equal(t, 1, sink.Stopped(), "stop before failed result write")
		return 0, fmt.Errorf("isolated writer failure")
	}))
	confirmCmdOrAbortFn = func(io.Writer, bool, string) error {
		require.NotZero(t, wantActivity, "absent reports must not prompt")
		require.Zero(t, sink.Started())
		return nil
	}
	api := resumptionActivityAPI{resume: func(ctx context.Context, keys typex.Keys, workers int, opts ...foptions.FacadeOption) (process.ResumptionResults, error) {
		item := process.ResumptionResult{Key: resumeKeyA, Status: process.ResumptionStatusPreview}
		if foptions.ApplyFacadeOptions(opts).DryRun {
			require.Zero(t, sink.Started())
			if wantActivity == 0 {
				return process.ResumptionResults{}, nil
			}
			return process.ResumptionResults{Items: []process.ResumptionResult{item}}, nil
		}
		require.Equal(t, 1, sink.Started())
		require.Zero(t, sink.Stopped())
		item.Status, item.Submission = process.ResumptionStatusFailed, process.ResumptionSubmissionRejected
		operationErr := ferrors.ErrConflict
		if scenario == "interrupted" {
			item.Status, item.Submission, item.Confirmation = process.ResumptionStatusInterrupted, process.ResumptionSubmissionAccepted, process.ResumptionConfirmationUnconfirmed
			operationErr = context.Canceled
		}
		if scenario == "writer" {
			// Switch output only after activity starts to exercise a failed final write.
			flagViewAsJson = true
			operationErr = nil
			item.Status, item.Submission = process.ResumptionStatusConfirmed, process.ResumptionSubmissionAccepted
		}
		return process.ResumptionResults{Items: []process.ResumptionResult{item}}, operationErr
	}}
	runResumeProcessInstances(c, api, log, config.New(), typex.Keys{resumeKeyA})
	t.Fatal("expected failure exit")
}

// TestResumeProcessInstanceFrozenPreview protects the approved count and copied
// keys even if the caller or preview storage changes during approval.
func TestResumeProcessInstanceFrozenPreview(t *testing.T) {
	resetCommandTreeFlags(Root())
	t.Cleanup(func() { resetCommandTreeFlags(Root()) })
	flagNoIndicator = true
	keys := typex.Keys{resumeKeyA, resumeKeyB}
	preview := process.ResumptionResults{Items: []process.ResumptionResult{{Key: resumeKeyA}, {Key: resumeKeyB}}}
	previousConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = previousConfirm })
	confirmCmdOrAbortFn = func(_ io.Writer, _ bool, prompt string) error {
		require.Equal(t, "You are about to resume 2 process instance(s). Do you want to proceed?", prompt)
		keys[0], preview.Items[0].Key = resumeKeyC, resumeKeyC
		return nil
	}
	calls := 0
	api := resumptionActivityAPI{resume: func(ctx context.Context, got typex.Keys, workers int, opts ...foptions.FacadeOption) (process.ResumptionResults, error) {
		calls++
		require.Equal(t, typex.Keys{resumeKeyA, resumeKeyB}, got)
		if foptions.ApplyFacadeOptions(opts).DryRun {
			return preview, nil
		}
		return process.ResumptionResults{Items: []process.ResumptionResult{{Key: got[0]}, {Key: got[1]}}}, nil
	}}
	log := logging.New(logging.LoggerConfig{Level: "warn", Writer: io.Discard})
	c := &cobra.Command{Use: "process-instance"}
	c.SetContext(logging.ToContext(context.Background(), log))
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	runResumeProcessInstances(c, api, log, config.New(), keys)
	require.Equal(t, 2, calls, "preview once, execute once, no discovery")
}
