package logging

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeActivitySink struct {
	started int
	stopped int
	msgs    []string
}

func (s *fakeActivitySink) StartActivity(msg string) {
	s.started++
	s.msgs = append(s.msgs, msg)
}

func (s *fakeActivitySink) StopActivity() {
	s.stopped++
}

func TestStartActivity_UsesContextSink(t *testing.T) {
	t.Parallel()

	sink := &fakeActivitySink{}
	stop := StartActivity(ToActivityContext(context.Background(), sink), "working")
	stop()

	require.Equal(t, 1, sink.started)
	require.Equal(t, 1, sink.stopped)
	require.Equal(t, []string{"working"}, sink.msgs)
}

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
