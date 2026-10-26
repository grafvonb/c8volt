// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// suspensionAPIStub explicitly owns both new methods; other APIs must never be used.
type suspensionAPIStub struct {
	API
	guard error
	call  func(context.Context, string, ...services.CallOption) (d.ProcessInstanceSuspensionResult, error)
}

// ValidateProcessInstanceSuspension exposes local capability without remote calls.
func (s suspensionAPIStub) ValidateProcessInstanceSuspension() error { return s.guard }

// SuspendProcessInstance delegates to an explicit test callback, including previews.
func (s suspensionAPIStub) SuspendProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
	return s.call(ctx, key, opts...)
}

// suspensionKeys creates valid independent targets in input order.
func suspensionKeys(n int) typex.Keys {
	keys := make(typex.Keys, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("%016d", i+1)
	}
	return keys
}

// suspensionItem supplies evidence appropriate to a test-owned preview or mutation.
func suspensionItem(key string, preview bool) d.ProcessInstanceSuspensionResult {
	r := d.ProcessInstanceSuspensionResult{Key: key, Status: d.ProcessInstanceSuspensionStatusAccepted, Submission: d.ProcessInstanceSuspensionSubmissionAccepted, Confirmation: d.ProcessInstanceSuspensionConfirmationNotRequested, StatusCode: 204}
	if preview {
		r.Status = d.ProcessInstanceSuspensionStatusPreview
		r.Submission = d.ProcessInstanceSuspensionSubmissionNotSubmitted
		r.StatusCode = 0
	}
	return r
}

// TestProcessInstanceSuspensionCollectionInput checks capability-first guards,
// complete normalization, no-op arrays, cancellation and no-I/O preview behavior.
func TestProcessInstanceSuspensionCollectionInput(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"unsupported empty", "unsupported malformed", "invalid", "empty", "preview", "preview no wait", "normalized", "canceled", "canceled after preview", "invalid timing"} {
		t.Run(mode, func(t *testing.T) {
			keys := suspensionKeys(2)
			selected := append(typex.Keys(nil), keys...)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opts := []services.CallOption{}
			previews, mutations := 0, 0
			api := suspensionAPIStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
				preview := services.ApplyCallOptions(opts).DryRun
				if preview {
					previews++
					if mode == "canceled after preview" && key == keys[1] {
						cancel()
					}
				} else {
					mutations++
				}
				if mode == "invalid timing" && key == keys[1] {
					return d.ProcessInstanceSuspensionResult{}, d.ErrValidation
				}
				return suspensionItem(key, preview), nil
			}}
			switch mode {
			case "unsupported empty":
				api.guard = d.ErrUnsupported
				selected = nil
			case "unsupported malformed":
				api.guard = d.ErrUnsupported
				selected = typex.Keys{"bad"}
			case "invalid":
				selected = append(selected, "bad")
			case "empty":
				selected = nil
			case "preview":
				opts = append(opts, services.WithDryRun())
			case "preview no wait":
				opts = append(opts, services.WithDryRun(), services.WithNoWait())
			case "normalized":
				selected = typex.Keys{" " + keys[0] + "\n", keys[1], keys[0]}
			case "canceled":
				cancel()
			}
			got, err := SuspendProcessInstances(ctx, api, nil, selected, 1, opts...)
			switch mode {
			case "unsupported empty", "unsupported malformed":
				require.ErrorIs(t, err, d.ErrUnsupported)
				require.Zero(t, previews)
			case "invalid":
				require.ErrorIs(t, err, d.ErrValidation)
				require.Zero(t, previews)
			case "invalid timing":
				require.ErrorIs(t, err, d.ErrValidation)
				require.Equal(t, 2, previews)
			case "canceled":
				require.ErrorIs(t, err, context.Canceled)
				require.Zero(t, previews)
				require.Len(t, got.Items, 2)
			case "canceled after preview":
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, 2, previews)
				require.Len(t, got.Items, 2)
				for _, item := range got.Items {
					require.Equal(t, d.ProcessInstanceSuspensionStatusNotAttempted, item.Status)
					require.NotEmpty(t, item.Error)
				}
			case "empty":
				require.NoError(t, err)
				require.NotNil(t, got.Items)
				require.Empty(t, got.Items)
				require.Zero(t, previews)
			default:
				require.NoError(t, err)
				require.Len(t, got.Items, 2)
				require.Equal(t, keys[0], got.Items[0].Key)
				require.Equal(t, keys[1], got.Items[1].Key)
				require.Equal(t, 2, previews)
			}
			if mode == "normalized" {
				require.Equal(t, 2, mutations)
			} else {
				require.Zero(t, mutations)
			}
			if mode == "preview" || mode == "preview no wait" {
				for _, r := range got.Items {
					require.Equal(t, d.ProcessInstanceSuspensionStatusPreview, r.Status)
					require.Equal(t, d.ProcessInstanceSuspensionSubmissionNotSubmitted, r.Submission)
				}
			}
		})
	}
}

// TestProcessInstanceSuspensionCollectionFailures retains started evidence, joined
// error causes and keyed unstarted slots under fail-fast and caller cancellation.
func TestProcessInstanceSuspensionCollectionFailures(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"mixed", "fail fast", "cancel during", "in flight accepted"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				keys := suspensionKeys(4)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				opts := []services.CallOption{}
				workers := 1
				if mode == "fail fast" || mode == "in flight accepted" {
					opts = append(opts, services.WithFailFast())
				}
				if mode == "in flight accepted" {
					workers = 2
				}
				ready := make(chan struct{})
				started := atomic.Int32{}
				api := suspensionAPIStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
					if services.ApplyCallOptions(opts).DryRun {
						return suspensionItem(key, true), nil
					}
					started.Add(1)
					r := suspensionItem(key, false)
					if key == keys[0] {
						if mode == "in flight accepted" {
							<-ready
						}
						r.Status = d.ProcessInstanceSuspensionStatusFailed
						r.Submission = d.ProcessInstanceSuspensionSubmissionRejected
						r.StatusCode = 409
						r.Error = "conflict"
						return r, d.ErrConflict
					}
					if mode == "in flight accepted" {
						close(ready)
						<-ctx.Done()
						r.Status = d.ProcessInstanceSuspensionStatusInterrupted
						r.Confirmation = d.ProcessInstanceSuspensionConfirmationUnconfirmed
						r.Error = ctx.Err().Error()
						return r, ctx.Err()
					}
					if mode == "cancel during" {
						cancel()
						return r, context.Canceled
					}
					return r, nil
				}}
				got, err := SuspendProcessInstances(ctx, api, nil, keys, workers, opts...)
				require.ErrorIs(t, err, d.ErrConflict)
				require.Len(t, got.Items, len(keys))
				for i, r := range got.Items {
					require.Equal(t, keys[i], r.Key)
				}
				count := 4
				if mode == "fail fast" {
					count = 1
				}
				if mode == "cancel during" || mode == "in flight accepted" {
					count = 2
					require.ErrorIs(t, err, context.Canceled)
				}
				require.Equal(t, int32(count), started.Load())
				for i := count; i < len(keys); i++ {
					require.Equal(t, d.ProcessInstanceSuspensionStatusNotAttempted, got.Items[i].Status)
					require.NotEmpty(t, got.Items[i].Error)
				}
				if count > 1 {
					require.Equal(t, d.ProcessInstanceSuspensionSubmissionAccepted, got.Items[1].Submission)
					require.Equal(t, 204, got.Items[1].StatusCode)
				}
			})
		})
	}
}

// TestProcessInstanceSuspensionWorkers verifies worker limiting and option/context
// propagation without relying on completion order or live service timing.
func TestProcessInstanceSuspensionWorkers(t *testing.T) {
	t.Parallel()
	for _, unlimited := range []bool{false, true} {
		t.Run(fmt.Sprint(unlimited), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				const n = 40
				type marker struct{}
				ctx := context.WithValue(context.Background(), marker{}, "context")
				opts := []services.CallOption{services.WithNoWait(), services.WithVerbose(), services.WithSuppressWorkflowDetailLogs(), services.WithSuppressProcessInstanceDetailLogs()}
				if unlimited {
					opts = append(opts, services.WithNoWorkerLimit())
				}
				var active, max atomic.Int32
				api := suspensionAPIStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
					cfg := services.ApplyCallOptions(opts)
					require.Equal(t, "context", ctx.Value(marker{}))
					require.True(t, cfg.NoWait)
					require.True(t, cfg.Verbose)
					require.True(t, cfg.SuppressWorkflowDetailLogs)
					require.True(t, cfg.SuppressProcessInstanceDetailLogs)
					require.Equal(t, unlimited, cfg.NoWorkerLimit)
					if cfg.DryRun {
						return suspensionItem(key, true), nil
					}
					count := active.Add(1)
					defer active.Add(-1)
					for old := max.Load(); count > old; old = max.Load() {
						if max.CompareAndSwap(old, count) {
							break
						}
					}
					time.Sleep(time.Second)
					return suspensionItem(key, false), nil
				}}
				got, err := SuspendProcessInstances(ctx, api, nil, suspensionKeys(n), 0, opts...)
				require.NoError(t, err)
				require.Len(t, got.Items, n)
				require.Equal(t, int32(toolx.DetermineNoOfWorkers(n, 0, unlimited)), max.Load())
			})
		})
	}
}
