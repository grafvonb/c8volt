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

// TestActivityWriter_DisabledSuppressesActivityOutput verifies root-level gating can silence the shared writer.
func TestActivityWriter_DisabledSuppressesActivityOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, false)

	w.StartActivity("waiting")
	w.UpdateActivity("checked 10")
	stop := w.StartActivityWithImportance("workflow", ActivityImportanceWorkflow)
	w.UpdateActivityWithImportance("workflow checked 20", ActivityImportanceWorkflow)
	stop()
	w.StopActivity()

	require.Empty(t, buf.String())
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

func TestActivityWriter_TruncatesIndicatorToConfiguredWidth(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)
	w.delay = 1 * time.Millisecond
	w.interval = 1 * time.Millisecond
	w.maxWidth = 20
	longMessage := "page size: 1000, current page: 1000, total so far: 16000, more matches: yes"

	w.StartActivity(longMessage)
	time.Sleep(5 * time.Millisecond)
	w.StopActivity()

	for _, segment := range strings.Split(buf.String(), "\r") {
		require.LessOrEqual(t, len(segment), 20)
	}
	require.Contains(t, buf.String(), "...")
	require.NotContains(t, buf.String(), "more matches")
}

func TestActivityWriter_PriorityOrderingKeepsHigherImportanceVisible(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)

	stopWorkflow := w.StartActivityWithImportance("workflow progress", ActivityImportanceWorkflow)
	w.tick()
	stopHTTP := w.StartActivityWithImportance("loading process instance", ActivityImportanceHTTP)

	require.Equal(t, "| workflow progress", lastActivityLine(buf.String()))

	stopHTTP()
	stopWorkflow()
}

func TestActivityWriter_EqualPrioritySelectsNewestScope(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)

	stopFirst := w.StartActivityWithImportance("first wait", ActivityImportanceWait)
	w.tick()
	stopSecond := w.StartActivityWithImportance("second wait", ActivityImportanceWait)

	require.Equal(t, "/ second wait", lastActivityLine(buf.String()))

	stopSecond()
	stopFirst()
}

func TestActivityWriter_StopFallsBackToRemainingHighestScope(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)

	stopWait := w.StartActivityWithImportance("waiting for completion", ActivityImportanceWait)
	w.tick()
	stopWorkflow := w.StartActivityWithImportance("workflow progress", ActivityImportanceWorkflow)
	w.tick()
	stopWorkflow()

	require.Equal(t, `\ waiting for completion`, lastActivityLine(buf.String()))

	stopWait()
}

func TestActivityWriter_ScopedStopIsIdempotent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)

	stopBatch := w.StartActivityWithImportance("batch progress", ActivityImportanceBatch)
	w.tick()
	stopBatch()
	stopBatch()

	require.Contains(t, buf.String(), "batch progress")
}

func TestActivityWriter_PriorityUpdateTargetsMatchingScope(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)

	stopWait := w.StartActivityWithImportance("waiting", ActivityImportanceWait)
	w.tick()
	stopWorkflow := w.StartActivityWithImportance("workflow", ActivityImportanceWorkflow)
	w.tick()
	w.UpdateActivityWithImportance("waiting 10/20", ActivityImportanceWait)
	w.tick()

	require.Equal(t, `\ workflow`, lastActivityLine(buf.String()))

	w.UpdateActivityWithImportance("workflow 5/10", ActivityImportanceWorkflow)

	require.Equal(t, "| workflow 5/10", lastActivityLine(buf.String()))

	stopWorkflow()

	require.Equal(t, "/ waiting 10/20", lastActivityLine(buf.String()))

	stopWait()
}

// TestActivityWriter_WorkflowRemainsVisibleAboveWaitAndHTTP verifies the documented US1 hierarchy examples.
func TestActivityWriter_WorkflowRemainsVisibleAboveWaitAndHTTP(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)

	stopWorkflow := w.StartActivityWithImportance("deleting process-instance trees 48/800", ActivityImportanceWorkflow)
	w.tick()
	stopWait := w.StartActivityWithImportance("waiting for pi 123 state", ActivityImportanceWait)
	w.tick()
	stopHTTP := w.StartActivityWithImportance("loading process instance", ActivityImportanceHTTP)
	w.tick()

	require.Equal(t, "- deleting process-instance trees 48/800", lastActivityLine(buf.String()))

	stopHTTP()
	stopWait()
	stopWorkflow()
}

// TestActivityWriter_WaitFallsBackAboveHTTPAfterWorkflowStops verifies wait scopes outrank HTTP fallback after workflow progress ends.
func TestActivityWriter_WaitFallsBackAboveHTTPAfterWorkflowStops(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := newActivityWriter(&buf, true)

	stopHTTP := w.StartActivityWithImportance("loading process instance", ActivityImportanceHTTP)
	w.tick()
	stopWorkflow := w.StartActivityWithImportance("analyzing process instances", ActivityImportanceWorkflow)
	w.tick()
	stopWait := w.StartActivityWithImportance("waiting for pi 123 state", ActivityImportanceWait)
	w.tick()
	stopWorkflow()

	require.Equal(t, "| waiting for pi 123 state", lastActivityLine(buf.String()))

	stopWait()
	stopHTTP()
}

func TestStartActivityWithImportance_UsesPriorityContextSink(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	ctx := ToActivityContext(context.Background(), sink)

	stop := StartActivityWithImportance(ctx, "workflow", ActivityImportanceWorkflow)
	stop()
	stop()

	require.Equal(t, []activitysink.Start{{
		Message:    "workflow",
		Importance: ActivityImportanceWorkflow,
	}}, sink.Starts())
	require.Equal(t, 1, sink.Stopped())
}

func TestUpdateActivityWithImportance_UsesPriorityContextUpdater(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	ctx := ToActivityContext(context.Background(), sink)

	UpdateActivityWithImportance(ctx, "workflow 1/2", ActivityImportanceWorkflow)

	require.Equal(t, []activitysink.Update{{
		Message:    "workflow 1/2",
		Importance: ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}

func lastActivityLine(out string) string {
	segments := strings.Split(out, "\r")
	for i := len(segments) - 1; i >= 0; i-- {
		segment := strings.TrimSpace(segments[i])
		if segment != "" {
			return segment
		}
	}
	return ""
}
