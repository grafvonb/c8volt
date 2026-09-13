// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"io"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
)

const (
	diagnosticBenchmarkSmallBodyBytes     = 32
	diagnosticBenchmarkStreamingBodyBytes = 8 << 20
)

// BenchmarkAPIDiagnostics compares the disabled pass-through with enabled
// metadata observation for both small and streamed response bodies.
func BenchmarkAPIDiagnostics(b *testing.B) {
	for _, enabled := range []bool{false, true} {
		mode := "disabled"
		if enabled {
			mode = "enabled"
		}
		for _, size := range []struct {
			name  string
			bytes int64
		}{
			{name: "small-body", bytes: diagnosticBenchmarkSmallBodyBytes},
			{name: "streaming-body", bytes: diagnosticBenchmarkStreamingBodyBytes},
		} {
			b.Run(mode+"/"+size.name, func(b *testing.B) {
				benchmarkAPIDiagnostics(b, enabled, size.bytes)
			})
		}
	}
}

func benchmarkAPIDiagnostics(b *testing.B, enabled bool, bodyBytes int64) {
	b.Helper()
	transport := &DiagnosticsTransport{base: diagnosticBenchmarkTransport{bodyBytes: bodyBytes}}
	if enabled {
		log := logging.New(logging.LoggerConfig{Writer: io.Discard, Level: "debug", Format: "plain"})
		transport.collector = newDiagnosticCollector(config.New(), log)
		if transport.collector == nil {
			b.Fatal("enabled benchmark requires an admitted diagnostic collector")
		}
	}

	b.ReportAllocs()
	b.SetBytes(bodyBytes)
	b.ResetTimer()
	for b.Loop() {
		req, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/v2/benchmark", nil)
		if err != nil {
			b.Fatal(err)
		}
		response, err := transport.RoundTrip(req)
		if err != nil {
			b.Fatal(err)
		}
		if _, err = io.Copy(io.Discard, response.Body); err != nil {
			b.Fatal(err)
		}
		if err = response.Body.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

type diagnosticBenchmarkTransport struct {
	bodyBytes int64
}

func (transport diagnosticBenchmarkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(io.LimitReader(diagnosticBenchmarkZeroReader{}, transport.bodyBytes)),
		Request:    req,
	}, nil
}

type diagnosticBenchmarkZeroReader struct{}

func (diagnosticBenchmarkZeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}
