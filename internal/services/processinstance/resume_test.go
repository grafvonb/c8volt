// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// resumptionAPIStub explicitly owns both new methods; other APIs must never be used.
type resumptionAPIStub struct {
	API
	guard error
	call  func(context.Context, string, ...services.CallOption) (d.ProcessInstanceResumptionResult, error)
}

// ValidateProcessInstanceResumption exposes local capability without remote calls.
func (s resumptionAPIStub) ValidateProcessInstanceResumption() error { return s.guard }

// ResumeProcessInstance delegates to an explicit test callback, including previews.
func (s resumptionAPIStub) ResumeProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
	return s.call(ctx, key, opts...)
}

// resumptionKeys creates valid independent targets in input order.
func resumptionKeys(n int) typex.Keys {
	keys := make(typex.Keys, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("%016d", i+1)
	}
	return keys
}

// resumptionItem supplies evidence appropriate to a test-owned preview or mutation.
func resumptionItem(key string, preview bool) d.ProcessInstanceResumptionResult {
	r := d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusAccepted, Submission: d.ProcessInstanceResumptionSubmissionAccepted, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested, StatusCode: 204}
	if preview {
		r.Status = d.ProcessInstanceResumptionStatusPreview
		r.Submission = d.ProcessInstanceResumptionSubmissionNotSubmitted
		r.StatusCode = 0
	}
	return r
}

// TestResumeProcessInstancesInput checks capability-first guards,
// complete normalization, no-op arrays, cancellation and no-I/O preview behavior.
func TestResumeProcessInstancesInput(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"unsupported empty", "unsupported malformed", "invalid", "empty", "preview", "preview no wait", "normalized", "canceled", "canceled after preview", "invalid timing"} {
		t.Run(mode, func(t *testing.T) {
			keys := resumptionKeys(2)
			selected := append(typex.Keys(nil), keys...)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opts := []services.CallOption{}
			previews, mutations := 0, 0
			api := resumptionAPIStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
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
					return d.ProcessInstanceResumptionResult{}, d.ErrValidation
				}
				return resumptionItem(key, preview), nil
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
			got, err := ResumeProcessInstances(ctx, api, nil, selected, 1, opts...)
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
					require.Equal(t, d.ProcessInstanceResumptionStatusNotAttempted, item.Status)
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
					require.Equal(t, d.ProcessInstanceResumptionStatusPreview, r.Status)
					require.Equal(t, d.ProcessInstanceResumptionSubmissionNotSubmitted, r.Submission)
				}
			}
		})
	}
}

// TestResumeProcessInstancesFailures retains started evidence, joined
// error causes and keyed unstarted slots under fail-fast and caller cancellation.
func TestResumeProcessInstancesFailures(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"mixed", "fail fast", "cancel during", "in flight accepted"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				keys := resumptionKeys(4)
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
				api := resumptionAPIStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
					if services.ApplyCallOptions(opts).DryRun {
						return resumptionItem(key, true), nil
					}
					started.Add(1)
					r := resumptionItem(key, false)
					if key == keys[0] {
						if mode == "in flight accepted" {
							<-ready
						}
						r.Status = d.ProcessInstanceResumptionStatusFailed
						r.Submission = d.ProcessInstanceResumptionSubmissionRejected
						r.StatusCode = 409
						r.Error = "conflict"
						return r, d.ErrConflict
					}
					if mode == "in flight accepted" {
						close(ready)
						<-ctx.Done()
						r.Status = d.ProcessInstanceResumptionStatusInterrupted
						r.Confirmation = d.ProcessInstanceResumptionConfirmationUnconfirmed
						r.Error = ctx.Err().Error()
						return r, ctx.Err()
					}
					if mode == "cancel during" {
						cancel()
						return r, context.Canceled
					}
					return r, nil
				}}
				got, err := ResumeProcessInstances(ctx, api, nil, keys, workers, opts...)
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
					require.Equal(t, d.ProcessInstanceResumptionStatusNotAttempted, got.Items[i].Status)
					require.NotEmpty(t, got.Items[i].Error)
				}
				if count > 1 {
					require.Equal(t, d.ProcessInstanceResumptionSubmissionAccepted, got.Items[1].Submission)
					require.Equal(t, 204, got.Items[1].StatusCode)
				}
			})
		})
	}
}

// TestResumeProcessInstancesWorkers verifies worker limiting and option/context
// propagation without relying on completion order or live service timing.
func TestResumeProcessInstancesWorkers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		wanted    int
		unlimited bool
	}{{0, false}, {0, true}, {3, false}, {3, true}, {100, false}} {
		unlimited, wanted := tc.unlimited, tc.wanted
		t.Run(fmt.Sprintf("workers=%d/unlimited=%t", wanted, unlimited), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				const n = 40
				type marker struct{}
				ctx := context.WithValue(context.Background(), marker{}, "context")
				opts := []services.CallOption{services.WithNoWait(), services.WithVerbose(), services.WithSuppressWorkflowDetailLogs(), services.WithSuppressProcessInstanceDetailLogs()}
				if unlimited {
					opts = append(opts, services.WithNoWorkerLimit())
				}
				var active, max atomic.Int32
				api := resumptionAPIStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
					cfg := services.ApplyCallOptions(opts)
					require.Equal(t, "context", ctx.Value(marker{}))
					require.True(t, cfg.NoWait)
					require.True(t, cfg.Verbose)
					require.True(t, cfg.SuppressWorkflowDetailLogs)
					require.True(t, cfg.SuppressProcessInstanceDetailLogs)
					require.Equal(t, unlimited, cfg.NoWorkerLimit)
					if cfg.DryRun {
						return resumptionItem(key, true), nil
					}
					count := active.Add(1)
					defer active.Add(-1)
					for old := max.Load(); count > old; old = max.Load() {
						if max.CompareAndSwap(old, count) {
							break
						}
					}
					time.Sleep(time.Second)
					return resumptionItem(key, false), nil
				}}
				got, err := ResumeProcessInstances(ctx, api, nil, resumptionKeys(n), wanted, opts...)
				require.NoError(t, err)
				require.Len(t, got.Items, n)
				require.Equal(t, int32(toolx.DetermineNoOfWorkers(n, wanted, unlimited)), max.Load())
			})
		})
	}
}

// TestResumeProcessInstancesCompletionOrder preserves every evidence field and
// joined cause when workers finish in the opposite order to the selected keys.
func TestResumeProcessInstancesCompletionOrder(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		keys := resumptionKeys(4)
		want := []d.ProcessInstanceResumptionResult{
			{Key: keys[0], Status: d.ProcessInstanceResumptionStatusConfirmed, Submission: d.ProcessInstanceResumptionSubmissionAccepted, Confirmation: d.ProcessInstanceResumptionConfirmationConfirmed, StatusCode: 204, HasObservation: true, LastObservedState: d.StateCompleted},
			{Key: keys[1], Status: d.ProcessInstanceResumptionStatusFailed, Submission: d.ProcessInstanceResumptionSubmissionRejected, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested, StatusCode: 409, Error: "POST https://backend.invalid/resume body=distinctive key=" + keys[1]},
			{Key: keys[2], Status: d.ProcessInstanceResumptionStatusFailed, Submission: d.ProcessInstanceResumptionSubmissionUnknown, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested, Error: "POST https://backend.invalid/resume body=distinctive key=" + keys[2]},
			{Key: keys[3], Status: d.ProcessInstanceResumptionStatusTimedOut, Submission: d.ProcessInstanceResumptionSubmissionAccepted, Confirmation: d.ProcessInstanceResumptionConfirmationUnconfirmed, StatusCode: 204, HasObservation: true, LastObservedState: d.StateSuspended, SuspendedDate: "supplied-date", Error: "timed out"},
		}
		causes := []error{nil, d.ErrConflict, d.ErrUnavailable, d.ErrGatewayTimeout}
		completed := make(chan string, len(keys))
		api := resumptionAPIStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
			if services.ApplyCallOptions(opts).DryRun {
				return resumptionItem(key, true), nil
			}
			i := slices.Index(keys, key)
			time.Sleep(time.Duration(len(keys)-i) * time.Second)
			completed <- key
			return want[i], causes[i]
		}}
		var logs bytes.Buffer
		log := slog.New(slog.NewTextHandler(&logs, nil))
		got, err := ResumeProcessInstances(logging.ToContext(context.Background(), log), api, log, keys, 4, services.WithVerbose())
		require.ErrorIs(t, err, d.ErrConflict)
		require.ErrorIs(t, err, d.ErrUnavailable)
		require.ErrorIs(t, err, d.ErrGatewayTimeout)
		require.Equal(t, want, got.Items)
		for i := len(keys) - 1; i >= 0; i-- {
			require.Equal(t, keys[i], <-completed)
		}
		require.Empty(t, logs.String(), "collection returns evidence without logging raw joined errors")
	})
}
