// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ferrors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/stretchr/testify/require"
)

// TestRequestRateNormalization preserves inspectable unsent causes and context
// identity while keeping class priority and machine-facing error text unchanged.
func TestRequestRateNormalization(t *testing.T) {
	t.Parallel()
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			service, err := httpc.New(&config.Config{HTTP: config.HTTP{RequestRateReportInterval: "5s", RequestRate: 1, Timeout: "1s"}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			wantCause, wantClass, wantExit := context.DeadlineExceeded, ClassTimeout, exitcode.Timeout
			if canceled {
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
				wantCause, wantClass, wantExit = context.Canceled, ClassLocalPrecondition, exitcode.Error
			}
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://never-dispatched.invalid", nil)
			require.NoError(t, err)
			_, local := service.Client().Transport.RoundTrip(request)
			require.True(t, httpc.IsRequestRateWaitError(local))
			normalized := Normalize(FromDomain(fmt.Errorf("batch: %w", errors.Join(local, local))))
			require.True(t, httpc.IsRequestRateWaitError(normalized))
			require.ErrorIs(t, normalized, wantCause)
			require.Equal(t, wantClass, Classify(normalized))
			require.Equal(t, wantExit, ExitCode(normalized))
			require.Contains(t, normalized.Error(), "batch: "+local.Error()+"\n"+local.Error())
			require.Same(t, normalized, Normalize(normalized))
			// Inspecting local evidence must not override an established outer class.
			priority := Normalize(FromDomain(errors.Join(domain.ErrBadRequest, local)))
			require.Equal(t, ClassInvalidInput, Classify(priority))
			require.True(t, httpc.IsRequestRateWaitError(priority))
		})
	}
}
