// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsCollectorGating verifies disabled or INFO-filtered invocations install no collector.
func TestAPIDiagnosticsCollectorGating(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		logger  *slog.Logger
		verbose bool
	}{
		{name: "verbose disabled", logger: logging.New(logging.LoggerConfig{Writer: &bytes.Buffer{}, Level: "info", Format: "plain"})},
		{name: "info filtered", logger: logging.New(logging.LoggerConfig{Writer: &bytes.Buffer{}, Level: "warn", Format: "plain"}), verbose: true},
		{name: "logger absent", verbose: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Nil(t, newDiagnosticCollector(config.New(), test.logger, test.verbose))
		})
	}
}

// TestAPIDiagnosticsCollectorCopiesPrivateContext verifies later config changes cannot alter retained redaction inputs.
func TestAPIDiagnosticsCollectorCopiesPrivateContext(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	cfg := config.New()
	cfg.ActiveProfile = "prod-secret-value"
	cfg.App.Tenant = "tenant-secret-value"
	cfg.Auth.OAuth2.ClientSecret = "secret-value"
	collector := newDiagnosticCollector(cfg, logging.New(logging.LoggerConfig{Writer: &output, Level: "info", Format: "plain"}), true)
	require.NotNil(t, collector)

	cfg.ActiveProfile = "changed-profile"
	cfg.App.Tenant = "changed-tenant"
	cfg.Auth.OAuth2.ClientSecret = "changed-secret"
	collector.start(&http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/v2/items/secret-value"}}).finish(nil)

	require.Contains(t, output.String(), "GET /v2/items/[REDACTED]: total=0s profile=prod-[REDACTED] tenant=tenant-[REDACTED]")
	require.NotContains(t, output.String(), "secret-value")
	require.NotContains(t, output.String(), "changed-")
}

// TestAPIDiagnosticsCollectorSequencesAndIsolation verifies each invocation owns its logger, context and sequence.
func TestAPIDiagnosticsCollectorSequencesAndIsolation(t *testing.T) {
	t.Parallel()

	const exchanges = 64
	newCollector := func(profile string) (*diagnosticCollector, *bytes.Buffer) {
		var output bytes.Buffer
		cfg := config.New()
		cfg.ActiveProfile = profile
		collector := newDiagnosticCollector(cfg, logging.New(logging.LoggerConfig{Writer: &output, Level: "info", Format: "plain"}), true)
		require.NotNil(t, collector)
		return collector, &output
	}

	first, firstOutput := newCollector("first")
	second, secondOutput := newCollector("second")
	var workers sync.WaitGroup
	workers.Add(exchanges)
	for index := range exchanges {
		go func() {
			defer workers.Done()
			exchange := first.start(&http.Request{
				Method: http.MethodGet,
				URL:    &url.URL{Path: "/v2/items"},
			})
			exchange.finish(func(record *diagnosticRecord) {
				record.total = time.Duration(index) * time.Millisecond
			})
		}()
	}
	workers.Wait()

	second.start(&http.Request{Method: http.MethodHead, URL: &url.URL{Path: "/v2/topology"}}).finish(nil)

	firstLines := strings.Split(strings.TrimSpace(firstOutput.String()), "\n")
	require.Len(t, firstLines, exchanges)
	seen := make(map[string]struct{}, exchanges)
	for _, line := range firstLines {
		require.Contains(t, line, " GET /v2/items:")
		require.Contains(t, line, " profile=first")
		require.NotContains(t, line, "profile=second")
		var sequence string
		for _, field := range strings.Fields(line) {
			if strings.HasPrefix(field, "#") {
				sequence = field
				break
			}
		}
		require.NotEmpty(t, sequence)
		_, duplicate := seen[sequence]
		require.False(t, duplicate)
		seen[sequence] = struct{}{}
	}
	for expected := 1; expected <= exchanges; expected++ {
		require.Contains(t, seen, "#"+strconv.Itoa(expected))
	}

	require.Contains(t, secondOutput.String(), "INFO api #1 HEAD /v2/topology: total=0s profile=second\n")
}

// TestAPIDiagnosticsCollectorFreezesAndEmitsAfterUnlock verifies concurrent terminal events emit one immutable record without holding the state lock.
func TestAPIDiagnosticsCollectorFreezesAndEmitsAfterUnlock(t *testing.T) {
	t.Parallel()

	writer := &diagnosticCallbackWriter{}
	collector := newDiagnosticCollector(config.New(), logging.New(logging.LoggerConfig{Writer: writer, Level: "info", Format: "plain"}), true)
	require.NotNil(t, collector)
	exchange := collector.start(&http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/v2/topology"}})
	writer.callback = func() {
		writer.lateUpdate.Store(exchange.update(func(record *diagnosticRecord) {
			record.host = "late.example"
		}))
	}

	const terminalEvents = 32
	var workers sync.WaitGroup
	workers.Add(terminalEvents)
	for range terminalEvents {
		go func() {
			defer workers.Done()
			exchange.finish(func(record *diagnosticRecord) {
				record.total = time.Second
				record.host = "frozen.example"
			})
		}()
	}
	workers.Wait()

	require.False(t, writer.lateUpdate.Load())
	require.Contains(t, writer.String(), "INFO api #1 GET /v2/topology: total=1s host=frozen.example\n")
	require.False(t, exchange.update(func(record *diagnosticRecord) { record.host = "later.example" }))
}

// diagnosticCallbackWriter attempts a late state update during logging to expose lock-order regressions.
type diagnosticCallbackWriter struct {
	mu         sync.Mutex
	output     bytes.Buffer
	callback   func()
	callbackDo sync.Once
	lateUpdate atomic.Bool
}

// Write runs the callback synchronously before retaining the complete log write.
func (writer *diagnosticCallbackWriter) Write(value []byte) (int, error) {
	writer.callbackDo.Do(func() {
		if writer.callback != nil {
			writer.callback()
		}
	})
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.output.Write(value)
}

// String returns a synchronized copy of the captured output.
func (writer *diagnosticCallbackWriter) String() string {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.output.String()
}
