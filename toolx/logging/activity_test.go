package logging

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
