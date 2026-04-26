// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package poller

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

type fakeActivitySink struct {
	mu      sync.Mutex
	started int
	stopped int
	msgs    []string
}

func (s *fakeActivitySink) StartActivity(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started++
	s.msgs = append(s.msgs, msg)
}

func (s *fakeActivitySink) StopActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped++
}

func TestWaitForCompletion_UsesSharedActivitySink(t *testing.T) {
	t.Parallel()

	sink := &fakeActivitySink{}
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

	sink.mu.Lock()
	defer sink.mu.Unlock()
	require.Equal(t, 1, sink.started)
	require.Equal(t, 1, sink.stopped)
	require.Equal(t, []string{"waiting for completion"}, sink.msgs)
}
