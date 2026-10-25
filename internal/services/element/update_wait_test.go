// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestElementVariableUpdatesWaitEvidence requires complete local values for every
// requested name, including names excluded from mutation as previously unchanged.
func TestElementVariableUpdatesWaitEvidence(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"delayed", "parent only", "truncated", "unchanged drift", "wrong owner", "wrong tenant", "missing identity", "late read failure", "canceled read", "timeout", "backoff cancellation"} {
		t.Run(mode, func(t *testing.T) {
			plan := elementValidationPlan(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cfg := config.New()
			cfg.App.Backoff.InitialDelay = time.Millisecond
			cfg.App.Backoff.MaxRetries = 2
			if mode == "timeout" {
				cfg.App.Backoff.Timeout = 5 * time.Millisecond
				cfg.App.Backoff.InitialDelay = time.Second
				cfg.App.Backoff.MaxRetries = 0
			}
			if mode == "backoff cancellation" {
				cfg.App.Backoff.InitialDelay = time.Second
			}
			reads, writes := 0, 0
			vars := elementExecutionVariables{write: func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
				writes++
				return acceptedElementWrite(), nil
			}, read: func(ctx context.Context, q d.ScopeVariableQuery, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				reads++
				require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
				require.Equal(t, "P-"+q.ScopeKey, q.ProcessInstanceKey)
				require.Equal(t, q.ScopeKey, *q.ExpectedTenantId)
				rows := elementVerificationRows(q, plan.RequestedValues)
				switch mode {
				case "delayed":
					if reads%2 == 1 {
						return nil, nil
					}
				case "parent only":
					for i := range rows {
						rows[i].ScopeKey = "parent"
					}
				case "truncated":
					for i := range rows {
						rows[i].APITruncated = true
					}
				case "unchanged drift":
					for i := range rows {
						if rows[i].Name == "same" {
							rows[i].Value = `false`
						}
					}
				case "wrong owner":
					rows[0].ProcessInstanceKey = "called-root"
				case "wrong tenant":
					rows[0].TenantId = "discovery"
				case "missing identity":
					rows[0].ScopeKey = ""
				case "late read failure":
					if reads%2 == 0 {
						return nil, d.ErrForbidden
					}
					return nil, nil
				case "canceled read":
					cancel()
				case "timeout":
					return nil, nil
				case "backoff cancellation":
					time.AfterFunc(5*time.Millisecond, cancel)
					return nil, nil
				}
				return rows, nil
			}}
			result, err := NewVariableUpdates(nil, vars, cfg).ExecuteElementVariableUpdates(ctx, plan, 1)
			require.Equal(t, 2, writes)
			require.Len(t, result.Items, 2)
			if mode == "delayed" {
				require.NoError(t, err)
				require.Equal(t, 4, reads)
				for _, item := range result.Items {
					require.True(t, item.OK())
					require.Equal(t, d.ElementVariableUpdateStatusConfirmed, item.Status)
				}
				return
			}
			require.Error(t, err)
			if mode == "late read failure" {
				require.ErrorIs(t, err, d.ErrForbidden)
			}
			if mode == "canceled read" || mode == "backoff cancellation" {
				require.ErrorIs(t, err, context.Canceled)
			}
			if mode == "timeout" {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
			require.Equal(t, d.ElementVariableUpdateStatusConfirmationFailed, result.Items[0].Status)
			require.Equal(t, "failed", result.Items[0].ConfirmationStatus)
			for _, item := range result.Items {
				require.True(t, item.MutationAccepted)
				require.Equal(t, d.ScopeVariableSubmissionAccepted, item.SubmissionStatus)
				require.Equal(t, 204, item.StatusCode)
				require.False(t, item.OK())
				require.NotEmpty(t, item.Error)
			}
		})
	}
}

// TestElementVariableUpdatesWaitFailFast preserves accepted unscheduled confirmations
// and accurately marks a concurrently started read canceled by another read's failure.
func TestElementVariableUpdatesWaitFailFast(t *testing.T) {
	t.Parallel()
	for _, workers := range []int{1, 2} {
		plan := elementValidationPlan(t)
		started := make(chan struct{})
		var reads atomic.Int32
		vars := elementExecutionVariables{write: func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
			return acceptedElementWrite(), nil
		}, read: func(ctx context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			reads.Add(1)
			if q.ScopeKey == "E2" {
				if workers == 2 {
					<-started
				}
				return nil, d.ErrNotFound
			}
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		}}
		result, err := NewVariableUpdates(nil, vars, nil).ExecuteElementVariableUpdates(context.Background(), plan, workers, services.WithFailFast())
		require.ErrorIs(t, err, d.ErrNotFound)
		require.Equal(t, int32(workers), reads.Load())
		require.Equal(t, "failed", result.Items[0].ConfirmationStatus)
		if workers == 1 {
			require.Equal(t, "not_started", result.Items[1].ConfirmationStatus)
			require.Equal(t, d.ElementVariableUpdateStatusSubmitted, result.Items[1].Status)
		} else {
			require.Equal(t, "failed", result.Items[1].ConfirmationStatus)
			require.ErrorIs(t, err, context.Canceled)
		}
		for _, item := range result.Items {
			require.True(t, item.MutationAccepted)
			require.False(t, item.OK())
			require.NotEmpty(t, item.Error)
		}
	}
}
