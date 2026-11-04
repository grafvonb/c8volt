// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// facadeResumptionStub has explicit capability and mutation callbacks so an embedded
// legacy interface cannot hide missing resumption behavior.
type facadeResumptionStub struct {
	pisvc.API
	call func(context.Context, string, ...services.CallOption) (d.ProcessInstanceResumptionResult, error)
}

// ValidateProcessInstanceResumption enables the isolated facade fixture.
func (s facadeResumptionStub) ValidateProcessInstanceResumption() error { return nil }

// ResumeProcessInstance delegates to the fixture's evidence-returning callback.
func (s facadeResumptionStub) ResumeProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
	return s.call(ctx, key, opts...)
}

// TestProcessInstanceResumptionFacadeMapping verifies every field, partial-error
// conversion and copied collections while preserving options and caller context.
func TestProcessInstanceResumptionFacadeMapping(t *testing.T) {
	t.Parallel()
	type marker struct{}
	ctx := context.WithValue(context.Background(), marker{}, "context")
	const key = "2251799813685249"
	original := d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusTimedOut, Submission: d.ProcessInstanceResumptionSubmissionAccepted, Confirmation: d.ProcessInstanceResumptionConfirmationUnconfirmed, StatusCode: 204, HasObservation: true, LastObservedState: d.StateSuspended, SuspendedDate: "date", Error: "machine evidence"}
	previews, mutations := 0, 0
	api := facadeResumptionStub{call: func(gotCtx context.Context, gotKey string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
		require.Equal(t, "context", gotCtx.Value(marker{}))
		require.Equal(t, key, gotKey)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.NoWait)
		require.True(t, cfg.FailFast)
		require.True(t, cfg.NoWorkerLimit)
		require.True(t, cfg.Verbose)
		if cfg.DryRun {
			previews++
			return d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusPreview}, nil
		}
		mutations++
		return original, d.ErrGatewayTimeout
	}}
	facade := New(nil, api, nil, nil)
	opts := []options.FacadeOption{options.WithNoWait(), options.WithFailFast(), options.WithNoWorkerLimit(), options.WithVerbose()}
	single, err := facade.ResumeProcessInstance(ctx, key, opts...)
	require.ErrorIs(t, err, ferr.ErrTimeout)
	expected := ResumptionResult{Key: key, Status: ResumptionStatusTimedOut, Submission: ResumptionSubmissionAccepted, Confirmation: ResumptionConfirmationUnconfirmed, StatusCode: 204, HasObservation: true, LastObservedState: StateSuspended, SuspendedDate: "date", Error: "machine evidence"}
	require.Equal(t, expected, single)
	require.Equal(t, 1, previews)
	require.Equal(t, 1, mutations)
	keys := typex.Keys{key, key}
	multiple, err := facade.ResumeProcessInstances(ctx, keys, 2, opts...)
	require.ErrorIs(t, err, ferr.ErrTimeout)
	require.Equal(t, []ResumptionResult{expected}, multiple.Items)
	require.Equal(t, 2, previews)
	require.Equal(t, 2, mutations)
	require.Equal(t, typex.Keys{key, key}, keys)
	domainResult := d.ProcessInstanceResumptionResults{Items: []d.ProcessInstanceResumptionResult{original}}
	mapped := fromDomainResumptionResults(domainResult)
	mapped.Items[0].Key = "changed"
	require.Equal(t, key, domainResult.Items[0].Key)
	domainResult.Items[0].Error = "changed internally"
	require.Equal(t, original.Error, mapped.Items[0].Error)
	for _, result := range []ResumptionResult{{Key: key, Status: ResumptionStatusPreview, Submission: ResumptionSubmissionNotSubmitted, Confirmation: ResumptionConfirmationNotRequested}, {Key: key, HasObservation: true, LastObservedState: StateUnknown}} {
		encoded, err := json.Marshal(result)
		require.NoError(t, err)
		var fields map[string]any
		require.NoError(t, json.Unmarshal(encoded, &fields))
		require.Contains(t, fields, "hasObservation")
		require.NotContains(t, fields, "statusCode")
		require.NotContains(t, fields, "suspendedDate")
		require.NotContains(t, fields, "error")
		if result.HasObservation {
			require.Equal(t, "UNKNOWN", fields["lastObservedState"])
		} else {
			require.NotContains(t, fields, "lastObservedState")
		}
	}
	encoded, err := json.Marshal(fromDomainResumptionResults(d.ProcessInstanceResumptionResults{}))
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[]}`, string(encoded))
}

// TestProcessInstanceResumptionFactoryGuards proves real factory-selected adapters
// reject unsupported single, preview and empty collection paths before any I/O.
func TestProcessInstanceResumptionFactoryGuards(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		t.Run(string(version), func(t *testing.T) {
			calls := 0
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				t.Error("guard/preview made a request")
				w.WriteHeader(500)
			}))
			defer server.Close()
			cfg := &config.Config{App: config.App{CamundaVersion: version, Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Millisecond, Timeout: time.Second}}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL}, Operate: config.API{BaseURL: server.URL}}}
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			api, err := pisvc.New(cfg, server.Client(), log)
			require.NoError(t, err)
			facade := New(nil, api, nil, log)
			for _, noWait := range []bool{false, true} {
				opts := []options.FacadeOption{options.WithDryRun()}
				if noWait {
					opts = append(opts, options.WithNoWait())
				}
				single, err := facade.ResumeProcessInstance(context.Background(), "2251799813685249", opts...)
				if version == toolx.V810 {
					require.NoError(t, err)
					require.Equal(t, ResumptionStatusPreview, single.Status)
				} else {
					require.ErrorIs(t, err, ferr.ErrUnsupported)
				}
				for _, keys := range []typex.Keys{nil, {"2251799813685249"}} {
					result, err := facade.ResumeProcessInstances(context.Background(), keys, 3, opts...)
					if version == toolx.V810 {
						require.NoError(t, err)
						require.Len(t, result.Items, len(keys))
						require.NotNil(t, result.Items)
					} else {
						require.ErrorIs(t, err, ferr.ErrUnsupported)
					}
				}
			}
			result, err := facade.ResumeProcessInstances(context.Background(), nil, 1)
			if version == toolx.V810 {
				require.NoError(t, err)
				require.NotNil(t, result.Items)
			} else {
				require.ErrorIs(t, err, ferr.ErrUnsupported)
				_, err = facade.ResumeProcessInstance(context.Background(), "invalid")
				require.ErrorIs(t, err, ferr.ErrUnsupported)
				_, err = facade.ResumeProcessInstances(context.Background(), typex.Keys{"invalid"}, 1)
				require.ErrorIs(t, err, ferr.ErrUnsupported)
			}
			require.Zero(t, calls)
		})
	}
}

// TestProcessInstanceResumptionFactoryTargets verifies parent-only, child-only,
// duplicate and simultaneous parent/child scopes through generated HTTP clients.
// Every non-keyed route fails, making hierarchy or selection expansion observable.
func TestProcessInstanceResumptionFactoryTargets(t *testing.T) {
	t.Parallel()
	const parent = "2251799813685249"
	const child = "2251799813685250"
	for _, noWait := range []bool{false, true} {
		for _, keys := range []typex.Keys{{parent}, {child}, {parent, child, parent}, {child, parent, child}} {
			t.Run(fmt.Sprintf("noWait=%t/%s", noWait, strings.Join(keys, ",")), func(t *testing.T) {
				var mu sync.Mutex
				submitted := map[string]int{}
				reads := map[string]int{}
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
					if len(parts) < 3 || parts[0] != "v2" || parts[1] != "process-instances" {
						t.Errorf("unexpected route: %s", r.URL.Path)
						w.WriteHeader(500)
						return
					}
					key := parts[2]
					if key != parent && key != child {
						t.Errorf("unexpected key: %s", key)
						w.WriteHeader(500)
						return
					}
					switch {
					case r.Method == http.MethodPost && len(parts) == 4 && parts[3] == "resumption":
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						require.JSONEq(t, `{}`, string(body))
						submitted[key]++
						w.WriteHeader(204)
					case r.Method == http.MethodGet && len(parts) == 3:
						require.False(t, noWait, "no-wait must not read confirmation")
						require.Equal(t, 1, submitted[key])
						reads[key]++
						state := "SUSPENDED"
						if reads[key] > 1 {
							state = "ACTIVE"
						}
						if key == child {
							state = "COMPLETED"
						}
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprintf(w, `{"processInstanceKey":%q,"parentProcessInstanceKey":%q,"state":%q}`, key, parent, state)
					default:
						t.Errorf("unexpected method/path %s %s", r.Method, r.URL.Path)
						w.WriteHeader(500)
					}
				}))
				defer server.Close()
				cfg := &config.Config{App: config.App{CamundaVersion: toolx.V810, Tenant: "discovery-only", Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Millisecond, Timeout: time.Second, MaxRetries: 3}}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL + "/v2"}}}
				var logs bytes.Buffer
				log := slog.New(slog.NewTextHandler(&logs, nil))
				api, err := pisvc.New(cfg, server.Client(), log)
				require.NoError(t, err)
				facade := New(nil, api, nil, log)
				opts := []options.FacadeOption{options.WithVerbose()}
				if noWait {
					opts = append(opts, options.WithNoWait())
				}
				got, err := facade.ResumeProcessInstances(context.Background(), keys, 2, opts...)
				require.NoError(t, err)
				unique := keys.Unique()
				require.Len(t, got.Items, len(unique))
				for i, result := range got.Items {
					require.Equal(t, unique[i], result.Key)
					if noWait {
						require.Equal(t, ResumptionStatusAccepted, result.Status)
						require.Equal(t, ResumptionSubmissionAccepted, result.Submission)
						require.Equal(t, ResumptionConfirmationNotRequested, result.Confirmation)
						require.False(t, result.HasObservation)
						require.Empty(t, result.LastObservedState)
						require.Empty(t, result.SuspendedDate)
						require.Equal(t, 204, result.StatusCode)
						require.Equal(t, 1, submitted[result.Key])
						require.Zero(t, reads[result.Key])
						continue
					}
					require.Equal(t, ResumptionStatusConfirmed, result.Status)
					require.Equal(t, ResumptionSubmissionAccepted, result.Submission)
					require.Equal(t, ResumptionConfirmationConfirmed, result.Confirmation)
					if result.Key == child {
						require.Equal(t, StateCompleted, result.LastObservedState)
						require.Equal(t, 1, reads[result.Key])
					} else {
						require.Equal(t, StateActive, result.LastObservedState)
						require.Equal(t, 2, reads[result.Key])
					}
					require.Empty(t, result.SuspendedDate)
					require.Equal(t, 204, result.StatusCode)
					require.Equal(t, 1, submitted[result.Key])
				}
				require.Len(t, submitted, len(unique))
				require.Empty(t, logs.String())
			})
		}
	}
}

// TestProcessInstanceResumptionSerialization fixes the evidence vocabulary and
// distinguishes absent observations from UNKNOWN without inventing timestamps.
func TestProcessInstanceResumptionSerialization(t *testing.T) {
	t.Parallel()
	preview := ResumptionResult{
		Key: "2251799813685249", Status: ResumptionStatusPreview,
		Submission:   ResumptionSubmissionNotSubmitted,
		Confirmation: ResumptionConfirmationNotRequested,
	}
	data, err := json.Marshal(preview)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"2251799813685249","status":"preview","submission":"not_submitted","confirmation":"not_requested","hasObservation":false}`, string(data))
	observed := ResumptionResult{
		Key: preview.Key, Status: ResumptionStatusTimedOut,
		Submission:   ResumptionSubmissionAccepted,
		Confirmation: ResumptionConfirmationUnconfirmed,
		StatusCode:   204, HasObservation: true, LastObservedState: StateUnknown,
		SuspendedDate: "supplied-date", Error: "machine evidence",
	}
	data, err = json.Marshal(observed)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"2251799813685249","status":"timed_out","submission":"accepted","confirmation":"unconfirmed","statusCode":204,"hasObservation":true,"lastObservedState":"UNKNOWN","suspendedDate":"supplied-date","error":"machine evidence"}`, string(data))
	var roundTrip ResumptionResult
	require.NoError(t, json.Unmarshal(data, &roundTrip))
	require.Equal(t, observed, roundTrip)
	observed.SuspendedDate = ""
	data, err = json.Marshal(observed)
	require.NoError(t, err)
	require.NotContains(t, string(data), "suspendedDate")
	data, err = json.Marshal(ResumptionResults{Items: []ResumptionResult{}})
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[]}`, string(data))

	data, err = json.Marshal([]ResumptionStatus{
		ResumptionStatusPreview, ResumptionStatusAccepted,
		ResumptionStatusConfirmed, ResumptionStatusFailed,
		ResumptionStatusTimedOut, ResumptionStatusInterrupted,
		ResumptionStatusNotAttempted,
	})
	require.NoError(t, err)
	require.JSONEq(t, `["preview","accepted","confirmed","failed","timed_out","interrupted","not_attempted"]`, string(data))
	data, err = json.Marshal([]ResumptionSubmission{
		ResumptionSubmissionNotSubmitted, ResumptionSubmissionAccepted,
		ResumptionSubmissionRejected, ResumptionSubmissionUnknown,
	})
	require.NoError(t, err)
	require.JSONEq(t, `["not_submitted","accepted","rejected","unknown"]`, string(data))
	data, err = json.Marshal([]ResumptionConfirmation{
		ResumptionConfirmationNotRequested, ResumptionConfirmationUnconfirmed,
		ResumptionConfirmationConfirmed,
	})
	require.NoError(t, err)
	require.JSONEq(t, `["not_requested","unconfirmed","confirmed"]`, string(data))
}

// TestProcessInstanceResumptionSingletonValidation proves singular calls use the
// same capability, normalization, cancellation and local-preview path as collections.
func TestProcessInstanceResumptionSingletonValidation(t *testing.T) {
	t.Parallel()
	const key = "2251799813685249"
	for _, mode := range []string{"empty", "invalid", "canceled", "preview", "preview failure"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			api := facadeResumptionStub{call: func(ctx context.Context, gotKey string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
				calls++
				require.Equal(t, key, gotKey)
				require.True(t, services.ApplyCallOptions(opts).DryRun)
				if mode == "preview failure" {
					return d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusFailed, Submission: d.ProcessInstanceResumptionSubmissionNotSubmitted, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested, Error: "invalid timing"}, d.ErrValidation
				}
				return d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusPreview, Submission: d.ProcessInstanceResumptionSubmissionNotSubmitted, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested}, nil
			}}
			selected := " " + key + "\n"
			opts := []options.FacadeOption{}
			switch mode {
			case "empty":
				selected = " "
			case "invalid":
				selected = "bad"
			case "canceled":
				cancel()
			case "preview":
				opts = append(opts, options.WithDryRun())
			}
			got, err := New(nil, api, nil, nil).ResumeProcessInstance(ctx, selected, opts...)
			switch mode {
			case "empty", "invalid":
				require.ErrorIs(t, err, ferr.ErrInvalidInput)
				require.Zero(t, got)
				require.Zero(t, calls)
			case "canceled":
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, key, got.Key)
				require.Equal(t, ResumptionStatusNotAttempted, got.Status)
				require.Equal(t, ResumptionSubmissionNotSubmitted, got.Submission)
				require.Zero(t, calls)
			case "preview":
				require.NoError(t, err)
				require.Equal(t, ResumptionStatusPreview, got.Status)
				require.Equal(t, key, got.Key)
				require.Equal(t, 1, calls)
			case "preview failure":
				require.ErrorIs(t, err, ferr.ErrInvalidInput)
				require.Equal(t, ResumptionStatusFailed, got.Status)
				require.Equal(t, "invalid timing", got.Error)
				require.Equal(t, 1, calls)
			}
		})
	}
}

// TestProcessInstanceResumptionFacadeWorkers verifies public worker/options mapping
// reaches the internal executor and does not mutate the caller's selected slice.
func TestProcessInstanceResumptionFacadeWorkers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		wanted    int
		unlimited bool
	}{{2, false}, {2, true}, {0, true}, {0, false}} {
		t.Run(fmt.Sprint(tc), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				const n = 40
				keys := make(typex.Keys, n)
				for i := range keys {
					keys[i] = fmt.Sprintf("%016d", i+1)
				}
				before := append(typex.Keys(nil), keys...)
				var active, max atomic.Int32
				api := facadeResumptionStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
					cfg := services.ApplyCallOptions(opts)
					require.True(t, cfg.NoWait)
					require.True(t, cfg.IgnoreTenant)
					require.Equal(t, tc.unlimited, cfg.NoWorkerLimit)
					if cfg.DryRun {
						return d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusPreview}, nil
					}
					current := active.Add(1)
					defer active.Add(-1)
					for old := max.Load(); current > old; old = max.Load() {
						if max.CompareAndSwap(old, current) {
							break
						}
					}
					time.Sleep(time.Second)
					return d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusAccepted, Submission: d.ProcessInstanceResumptionSubmissionAccepted, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested, StatusCode: 204}, nil
				}}
				opts := []options.FacadeOption{options.WithNoWait(), options.WithIgnoreTenant()}
				if tc.unlimited {
					opts = append(opts, options.WithNoWorkerLimit())
				}
				got, err := New(nil, api, nil, nil).ResumeProcessInstances(context.Background(), keys, tc.wanted, opts...)
				require.NoError(t, err)
				require.Len(t, got.Items, n)
				require.Equal(t, int32(toolx.DetermineNoOfWorkers(n, tc.wanted, tc.unlimited)), max.Load())
				require.Equal(t, before, keys)
				for i, item := range got.Items {
					require.Equal(t, keys[i], item.Key)
				}
			})
		})
	}
}

// TestProcessInstanceResumptionFacadePartialFailure retains accepted/unconfirmed,
// unknown, rejected and unattempted evidence while converting the original class.
func TestProcessInstanceResumptionFacadePartialFailure(t *testing.T) {
	t.Parallel()
	for _, failFast := range []bool{false, true} {
		t.Run(fmt.Sprint(failFast), func(t *testing.T) {
			keys := typex.Keys{"2251799813685249", "2251799813685250", "2251799813685251"}
			original := []d.ProcessInstanceResumptionResult{
				{Key: keys[0], Status: d.ProcessInstanceResumptionStatusFailed, Submission: d.ProcessInstanceResumptionSubmissionAccepted, Confirmation: d.ProcessInstanceResumptionConfirmationUnconfirmed, HasObservation: true, LastObservedState: d.StateCanceled, StatusCode: 204, Error: "cancellation observed"},
				{Key: keys[1], Status: d.ProcessInstanceResumptionStatusFailed, Submission: d.ProcessInstanceResumptionSubmissionUnknown, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested, Error: "POST https://backend.invalid/resume body=distinctive key=" + keys[1]},
				{Key: keys[2], Status: d.ProcessInstanceResumptionStatusFailed, Submission: d.ProcessInstanceResumptionSubmissionRejected, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested, StatusCode: 409, Error: "POST https://backend.invalid/resume body=distinctive key=" + keys[2]},
			}
			calls := 0
			api := facadeResumptionStub{call: func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
				if services.ApplyCallOptions(opts).DryRun {
					return d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusPreview}, nil
				}
				result := original[calls]
				calls++
				return result, fmt.Errorf("%w: %s", d.ErrConflict, result.Error)
			}}
			var logs bytes.Buffer
			log := slog.New(slog.NewTextHandler(&logs, nil))
			opts := []options.FacadeOption{options.WithVerbose()}
			if failFast {
				opts = append(opts, options.WithFailFast())
			}
			got, err := New(nil, api, nil, log).ResumeProcessInstances(logging.ToContext(context.Background(), log), keys, 1, opts...)
			require.ErrorIs(t, err, ferr.ErrConflict)
			require.Equal(t, ferr.ClassConflict, ferr.Classify(err))
			require.Equal(t, ferr.ExitCode(ferr.ErrConflict), ferr.ExitCode(err))
			require.Len(t, got.Items, len(keys))
			require.Equal(t, fromDomainResumptionResult(original[0]), got.Items[0])
			if failFast {
				require.Equal(t, 1, calls)
				for i := 1; i < len(keys); i++ {
					require.Equal(t, keys[i], got.Items[i].Key)
					require.Equal(t, ResumptionStatusNotAttempted, got.Items[i].Status)
					require.Equal(t, ResumptionSubmissionNotSubmitted, got.Items[i].Submission)
					require.Equal(t, ResumptionConfirmationNotRequested, got.Items[i].Confirmation)
					require.NotEmpty(t, got.Items[i].Error)
				}
			} else {
				require.Equal(t, 3, calls)
				require.Equal(t, fromDomainResumptionResults(d.ProcessInstanceResumptionResults{Items: original}), got)
				require.Contains(t, err.Error(), original[1].Error)
				require.Contains(t, err.Error(), original[2].Error)
			}
			require.Empty(t, logs.String(), "raw machine failures must not become facade diagnostics")
		})
	}
}

// TestProcessInstanceResumptionFactoryValidation proves real adapter timing and
// all-key validation precede every request; no-wait exempts confirmation timing.
func TestProcessInstanceResumptionFactoryValidation(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"invalid key", "invalid timing", "invalid timing preview", "no wait", "no wait preview"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "no wait", mode)
				require.Equal(t, http.MethodPost, r.Method)
				require.True(t, strings.HasSuffix(r.URL.Path, "/resumption"))
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			// Negative confirmation timing must fail locally. It is intentionally
			// irrelevant under NoWait, which still submits through the real adapter.
			cfg := &config.Config{App: config.App{CamundaVersion: toolx.V810, Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Millisecond, Timeout: -time.Second}}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL + "/v2"}}}
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			api, err := pisvc.New(cfg, server.Client(), log)
			require.NoError(t, err)
			keys := typex.Keys{"2251799813685249", "2251799813685250"}
			opts := []options.FacadeOption{}
			switch mode {
			case "invalid key":
				cfg.App.Backoff.Timeout = time.Second
				keys = append(keys, "bad")
			case "invalid timing preview":
				opts = append(opts, options.WithDryRun())
			case "no wait":
				opts = append(opts, options.WithNoWait())
			case "no wait preview":
				opts = append(opts, options.WithNoWait(), options.WithDryRun())
			}
			result, err := New(nil, api, nil, log).ResumeProcessInstances(context.Background(), keys, 2, opts...)
			switch mode {
			case "no wait", "no wait preview":
				require.NoError(t, err)
				require.Len(t, result.Items, 2)
				for _, item := range result.Items {
					require.False(t, item.HasObservation)
					require.Equal(t, ResumptionConfirmationNotRequested, item.Confirmation)
					if mode == "no wait" {
						require.Equal(t, ResumptionStatusAccepted, item.Status)
						require.Equal(t, ResumptionSubmissionAccepted, item.Submission)
					} else {
						require.Equal(t, ResumptionStatusPreview, item.Status)
						require.Equal(t, ResumptionSubmissionNotSubmitted, item.Submission)
					}
				}
			default:
				require.ErrorIs(t, err, ferr.ErrInvalidInput)
				if mode == "invalid key" {
					require.Empty(t, result.Items)
				} else {
					require.Len(t, result.Items, 2)
					require.Equal(t, ResumptionStatusFailed, result.Items[0].Status)
					require.Equal(t, ResumptionStatusNotAttempted, result.Items[1].Status)
					for _, item := range result.Items {
						require.Equal(t, ResumptionSubmissionNotSubmitted, item.Submission)
					}
				}
			}
			if mode == "no wait" {
				require.Equal(t, int32(2), calls.Load())
			} else {
				require.Zero(t, calls.Load())
			}
		})
	}
}
