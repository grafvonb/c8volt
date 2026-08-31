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

// TestOpsRepairSemanticProgressRoutesCompletion verifies repair completion
// facts drive a workflow-priority semantic activity while unrelated phases are
// ignored.
func TestOpsRepairSemanticProgressRoutesCompletion(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	sink := &activitysink.Sink{}
	cmd := newHTTPFallbackActivityCommand(sink)
	request := ops.RepairRequest{}
	progress := configureOpsRepairProgress(cmd, &request)
	defer progress.Close()

	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:       "loading process-instance repair incidents",
			Total:       1,
			Identity:    "2251799813685251",
			Disposition: ops.CompletionDispositionConfirmed,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:       opsRepairCompletionPhase,
			Total:       2,
			Identity:    "2251799813685249",
			Disposition: ops.CompletionDispositionSubmitted,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:       opsRepairCompletionPhase,
			Total:       2,
			Identity:    "2251799813685250",
			Disposition: ops.CompletionDispositionConfirmed,
		},
	})

	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "repairing incidents, 0/2 incident(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "repairing incidents, 1/2 incident(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "repairing incidents, 2/2 incident(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.NotContains(t, sink.Starts(), activitysink.Start{
		Message:    "loading process-instance repair incidents, 0/1 incident(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
}
