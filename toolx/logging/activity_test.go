// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package logging

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/stretchr/testify/require"
)

// TestStartActivity_UsesContextSink verifies context-bound activity sinks receive balanced start and stop calls.
func TestStartActivity_UsesContextSink(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	stop := StartActivity(ToActivityContext(context.Background(), sink), "working")
	stop()

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"working"}, msgs)
}

// TestActivityWriter_ClearsIndicatorBeforeNormalOutput verifies normal log output is not mixed with spinner frames.
func TestActivityWriter_ClearsIndicatorBeforeNormalOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)
	w.delay = 1 * time.Millisecond
	w.interval = 1 * time.Millisecond

	w.StartActivity("waiting")
	time.Sleep(5 * time.Millisecond)
	_, err := w.Write([]byte("INFO done\n"))
	require.NoError(t, err)
	w.StopActivity()

	out := buf.String()
	require.Contains(t, out, "waiting")
	require.Contains(t, out, "INFO done\n")
	require.NotContains(t, out, "INFO done\n/")
	require.NotContains(t, out, "INFO done\n|")
}

// TestActivityWriter_NestedActivityScopesRequireMatchingStops verifies nested activity scopes keep the indicator alive until all scopes finish.
func TestActivityWriter_NestedActivityScopesRequireMatchingStops(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)
	w.delay = 1 * time.Millisecond
	w.interval = 1 * time.Millisecond

	w.StartActivity("outer")
	w.StartActivity("inner")
	time.Sleep(5 * time.Millisecond)
	w.StopActivity()
	time.Sleep(5 * time.Millisecond)
	w.StopActivity()

	out := buf.String()
	require.Contains(t, out, "outer")
}
