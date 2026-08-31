// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestOpsExecuteSmokeTestSemanticProgressRoutesStageCompletions verifies
// smoke-test completion facts open stage-specific workflow activities while
// nested lower-level phases stay out of the smoke-test reporter.
func TestOpsExecuteSmokeTestSemanticProgressRoutesStageCompletions(t *testing.T) {
	prevVerbose, prevJSON, prevKeysOnly, prevQuiet, prevDebug := flagVerbose, flagViewAsJson, flagViewKeysOnly, flagQuiet, flagDebug
	t.Cleanup(func() {
		flagVerbose = prevVerbose
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
		flagQuiet = prevQuiet
		flagDebug = prevDebug
	})
	flagVerbose, flagViewAsJson, flagViewKeysOnly, flagQuiet, flagDebug = false, false, false, false, false

	sink := &activitysink.Sink{}
	cmd := newHTTPFallbackActivityCommand(sink)
	request := ops.SmokeTestRequest{}
	progress := configureOpsExecuteSmokeTestProgress(cmd, &request)
	defer progress.Close()

	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:       "create",
			Total:       1,
			Identity:    "101",
			Disposition: ops.CompletionDispositionConfirmed,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:        "deploying smoke-test fixture",
			CoreResource: "deployment(s)",
			Total:        1,
			Identity:     "pd-88",
			Disposition:  ops.CompletionDispositionConfirmed,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:        "starting process instances",
			CoreResource: "process instance(s)",
			Total:        2,
			Identity:     "101",
			Disposition:  ops.CompletionDispositionConfirmed,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:        "starting process instances",
			CoreResource: "process instance(s)",
			Total:        2,
			Identity:     "102",
			Disposition:  ops.CompletionDispositionConfirmed,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:        "walking process-instance families",
			CoreResource: "process instance(s)",
			Total:        1,
			Identity:     "101",
			Disposition:  ops.CompletionDispositionConfirmed,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:        "cleaning up smoke-test process instances",
			CoreResource: "process-instance tree(s)",
			Total:        1,
			Identity:     "101",
			Disposition:  ops.CompletionDispositionSubmitted,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:        "cleaning up smoke-test process definition",
			CoreResource: "process definition(s)",
			Total:        1,
			Identity:     "pd-88",
			Disposition:  ops.CompletionDispositionConfirmed,
		},
	})

	require.NotContains(t, sink.Starts(), activitysink.Start{
		Message:    "create, 0/1 resource(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "deploying smoke-test fixture, 0/1 deployment(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "deploying smoke-test fixture, 1/1 deployment(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "starting process instances, 0/2 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "starting process instances, 2/2 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "walking process-instance families, 0/1 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "walking process-instance families, 1/1 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "cleaning up smoke-test process instances, 0/1 process-instance tree(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "cleaning up smoke-test process instances, 1/1 process-instance tree(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "cleaning up smoke-test process definition, 0/1 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "cleaning up smoke-test process definition, 1/1 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
}
