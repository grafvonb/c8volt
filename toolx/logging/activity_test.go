// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package logging

import (
	"bytes"
	"context"
	"strings"
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

func TestUpdateActivity_UsesContextUpdater(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	ctx := ToActivityContext(context.Background(), sink)

	UpdateActivity(ctx, "checked 10")

	require.Equal(t, []string{"checked 10"}, sink.Updates())
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

func TestActivityWriter_UpdateActivityRefreshesMessage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)
	w.delay = 1 * time.Millisecond
	w.interval = 1 * time.Millisecond

	w.StartActivity("waiting")
	time.Sleep(5 * time.Millisecond)
	w.UpdateActivity("checked 10")
	time.Sleep(2 * time.Millisecond)
	w.StopActivity()

	out := buf.String()
	require.Contains(t, out, "waiting")
	require.Contains(t, out, "checked 10")
}

func TestActivityWriter_ClearsLongUpdatedMessageBeforeNormalOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)
	w.delay = 1 * time.Millisecond
	w.interval = 1 * time.Millisecond
	longMessage := "orphan search: page 1 checking 1000 child process instance(s) for missing parents; checked 0, found 0 orphan child process instance(s)"

	w.StartActivity("waiting")
	time.Sleep(5 * time.Millisecond)
	w.UpdateActivity(longMessage)
	_, err := w.Write([]byte("found: 0\n"))
	require.NoError(t, err)
	w.StopActivity()

	out := buf.String()
	require.Contains(t, out, longMessage)
	clearBeforeOutput := "\r" + strings.Repeat(" ", len("| "+longMessage)) + "\rfound: 0\n"
	require.Contains(t, out, clearBeforeOutput)
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
