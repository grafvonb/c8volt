// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package pool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
