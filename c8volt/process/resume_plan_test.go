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
	"slices"
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
	"github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// facadeResumptionPlanStub exposes discovery explicitly rather than silently using an embedded API.
type facadeResumptionPlanStub struct {
	facadeResumptionStub
	descendants func(context.Context, string, ...services.CallOption) (traversal.Result, error)
}

// DescendantsResult forwards the complete planner boundary to its fixture callback.
func (s facadeResumptionPlanStub) DescendantsResult(ctx context.Context, key string, opts ...services.CallOption) (traversal.Result, error) {
	return s.descendants(ctx, key, opts...)
}

// TestProcessInstanceResumptionPlanFacadeMapping checks copied boundaries, option
// and context propagation, exact JSON names, and non-null successful arrays.
func TestProcessInstanceResumptionPlanFacadeMapping(t *testing.T) {
	t.Parallel()
	const key = "2251799813685249"
	type marker struct{}
	ctx := context.WithValue(context.Background(), marker{}, "context")
	observed := traversal.Result{Mode: traversal.ModeDescendants, Outcome: traversal.OutcomeComplete, StartKey: key, RootKey: key, Keys: []string{key}, Chain: map[string]d.ProcessInstance{key: {Key: key, State: d.StateSuspended}}}
	input := typex.Keys{key, key}
	api := facadeResumptionPlanStub{facadeResumptionStub: facadeResumptionStub{call: func(gotCtx context.Context, gotKey string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
		require.Same(t, ctx, gotCtx)
		require.Equal(t, key, gotKey)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.DryRun && cfg.NoWait && cfg.Verbose && cfg.NoWorkerLimit && cfg.FailFast)
		return d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusPreview}, nil
	}}, descendants: func(gotCtx context.Context, gotKey string, opts ...services.CallOption) (traversal.Result, error) {
		require.Same(t, ctx, gotCtx)
		require.Equal(t, key, gotKey)
		cfg := services.ApplyCallOptions(opts)
		require.False(t, cfg.DryRun)
		require.True(t, cfg.NoWait && cfg.Verbose && cfg.NoWorkerLimit && cfg.FailFast)
		require.True(t, cfg.IgnoreTenant && cfg.SuppressWorkflowDetailLogs && cfg.SuppressProcessInstanceDetailLogs)
		return observed, nil
	}}
	facade := New(nil, api, nil, nil)
	plan, err := facade.PlanProcessInstanceResumption(ctx, input, options.WithNoWait(), options.WithVerbose(), options.WithFailFast(), options.WithNoWorkerLimit())
	require.NoError(t, err)
	require.Equal(t, typex.Keys{key, key}, input)
	raw, err := json.Marshal(plan)
	require.NoError(t, err)
	require.JSONEq(t, `{"requestedKeys":["2251799813685249"],"instances":[{"key":"2251799813685249","observedState":"SUSPENDED","classification":"eligible"}],"targetKeys":["2251799813685249"]}`, string(raw))
	input[0] = "input changed"
	observed.Keys[0] = "traversal changed"
	observed.Chain[key] = d.ProcessInstance{Key: "changed"}
	require.Equal(t, key, plan.RequestedKeys[0])
	require.Equal(t, key, plan.TargetKeys[0])
	require.Equal(t, key, plan.Instances[0].Key)
	source := d.ProcessInstanceResumptionPlan{RequestedKeys: typex.Keys{key}, TargetKeys: typex.Keys{key}, Instances: []d.ProcessInstanceResumptionPlanItem{{Key: key, ObservedState: d.StateActive, Classification: d.ProcessInstanceResumptionPlanExcluded}}}
	mapped := fromDomainProcessInstanceResumptionPlan(source)
	require.Equal(t, StateActive, mapped.Instances[0].ObservedState)
	require.Equal(t, ProcessInstanceResumptionPlanExcluded, mapped.Instances[0].Classification)
	mapped.RequestedKeys[0], mapped.TargetKeys[0], mapped.Instances[0].Key = "changed", "changed", "changed"
	require.Equal(t, key, source.RequestedKeys[0])
	require.Equal(t, key, source.TargetKeys[0])
	require.Equal(t, key, source.Instances[0].Key)
	empty := fromDomainProcessInstanceResumptionPlan(d.ProcessInstanceResumptionPlan{RequestedKeys: typex.Keys{}, Instances: []d.ProcessInstanceResumptionPlanItem{}, TargetKeys: typex.Keys{}})
	raw, err = json.Marshal(empty)
	require.NoError(t, err)
	require.JSONEq(t, `{"requestedKeys":[],"instances":[],"targetKeys":[]}`, string(raw))
	require.Equal(t, ProcessInstanceResumptionPlan{}, fromDomainProcessInstanceResumptionPlan(d.ProcessInstanceResumptionPlan{}))
}

// TestProcessInstanceResumptionPlanFacadeErrors retains public error classes and
// cause text while returning a zero plan for every discovery failure.
func TestProcessInstanceResumptionPlanFacadeErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ cause, public error }{
		{d.ErrForbidden, ferr.ErrLocalPrecondition}, {d.ErrNotFound, ferr.ErrNotFound},
		{d.ErrPrecondition, ferr.ErrLocalPrecondition}, {d.ErrMalformedResponse, ferr.ErrMalformedResponse},
		{context.Canceled, context.Canceled}, {context.DeadlineExceeded, ferr.ErrTimeout},
	} {
		t.Run(tc.cause.Error(), func(t *testing.T) {
			api := facadeResumptionPlanStub{facadeResumptionStub: facadeResumptionStub{call: func(context.Context, string, ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
				return d.ProcessInstanceResumptionResult{}, nil
			}}, descendants: func(context.Context, string, ...services.CallOption) (traversal.Result, error) {
				return traversal.Result{}, fmt.Errorf("discovery: %w", tc.cause)
			}}
			plan, err := New(nil, api, nil, nil).PlanProcessInstanceResumption(context.Background(), typex.Keys{"2251799813685249"})
			require.ErrorIs(t, err, tc.public)
			require.Contains(t, err.Error(), tc.cause.Error())
			require.Equal(t, ProcessInstanceResumptionPlan{}, plan)
		})
	}
}

// TestProcessInstanceResumptionPlanFactoryGuards exercises real configured adapters
// before any HTTP, including empty/malformed input and invalid waiting configuration.
func TestProcessInstanceResumptionPlanFactoryGuards(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		t.Run(string(version), func(t *testing.T) {
			calls := 0
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				t.Errorf("local guard made request: %s", r.URL.Path)
				w.WriteHeader(500)
			}))
			defer server.Close()
			// Negative delay is invalid even if configuration defaults fill other fields.
			cfg := &config.Config{App: config.App{CamundaVersion: version, Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: -time.Second, Timeout: time.Second}}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL + "/v2"}, Operate: config.API{BaseURL: server.URL}}}
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			api, err := pisvc.New(cfg, server.Client(), log)
			require.NoError(t, err)
			facade := New(nil, api, nil, log)
			for _, keys := range []typex.Keys{nil, {"bad"}, {"2251799813685249", "bad"}, {"2251799813685249"}} {
				for _, dryRun := range []bool{false, true} {
					opts := []options.FacadeOption{}
					if dryRun {
						opts = append(opts, options.WithDryRun())
					}
					plan, err := facade.PlanProcessInstanceResumption(context.Background(), keys, opts...)
					if version == toolx.V810 {
						require.ErrorIs(t, err, ferr.ErrInvalidInput)
					} else {
						require.ErrorIs(t, err, ferr.ErrUnsupported)
					}
					require.Equal(t, ProcessInstanceResumptionPlan{}, plan)
				}
			}
			if version != toolx.V810 {
				plan, err := facade.PlanProcessInstanceResumption(context.Background(), typex.Keys{"2251799813685249"}, options.WithNoWait(), options.WithDryRun())
				require.ErrorIs(t, err, ferr.ErrUnsupported)
				require.Equal(t, ProcessInstanceResumptionPlan{}, plan)
			}
			require.Zero(t, calls)
		})
	}
}

// TestProcessInstanceResumptionPlanFactoryScope composes real factory traversal and
// execution against HTTP fakes: no tenant/state pruning, no ancestor escalation,
// retained child limit, no mutation during planning, and frozen eligible execution only.
func TestProcessInstanceResumptionPlanFactoryScope(t *testing.T) {
	t.Parallel()
	const parent = "2251799813685249"
	const child = "2251799813685250"
	const grandchild = "2251799813685251"
	for _, mode := range []string{"wait", "no wait", "ineligible", "invalid timing", "dry run", "dry run no wait", "child only", "missing seed", "forbidden children"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			reads, searches, mutations := 0, 0, 0
			planning := true
			selected := typex.Keys{parent, child, parent}
			seed := parent
			if mode == "child only" {
				selected, seed = typex.Keys{child}, child
			}
			submitted := []string{}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search" {
					searches++
					require.True(t, planning, "execution must not rediscover")
					if mode == "forbidden children" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"detail":"fixture denied discovery"}`)
						return
					}
					var body struct {
						Filter map[string]json.RawMessage `json:"filter"`
						Page   struct {
							Limit int `json:"limit"`
						} `json:"page"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.NotContains(t, body.Filter, "tenantId")
					require.NotContains(t, body.Filter, "state")
					require.Equal(t, 1000, body.Page.Limit)
					var key string
					require.NoError(t, json.Unmarshal(body.Filter["parentProcessInstanceKey"], &key))
					switch key {
					case parent:
						fmt.Fprintf(w, `{"items":[{"processInstanceKey":%q,"parentProcessInstanceKey":%q,"state":"COMPLETED"}],"page":{"totalItems":1}}`, child, parent)
					case child:
						state := "SUSPENDED"
						if mode == "ineligible" || mode == "invalid timing" {
							state = "ACTIVE"
						}
						fmt.Fprintf(w, `{"items":[{"processInstanceKey":%q,"parentProcessInstanceKey":%q,"state":%q}],"page":{"totalItems":1}}`, grandchild, child, state)
					case grandchild:
						fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
					default:
						t.Errorf("unexpected parent search: %q", key)
						w.WriteHeader(500)
					}
					return
				}
				path := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
				if r.Method == http.MethodPost && path == grandchild+"/resumption" {
					require.False(t, planning, "planning must never submit")
					mutations++
					submitted = append(submitted, grandchild)
					w.WriteHeader(204)
					return
				}
				if r.Method == http.MethodGet && (path == seed || path == grandchild) {
					reads++
					if planning {
						require.Equal(t, seed, path, "descendants reuse child observations")
						if mode == "missing seed" {
							w.WriteHeader(404)
							fmt.Fprint(w, `{"detail":"fixture missing seed"}`)
							return
						}
					} else {
						require.Equal(t, grandchild, path)
						require.Equal(t, 1, mutations)
					}
					// A parent outside the selection must never be fetched via ancestry.
					fmt.Fprintf(w, `{"processInstanceKey":%q,"parentProcessInstanceKey":"2251799813685248","state":"ACTIVE"}`, path)
					return
				}
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(500)
			}))
			defer server.Close()
			backoff := config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Millisecond, Timeout: time.Second}
			opts := []options.FacadeOption{options.WithVerbose()}
			if mode == "no wait" || mode == "ineligible" || mode == "dry run no wait" {
				backoff.InitialDelay = -time.Second
				opts = append(opts, options.WithNoWait())
			}
			if mode == "invalid timing" {
				backoff.InitialDelay = -time.Second
			}
			if mode == "dry run" || mode == "dry run no wait" {
				opts = append(opts, options.WithDryRun())
			}
			cfg := &config.Config{App: config.App{CamundaVersion: toolx.V810, Tenant: "must-not-filter-explicit-scope", Backoff: backoff}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL + "/v2"}}}
			var logs bytes.Buffer
			log := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			api, err := pisvc.New(cfg, server.Client(), log)
			require.NoError(t, err)
			facade := New(nil, api, nil, log)
			plan, err := facade.PlanProcessInstanceResumption(context.Background(), selected, opts...)
			if mode == "invalid timing" {
				require.ErrorIs(t, err, ferr.ErrInvalidInput)
				require.Equal(t, ProcessInstanceResumptionPlan{}, plan)
				require.Zero(t, reads)
				require.Zero(t, searches)
				require.Zero(t, mutations)
				return
			}
			if mode == "missing seed" || mode == "forbidden children" {
				expected := ferr.ErrNotFound
				if mode == "forbidden children" {
					expected = ferr.ErrLocalPrecondition
				}
				require.ErrorIs(t, err, expected)
				require.Equal(t, ProcessInstanceResumptionPlan{}, plan)
				require.Zero(t, mutations)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, reads)
			wantSeeds := typex.Keys{parent, child}
			wantOrder := typex.Keys{parent, child, grandchild}
			if mode == "child only" {
				wantSeeds = typex.Keys{child}
				wantOrder = typex.Keys{child, grandchild}
			}
			require.Equal(t, len(wantOrder), searches)
			require.Zero(t, mutations)
			require.Equal(t, wantSeeds, plan.RequestedKeys)
			require.Len(t, plan.Instances, len(wantOrder))
			for i, item := range plan.Instances {
				require.Equal(t, wantOrder[i], item.Key)
				if item.Key != grandchild {
					require.Equal(t, ProcessInstanceResumptionPlanExcluded, item.Classification)
				}
			}
			require.Contains(t, logs.String(), "level=DEBUG")
			require.NotContains(t, logs.String(), "level=INFO")
			if mode == "ineligible" {
				require.Empty(t, plan.TargetKeys)
				require.NotNil(t, plan.TargetKeys)
				raw, err := json.Marshal(plan)
				require.NoError(t, err)
				require.JSONEq(t, `{"requestedKeys":["2251799813685249","2251799813685250"],"instances":[{"key":"2251799813685249","observedState":"ACTIVE","classification":"excluded"},{"key":"2251799813685250","observedState":"COMPLETED","classification":"excluded"},{"key":"2251799813685251","observedState":"ACTIVE","classification":"excluded"}],"targetKeys":[]}`, string(raw))
				return
			}
			require.Equal(t, typex.Keys{grandchild}, plan.TargetKeys)
			if mode == "dry run" || mode == "dry run no wait" {
				return
			}
			mu.Lock()
			planning = false
			mu.Unlock()
			result, err := facade.ResumeProcessInstances(context.Background(), slices.Clone(plan.TargetKeys), 2, opts...)
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			require.Equal(t, []string{grandchild}, submitted)
			require.Equal(t, ResumptionSubmissionAccepted, result.Items[0].Submission)
			if mode == "no wait" {
				require.Equal(t, ResumptionStatusAccepted, result.Items[0].Status)
				require.False(t, result.Items[0].HasObservation)
				require.Equal(t, 1, reads)
			} else {
				require.Equal(t, ResumptionStatusConfirmed, result.Items[0].Status)
				require.Equal(t, StateActive, result.Items[0].LastObservedState)
				require.Equal(t, 2, reads)
			}
			require.Equal(t, StateSuspended, plan.Instances[len(plan.Instances)-1].ObservedState, "execution must not rewrite plan evidence")
		})
	}
}
