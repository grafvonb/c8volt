// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestProcessDefinitionDeleteSemanticProgressRoutesFacadeCompletion verifies
// basic delete process-definition commands use the shared semantic reporter for
// post-confirmation deletion facts.
func TestProcessDefinitionDeleteSemanticProgressRoutesFacadeCompletion(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := newHTTPFallbackActivityCommand(sink)
	progress := newProcessDefinitionDeleteSemanticProgress(cmd, 2)
	progress.Start(2)
	defer progress.Close()

	progress.FacadeProgress(foptions.ProgressEvent{
		Kind: foptions.ProgressEventKindCompletion,
		Completion: &foptions.CompletionProgress{
			Phase:       processDefinitionDeleteCompletionPhase,
			Total:       2,
			Identity:    "pd-1",
			Disposition: foptions.CompletionDispositionConfirmed,
		},
	})

	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "deleting process definitions, 0/2 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "deleting process definitions, 1/2 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
}

// TestOpsPurgeAllProcessDefinitionsProgressKeepsDiscoverySeparate verifies APD
// discovery pages retain their existing progress path while deletion facts start
// a fresh semantic deletion scope.
func TestOpsPurgeAllProcessDefinitionsProgressKeepsDiscoverySeparate(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	sink := &activitysink.Sink{}
	cmd := newHTTPFallbackActivityCommand(sink)
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	deletionProgress := newProcessDefinitionDeleteSemanticProgress(cmd, 0)
	defer deletionProgress.Close()

	request := ops.AllProcessDefinitionsPurgeRequest{}
	configureOpsPurgeAllProcessDefinitionsProgress(cmd, &request, deletionProgress)
	pageCount := int64(2)
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindPage,
		Page: &ops.PageProgress{
			Phase:       "discovering process definitions",
			CurrentPage: 1,
			PageCount:   &pageCount,
			PageSize:    1,
			Seen:        1,
			Selected:    1,
		},
	})
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:       processDefinitionDeleteCompletionPhase,
			Total:       2,
			Identity:    "pd-1",
			Disposition: ops.CompletionDispositionConfirmed,
		},
	})

	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "discovering process definitions, page 1/2, 1 seen",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "deleting process definitions, 0/2 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "deleting process definitions, 1/2 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	})
}
