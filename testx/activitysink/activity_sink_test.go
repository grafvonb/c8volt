// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package activitysink

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSinkConcurrentPriorityRecording verifies command progress tests can safely capture concurrent priority-aware activity events.
func TestSinkConcurrentPriorityRecording(t *testing.T) {
	t.Parallel()

	sink := &Sink{}
	const scopes = 16

	var wg sync.WaitGroup
	stops := make([]func(), scopes)
	for i := range scopes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			stops[i] = sink.StartActivityWithImportance(fmt.Sprintf("workflow %02d", i), i%4)
			sink.UpdateActivityWithImportance(fmt.Sprintf("workflow %02d done", i), i%4)
		}(i)
	}
	wg.Wait()

	for _, stop := range stops {
		require.NotNil(t, stop)
		stop()
		stop()
	}

	require.Equal(t, scopes, sink.Started())
	require.Equal(t, scopes, sink.Stopped())
	require.Len(t, sink.Starts(), scopes)
	require.Len(t, sink.PriorityUpdates(), scopes)
}

// TestSinkSnapshotsReturnCopies verifies callers cannot mutate captured sink state outside the sink mutex.
func TestSinkSnapshotsReturnCopies(t *testing.T) {
	t.Parallel()

	sink := &Sink{}
	stop := sink.StartActivityWithImportance("workflow", 3)
	sink.UpdateActivityWithImportance("workflow 1/2", 3)
	stop()

	starts := sink.Starts()
	starts[0].Message = "mutated"
	updates := sink.PriorityUpdates()
	updates[0].Message = "mutated"
	messages := sink.Messages()
	messages[0] = "mutated"

	require.Equal(t, []Start{{Message: "workflow", Importance: 3}}, sink.Starts())
	require.Equal(t, []Update{{Message: "workflow 1/2", Importance: 3}}, sink.PriorityUpdates())
	require.Equal(t, []string{"workflow"}, sink.Messages())
}
