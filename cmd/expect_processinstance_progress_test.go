// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestExpectProcessInstanceSemanticProgressRoutesMultiKeyCompletions(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	reporter := newExpectProcessInstanceSemanticProgress(cmd, 2)
	defer reporter.Close()
	opts := appendExpectProcessInstanceProgressOption(nil, reporter)
	progress := foptions.ApplyFacadeOptions(opts).Progress
	require.NotNil(t, progress)

	progress(foptions.ProgressEvent{
		Kind: foptions.ProgressEventKindCompletion,
		Completion: &foptions.CompletionProgress{
			Phase:        expectProcessInstanceCompletionPhase,
			CoreResource: "process instance(s)",
			Total:        2,
			Identity:     "123",
			Disposition:  foptions.CompletionDispositionConfirmed,
		},
	})
	progress(foptions.ProgressEvent{
		Kind: foptions.ProgressEventKindCompletion,
		Completion: &foptions.CompletionProgress{
			Phase:        expectProcessInstanceCompletionPhase,
			CoreResource: "process instance(s)",
			Total:        2,
			Identity:     "124",
			Disposition:  foptions.CompletionDispositionConfirmed,
		},
	})

	require.Equal(t, []activitysink.Update{
		{
			Message:    "waiting for process-instance expectations, 1/2 process instance(s)",
			Importance: logging.ActivityImportanceWorkflow,
		},
		{
			Message:    "waiting for process-instance expectations, 2/2 process instance(s)",
			Importance: logging.ActivityImportanceWorkflow,
		},
	}, sink.PriorityUpdates())
}
