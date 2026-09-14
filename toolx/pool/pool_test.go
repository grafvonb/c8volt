// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package pool

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

func TestExecuteSliceWorkerDebugLogging(t *testing.T) {
	for _, tt := range []struct {
		name    string
		jobs    int
		workers int
		level   slog.Level
		want    string
	}{
		{"debug", 3, 2, slog.LevelDebug, "DEBUG worker pool: jobs=3 workers=2\n"},
		{"clamped", 2, 8, slog.LevelDebug, "DEBUG worker pool: jobs=2 workers=2\n"},
		{"normalized", 2, 0, slog.LevelDebug, "DEBUG worker pool: jobs=2 workers=1\n"},
		{"info", 3, 2, slog.LevelInfo, ""},
		{"empty", 0, 2, slog.LevelDebug, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			log := slog.New(logging.NewPlainHandler(&output, tt.level))
			ctx := logging.ToContext(context.Background(), log)
			results, err := ExecuteSlice(ctx, make([]int, tt.jobs), tt.workers, false, func(_ context.Context, _ int, i int) (int, error) {
				return i, nil
			})
			require.NoError(t, err)
			require.Len(t, results, tt.jobs)
			require.Equal(t, tt.want, output.String())
		})
	}
}

// TestStaggerWorkerStartKeepsSmallPoolsImmediate verifies common small worker
// pools avoid artificial startup delay.
func TestStaggerWorkerStartKeepsSmallPoolsImmediate(t *testing.T) {
	require.NoError(t, staggerWorkerStart(context.Background(), 1, workerStartStaggerThreshold))
}

// TestStaggerWorkerStartDelaysLargePools verifies larger worker pools are
// spread out enough to avoid a synchronized initial request burst.
func TestStaggerWorkerStartDelaysLargePools(t *testing.T) {
	started := time.Now()
	require.NoError(t, staggerWorkerStart(context.Background(), 4, 8))
	require.GreaterOrEqual(t, time.Since(started), 20*time.Millisecond)
}

// TestStaggerWorkerStartRespectsContext avoids keeping canceled bulk work alive
// while workers are waiting for their startup stagger.
func TestStaggerWorkerStartRespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.ErrorIs(t, staggerWorkerStart(ctx, 4, 8), context.Canceled)
}
