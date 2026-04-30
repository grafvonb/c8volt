// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package poller

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestWaitForCompletion_UsesSharedActivitySink verifies polling progress uses the activity context sink.
func TestWaitForCompletion_UsesSharedActivitySink(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	ctx := logging.ToActivityContext(context.Background(), sink)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	attempts := 0

	err := WaitForCompletion(ctx, log, 5*time.Second, false, func(ctx context.Context) (JobPollStatus, error) {
		attempts++
		if attempts == 1 {
			return JobPollStatus{Success: false, Message: "still waiting"}, nil
		}
		return JobPollStatus{Success: true, Message: "done"}, nil
	})
	require.NoError(t, err)

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"waiting for completion"}, msgs)
}
