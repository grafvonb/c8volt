// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestProcessDefinitionDeploySemanticProgressRoutesFacadeCompletion verifies
// deployment completion facts lazily start one workflow-priority reporter.
func TestProcessDefinitionDeploySemanticProgressRoutesFacadeCompletion(t *testing.T) {
	resetDeployCommandStateForTest()
	t.Cleanup(resetDeployCommandStateForTest)

	sink := &activitysink.Sink{}
	cmd := newHTTPFallbackActivityCommand(sink)
	progress := newProcessDefinitionDeploySemanticProgress(cmd)
	defer progress.Close()

	require.Empty(t, sink.Starts())
	progress.FacadeProgress(foptions.ProgressEvent{
		Kind: foptions.ProgressEventKindCompletion,
		Completion: &foptions.CompletionProgress{
			Phase:       processDefinitionDeployCompletionPhase,
			Total:       2,
			Identity:    "pd-1",
			Disposition: foptions.CompletionDispositionSubmitted,
		},
	})
	progress.FacadeProgress(foptions.ProgressEvent{
		Kind: foptions.ProgressEventKindCompletion,
		Completion: &foptions.CompletionProgress{
			Phase:       processDefinitionDeployCompletionPhase,
			Total:       2,
			Identity:    "pd-2",
			Disposition: foptions.CompletionDispositionConfirmed,
		},
	})

	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "deploying process definitions, 0/2 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "deploying process definitions, 1/2 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "deploying process definitions, 2/2 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
}

// TestProcessDefinitionDeploySemanticProgressIgnoresOtherCompletionPhases
// prevents follow-up run completions from opening a deployment activity when a
// deployment service cannot prove process-definition keys.
func TestProcessDefinitionDeploySemanticProgressIgnoresOtherCompletionPhases(t *testing.T) {
	resetDeployCommandStateForTest()
	t.Cleanup(resetDeployCommandStateForTest)

	sink := &activitysink.Sink{}
	cmd := newHTTPFallbackActivityCommand(sink)
	progress := newProcessDefinitionDeploySemanticProgress(cmd)
	defer progress.Close()

	progress.FacadeProgress(foptions.ProgressEvent{
		Kind: foptions.ProgressEventKindCompletion,
		Completion: &foptions.CompletionProgress{
			Phase:       "create",
			Total:       1,
			Identity:    "pi-1",
			Disposition: foptions.CompletionDispositionConfirmed,
		},
	})

	require.Empty(t, sink.Starts())
	require.Empty(t, sink.PriorityUpdates())
}

// TestAppendProcessDefinitionDeployProgressOptions verifies deployment commands
// preserve existing facade options while installing structured progress and
// suppressing duplicate service workflow detail logs.
func TestAppendProcessDefinitionDeployProgressOptions(t *testing.T) {
	resetDeployCommandStateForTest()
	t.Cleanup(resetDeployCommandStateForTest)

	sink := &activitysink.Sink{}
	cmd := newHTTPFallbackActivityCommand(sink)
	progress := newProcessDefinitionDeploySemanticProgress(cmd)
	defer progress.Close()

	opts := appendProcessDefinitionDeployProgressOptions(cmd, []foptions.FacadeOption{foptions.WithNoWait()}, progress)
	cfg := foptions.ApplyFacadeOptions(opts)

	require.True(t, cfg.NoWait)
	require.True(t, cfg.SuppressWorkflowDetailLogs)
	require.NotNil(t, cfg.Progress)

	cfg.Progress(foptions.ProgressEvent{
		Kind: foptions.ProgressEventKindCompletion,
		Completion: &foptions.CompletionProgress{
			Phase:       processDefinitionDeployCompletionPhase,
			Total:       1,
			Identity:    "pd-1",
			Disposition: foptions.CompletionDispositionConfirmed,
		},
	})

	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "deploying process definitions, 1/1 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
}
