// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
)

// BenchmarkRequestRate compares identical fake dispatches with no controls,
// disabled controls and measurement-only reporting, including allocation costs.
func BenchmarkRequestRate(b *testing.B) {
	for _, mode := range []string{"pass-through", "zero-reporting-off", "zero-reporting-on"} {
		b.Run(mode, func(b *testing.B) {
			response := &http.Response{StatusCode: 200, Body: http.NoBody}
			base := requestRateRoundTripFunc(func(*http.Request) (*http.Response, error) { return response, nil })
			var transport http.RoundTripper = base
			if mode != "pass-through" {
				interval := "0s"
				if mode == "zero-reporting-on" {
					interval = "1ms"
				}
				svc, err := New(&config.Config{HTTP: config.HTTP{Timeout: "1s", RequestRateReportInterval: interval}}, slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})), WithInvocationContext(context.Background()))
				if err != nil {
					b.Fatal(err)
				}
				defer svc.Close()
				// Compare the optional control boundary with the same fake delegate,
				// excluding unrelated read-retry, logging and net/http allocations.
				if svc.requestRate != nil {
					svc.requestRate.base = base
					transport = svc.requestRate
				}
			}
			request, _ := http.NewRequest(http.MethodPost, "https://benchmark.invalid", nil)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err := transport.RoundTrip(request)
				if err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
		})
	}
}
