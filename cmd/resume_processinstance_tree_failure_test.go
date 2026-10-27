// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestResumeProcessInstanceTreeIncompletePlan enforces the command's error-first
// boundary even if a custom facade returns partial plan facts together with an error.
func TestResumeProcessInstanceTreeIncompletePlan(t *testing.T) {
	for _, failure := range []string{"partial", "unresolved", "canceled", "missing report"} {
		for _, mode := range []string{"human", "json", "preview", "no-wait"} {
			t.Run(failure+"/"+mode, func(t *testing.T) {
				out, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestResumeProcessInstanceTreePlanFailureProcess", "", map[string]string{
					"C8VOLT_RESUME_PLAN_FAILURE": failure, "C8VOLT_RESUME_PLAN_MODE": mode,
				}, "")
				assertUserTaskCompletionExit(t, err, 1)
				class, reason := "local_precondition", "local precondition"
				if failure == "missing report" {
					reason = "no resolved scope"
				}
				if mode == "json" {
					require.Empty(t, stderr)
					env := requireSingleJSONObjectDocument(t, out)
					require.Equal(t, "failed", env["outcome"])
					require.Equal(t, class, env["class"])
					require.NotContains(t, env, "payload", "no partial executable plan escapes")
				} else {
					require.Empty(t, out)
					require.Equal(t, [][2]string{{"ERROR", "plan process instance resumption: " + reason}}, bulkCompletionDiagnosticRecords(t, stderr, "json"))
				}
			})
		}
	}
}

// TestResumeProcessInstanceTreePlanFailureProcess executes Cobra with an isolated
// planner; any approval, activity, or mutation after failed planning fails the child.
func TestResumeProcessInstanceTreePlanFailureProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	mode := os.Getenv("C8VOLT_RESUME_PLAN_MODE")
	flagViewAsJson = mode == "json"
	flagDryRun = mode == "preview"
	flagNoWait = mode == "no-wait"
	sink := &activitysink.Sink{}
	log := logging.New(logging.LoggerConfig{Level: "info", Format: "json", Writer: os.Stderr})
	cfg := config.New()
	cmd := &cobra.Command{Use: "process-instance"}
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SetArgs(nil)
	cmd.SetContext(logging.ToActivityContext(logging.ToContext(context.Background(), log), sink))
	setContractSupport(cmd, ContractSupportFull)
	setCommandMutation(cmd, CommandMutationStateChanging)
	confirmCmdOrAbortFn = func(io.Writer, bool, string) error { t.Fatal("failed planning must not prompt"); return nil }
	api := resumptionActivityAPI{
		plan: func(context.Context, typex.Keys, ...foptions.FacadeOption) (process.ProcessInstanceResumptionPlan, error) {
			require.Zero(t, sink.Started())
			if os.Getenv("C8VOLT_RESUME_PLAN_FAILURE") == "missing report" {
				return process.ProcessInstanceResumptionPlan{}, nil
			}
			err := ferrors.WrapClass(ferrors.ErrLocalPrecondition, fmt.Errorf("%s descendant discovery", os.Getenv("C8VOLT_RESUME_PLAN_FAILURE")))
			if os.Getenv("C8VOLT_RESUME_PLAN_FAILURE") == "canceled" {
				err = context.Canceled
			}
			return process.ProcessInstanceResumptionPlan{TargetKeys: typex.Keys{resumeKeyA}}, err
		},
		resume: func(context.Context, typex.Keys, int, ...foptions.FacadeOption) (process.ResumptionResults, error) {
			t.Fatal("failed planning must not execute")
			return process.ResumptionResults{}, nil
		},
	}
	cmd.Run = func(cmd *cobra.Command, _ []string) {
		runResumeProcessInstanceTree(cmd, api, log, cfg, typex.Keys{resumeKeyA})
	}
	require.NoError(t, cmd.Execute())
	t.Fatal("failed planning must exit")
}

// TestResumeProcessInstanceTreeChildScope proves selecting a child does not
// fetch or mutate its ancestor, even when the selected child itself is ineligible.
func TestResumeProcessInstanceTreeChildScope(t *testing.T) {
	f := newResumptionTreeFixture(t)
	f.observed[resumeKeyB] = "ACTIVE"
	out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", "resume", "pi", "--tree", "--key", resumeKeyB, "--json", "--auto-confirm")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	payload := requireJSONObject(t, requireSingleJSONObjectDocument(t, out)["payload"])
	plan := requireJSONObject(t, payload["plan"])
	require.Equal(t, []any{resumeKeyB}, plan["requestedKeys"])
	require.Equal(t, []any{resumeKeyC}, plan["targetKeys"])
	requireJSONItems(t, plan["instances"], 2)
	require.Equal(t, []string{"GET " + resumeKeyB, "POST " + resumeKeyB, "POST " + resumeKeyC}, f.discoverySnapshot())
	_, writes, reads := f.snapshot()
	require.Equal(t, map[string]int{resumeKeyC: 1}, writes)
	require.Equal(t, writes, reads)
}
