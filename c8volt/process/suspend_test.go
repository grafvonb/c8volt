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
	"testing"
	"time"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// facadeSuspensionStub has explicit capability and mutation callbacks so an embedded
// legacy interface cannot hide missing suspension behavior.
type facadeSuspensionStub struct {
	pisvc.API
	call func(context.Context, string, ...services.CallOption) (d.ProcessInstanceSuspensionResult, error)
}

// ValidateProcessInstanceSuspension enables the isolated facade fixture.
func (s facadeSuspensionStub) ValidateProcessInstanceSuspension() error { return nil }

// SuspendProcessInstance delegates to the fixture's evidence-returning callback.
func (s facadeSuspensionStub) SuspendProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
	return s.call(ctx, key, opts...)
}

// TestProcessInstanceSuspensionFacadeMapping verifies every field, partial-error
// conversion and copied collections while preserving options and caller context.
func TestProcessInstanceSuspensionFacadeMapping(t *testing.T) {
	t.Parallel()
	type marker struct{}
	ctx := context.WithValue(context.Background(), marker{}, "context")
	const key = "2251799813685249"
	original := d.ProcessInstanceSuspensionResult{Key: key, Status: d.ProcessInstanceSuspensionStatusTimedOut, Submission: d.ProcessInstanceSuspensionSubmissionAccepted, Confirmation: d.ProcessInstanceSuspensionConfirmationUnconfirmed, StatusCode: 204, HasObservation: true, LastObservedState: d.StateActive, SuspendedDate: "date", Error: "machine evidence"}
	api := facadeSuspensionStub{call: func(gotCtx context.Context, gotKey string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
		require.Equal(t, "context", gotCtx.Value(marker{}))
		require.Equal(t, key, gotKey)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.NoWait)
		require.True(t, cfg.FailFast)
		require.True(t, cfg.NoWorkerLimit)
		require.True(t, cfg.Verbose)
		if cfg.DryRun {
			return d.ProcessInstanceSuspensionResult{Key: key, Status: d.ProcessInstanceSuspensionStatusPreview}, nil
		}
		return original, d.ErrGatewayTimeout
	}}
	facade := New(nil, api, nil, nil)
	opts := []options.FacadeOption{options.WithNoWait(), options.WithFailFast(), options.WithNoWorkerLimit(), options.WithVerbose()}
	single, err := facade.SuspendProcessInstance(ctx, key, opts...)
	require.ErrorIs(t, err, ferr.ErrTimeout)
	expected := SuspensionResult{Key: key, Status: SuspensionStatusTimedOut, Submission: SuspensionSubmissionAccepted, Confirmation: SuspensionConfirmationUnconfirmed, StatusCode: 204, HasObservation: true, LastObservedState: StateActive, SuspendedDate: "date", Error: "machine evidence"}
	require.Equal(t, expected, single)
	keys := typex.Keys{key, key}
	multiple, err := facade.SuspendProcessInstances(ctx, keys, 2, opts...)
	require.ErrorIs(t, err, ferr.ErrTimeout)
	require.Equal(t, []SuspensionResult{expected}, multiple.Items)
	require.Equal(t, typex.Keys{key, key}, keys)
	domainResult := d.ProcessInstanceSuspensionResults{Items: []d.ProcessInstanceSuspensionResult{original}}
	mapped := fromDomainSuspensionResults(domainResult)
	mapped.Items[0].Key = "changed"
	require.Equal(t, key, domainResult.Items[0].Key)
	for _, result := range []SuspensionResult{{Key: key, Status: SuspensionStatusPreview, Submission: SuspensionSubmissionNotSubmitted, Confirmation: SuspensionConfirmationNotRequested}, {Key: key, HasObservation: true, LastObservedState: StateUnknown}} {
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
	encoded, err := json.Marshal(fromDomainSuspensionResults(d.ProcessInstanceSuspensionResults{}))
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[]}`, string(encoded))
}

// TestProcessInstanceSuspensionFactoryGuards proves real factory-selected adapters
// reject unsupported single, preview and empty collection paths before any I/O.
func TestProcessInstanceSuspensionFactoryGuards(t *testing.T) {
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
				single, err := facade.SuspendProcessInstance(context.Background(), "2251799813685249", opts...)
				if version == toolx.V810 {
					require.NoError(t, err)
					require.Equal(t, SuspensionStatusPreview, single.Status)
				} else {
					require.ErrorIs(t, err, ferr.ErrUnsupported)
				}
				for _, keys := range []typex.Keys{nil, {"2251799813685249"}} {
					result, err := facade.SuspendProcessInstances(context.Background(), keys, 3, opts...)
					if version == toolx.V810 {
						require.NoError(t, err)
						require.Len(t, result.Items, len(keys))
						require.NotNil(t, result.Items)
					} else {
						require.ErrorIs(t, err, ferr.ErrUnsupported)
					}
				}
			}
			result, err := facade.SuspendProcessInstances(context.Background(), nil, 1)
			if version == toolx.V810 {
				require.NoError(t, err)
				require.NotNil(t, result.Items)
			} else {
				require.ErrorIs(t, err, ferr.ErrUnsupported)
				_, err = facade.SuspendProcessInstance(context.Background(), "invalid")
				require.ErrorIs(t, err, ferr.ErrUnsupported)
				_, err = facade.SuspendProcessInstances(context.Background(), typex.Keys{"invalid"}, 1)
				require.ErrorIs(t, err, ferr.ErrUnsupported)
			}
			require.Zero(t, calls)
		})
	}
}

// TestProcessInstanceSuspensionFactoryTargets verifies parent-only, child-only,
// duplicate and simultaneous parent/child scopes through generated HTTP clients.
// Every non-keyed route fails, making hierarchy or selection expansion observable.
func TestProcessInstanceSuspensionFactoryTargets(t *testing.T) {
	t.Parallel()
	const parent = "2251799813685249"
	const child = "2251799813685250"
	for _, keys := range []typex.Keys{{parent}, {child}, {parent, child, parent}, {child, parent, child}} {
		t.Run(strings.Join(keys, ","), func(t *testing.T) {
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
				case r.Method == http.MethodPost && len(parts) == 4 && parts[3] == "suspension":
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					require.JSONEq(t, `{}`, string(body))
					submitted[key]++
					w.WriteHeader(204)
				case r.Method == http.MethodGet && len(parts) == 3:
					require.Equal(t, 1, submitted[key])
					reads[key]++
					state := "ACTIVE"
					if reads[key] > 1 {
						state = "SUSPENDED"
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(w, `{"processInstanceKey":%q,"parentProcessInstanceKey":%q,"state":%q,"suspendedDate":"2026-10-25T12:00:00Z"}`, key, parent, state)
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
			got, err := facade.SuspendProcessInstances(context.Background(), keys, 2, options.WithVerbose())
			require.NoError(t, err)
			unique := keys.Unique()
			require.Len(t, got.Items, len(unique))
			for i, result := range got.Items {
				require.Equal(t, unique[i], result.Key)
				require.Equal(t, SuspensionStatusConfirmed, result.Status)
				require.Equal(t, SuspensionSubmissionAccepted, result.Submission)
				require.Equal(t, SuspensionConfirmationConfirmed, result.Confirmation)
				require.Equal(t, StateSuspended, result.LastObservedState)
				require.Equal(t, "2026-10-25T12:00:00Z", result.SuspendedDate)
				require.Equal(t, 204, result.StatusCode)
				require.Equal(t, 1, submitted[result.Key])
				require.Equal(t, 2, reads[result.Key])
			}
			require.Len(t, submitted, len(unique))
			require.Empty(t, logs.String())
		})
	}
}
