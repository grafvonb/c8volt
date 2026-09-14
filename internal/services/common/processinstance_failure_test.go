// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestCancellationConfirmationFailureReasons verifies stop-reason precedence
// across joined waits while preserving the original error text and causes.
func TestCancellationConfirmationFailureReasons(t *testing.T) {
	t.Parallel()
	lookup := &d.ProcessInstanceWaitFailure{Reason: "failed during lookup", Err: d.ErrForbidden}
	attempts := &d.ProcessInstanceWaitFailure{Reason: "exhausted polling attempts", Err: errors.New("exceeded max_retries (1)")}
	deadline := &d.ProcessInstanceWaitFailure{Reason: "failed during lookup", Err: context.DeadlineExceeded}
	for _, tc := range []struct {
		err  error
		want string
	}{
		{lookup, "failed during lookup"},
		{&d.ProcessInstanceWaitFailure{Reason: "failed during lookup", Err: d.ErrGatewayTimeout}, "failed during lookup"},
		{attempts, "exhausted polling attempts"},
		{deadline, "timed out"},
		{context.Canceled, "was canceled"},
		{errors.Join(lookup, deadline), "failed during lookup"},
		{errors.Join(deadline, lookup), "failed during lookup"},
		{errors.Join(context.Canceled, lookup), "failed during lookup"},
		{errors.Join(deadline, attempts), "exhausted polling attempts"},
	} {
		err := fmt.Errorf("cancel wait: %w", tc.err)
		failure := NewProcessInstanceCancellationConfirmationFailure("cancel", "root", []string{"root"}, time.Second, err)
		require.Equal(t, tc.want, failure.FailureReason)
		require.Equal(t, err.Error(), failure.Error())
		require.ErrorIs(t, failure, tc.err)
	}
}
