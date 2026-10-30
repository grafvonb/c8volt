// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cluster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/batchoperation"
	"github.com/grafvonb/c8volt/internal/services/element"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/incident"
	"github.com/grafvonb/c8volt/internal/services/job"
	"github.com/grafvonb/c8volt/internal/services/processdefinition"
	"github.com/grafvonb/c8volt/internal/services/processinstance"
	pi88 "github.com/grafvonb/c8volt/internal/services/processinstance/v88"
	"github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/internal/services/tenant"
	"github.com/grafvonb/c8volt/internal/services/usertask"
	ut88 "github.com/grafvonb/c8volt/internal/services/usertask/v88"
	ut89 "github.com/grafvonb/c8volt/internal/services/usertask/v89"
	"github.com/grafvonb/c8volt/internal/services/variable"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestVersionCheckConstructorWiring exercises actual generated requests across every
// constructor, including nested variable services, with and without invocation gates.
func TestVersionCheckConstructorWiring(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		build func(*testing.T, *config.Config, *http.Client) func(context.Context) error
	}{
		{"batchoperation", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := batchoperation.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error { return svc.CheckReadAccess(ctx) }
		}},
		{"cluster", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.GetClusterLicense(ctx)
				return err
			}
		}},
		{"element", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := element.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.GetElement(ctx, "123")
				return err
			}
		}},
		{"incident", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := incident.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.GetIncident(ctx, "123")
				return err
			}
		}},
		{"job", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := job.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.GetJob(ctx, "123")
				return err
			}
		}},
		{"processdefinition", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := processdefinition.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.GetProcessDefinition(ctx, "123")
				return err
			}
		}},
		{"processinstance", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := processinstance.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.GetProcessInstance(ctx, "123")
				return err
			}
		}},
		{"resource", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := resource.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.Get(ctx, "123")
				return err
			}
		}},
		{"tenant", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := tenant.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.GetTenant(ctx, "tenant")
				return err
			}
		}},
		{"usertask", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := usertask.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.GetNativeUserTask(ctx, "123")
				return err
			}
		}},
		{"variable", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := variable.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.SearchProcessInstanceVariables(ctx, "123")
				return err
			}
		}},
		{"nested variable", func(t *testing.T, cfg *config.Config, client *http.Client) func(context.Context) error {
			svc, err := processinstance.New(cfg, client, versionCheckLogger())
			require.NoError(t, err)
			return func(ctx context.Context) error {
				_, err := svc.SearchProcessInstanceVariables(ctx, "123")
				return err
			}
		}},
	}
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, tc := range cases {
			for _, gated := range []bool{false, true} {
				mode := "ungated"
				if gated {
					mode = "gated"
				}
				t.Run(version.String()+"/"+tc.name+"/"+mode, func(t *testing.T) {
					t.Parallel()
					cfg := versionCheckConfig(version)
					cfg.APIs.Operate.BaseURL = "https://isolated.invalid/v1"
					operationErr := errors.New("operation failure retained")
					probes, operations, callbacks := 0, 0, 0
					client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
						if req.URL.Path == "/v2/topology" {
							probes++
							return versionCheckResponse(req, 403, `{}`), nil
						}
						operations++
						if gated {
							require.Equal(t, 1, callbacks, "operation must follow the completed advisory callback")
						}
						return nil, operationErr
					})}
					checker := NewVersionCheck(cfg, client, versionCheckLogger(), func(d.VersionCheckOutcome) { callbacks++ })
					ctx := context.Background()
					if gated {
						ctx = httpc.WithBeforeRequestGate(ctx, checker)
					}
					run := tc.build(t, cfg, client)
					require.Zero(t, probes)
					require.Zero(t, operations)
					for range 2 {
						require.ErrorIs(t, run(ctx), operationErr)
					}
					require.Equal(t, 2, operations)
					if gated {
						require.Equal(t, 1, probes)
						outcome, complete := checker.Outcome()
						require.True(t, complete)
						require.Equal(t, d.VersionCheckAccessDenied, outcome.FailureReason)
					} else {
						require.Zero(t, probes)
						require.Zero(t, callbacks)
					}
				})
			}
		}
	}
}

// TestVersionCheckLegacyClients checks all three additional generated clients directly
// so a preceding primary request cannot mask a missing wrapper on a fallback client.
func TestVersionCheckLegacyClients(t *testing.T) {
	t.Parallel()
	for _, route := range []string{"operate88", "tasklist88", "tasklist89"} {
		t.Run(route, func(t *testing.T) {
			t.Parallel()
			cfg := versionCheckConfig(toolx.V88)
			if route == "tasklist89" {
				cfg.App.CamundaVersion = toolx.V89
			}
			cfg.APIs.Operate.BaseURL = "https://isolated.invalid/v1"
			cfg.APIs.Tasklist.BaseURL = "https://isolated.invalid/v1"
			calls := []string{}
			client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
				calls = append(calls, req.URL.Path)
				if req.URL.Path == "/v2/topology" {
					return versionCheckResponse(req, 200, `{"gatewayVersion":"8.8"}`), nil
				}
				return versionCheckResponse(req, 200, `{}`), nil
			})}
			checker := NewVersionCheck(cfg, client, versionCheckLogger(), nil)
			ctx := httpc.WithBeforeRequestGate(context.Background(), checker)
			switch route {
			case "operate88":
				svc, err := pi88.New(cfg, client, versionCheckLogger())
				require.NoError(t, err)
				_, err = svc.ClientOperate().DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx, 123)
				require.NoError(t, err)
			case "tasklist88":
				svc, err := ut88.New(cfg, client, versionCheckLogger())
				require.NoError(t, err)
				_, err = svc.ClientTasklist().GetTaskByIdWithResponse(ctx, "123")
				require.NoError(t, err)
			case "tasklist89":
				svc, err := ut89.New(cfg, client, versionCheckLogger())
				require.NoError(t, err)
				_, err = svc.ClientTasklist().GetTaskByIdWithResponse(ctx, "123")
				require.NoError(t, err)
			}
			require.Len(t, calls, 2)
			require.Equal(t, "/v2/topology", calls[0])
			require.Contains(t, calls[1], "/v1/")
		})
	}
}

// TestVersionCheckWiredTimeout gives every real adapter a fresh operational HTTP
// deadline after its short-budget topology probe expires, retaining parent limits.
func TestVersionCheckWiredTimeout(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, parentExpires := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/parent=%t", version, parentExpires), func(t *testing.T) {
				t.Parallel()
				cfg := versionCheckConfig(version)
				const timeout = 100 * time.Millisecond
				cfg.HTTP.Timeout = timeout.String()
				probes, operations := 0, 0
				var probeDeadline time.Time
				svc, err := httpc.New(cfg, versionCheckLogger(), func(s *httpc.Service) {
					s.Client().Transport = versionCheckTransport(func(req *http.Request) (*http.Response, error) {
						deadline, ok := req.Context().Deadline()
						require.True(t, ok)
						if req.URL.Path == "/v2/topology" {
							probes++
							probeDeadline = deadline
							<-req.Context().Done()
							return nil, req.Context().Err()
						}
						operations++
						require.Greater(t, deadline.Sub(probeDeadline), timeout/2)
						require.Greater(t, time.Until(deadline), timeout/2)
						return versionCheckResponse(req, 200, `{}`), nil
					})
				})
				require.NoError(t, err)
				checker := NewVersionCheck(cfg, svc.Client(), versionCheckLogger(), nil)
				parent := context.Background()
				if parentExpires {
					var cancel context.CancelFunc
					parent, cancel = context.WithTimeout(parent, 30*time.Millisecond)
					defer cancel()
				}
				ctx := httpc.WithBeforeRequestGate(svc.ToContext(parent), checker)
				api, err := New(cfg, svc.Client(), versionCheckLogger())
				require.NoError(t, err)
				_, err = api.GetClusterLicense(ctx)
				if parentExpires {
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.Zero(t, operations)
				} else {
					require.NoError(t, err)
					require.NoError(t, parent.Err())
					require.Equal(t, 1, operations)
				}
				require.Equal(t, 1, probes)
				outcome, complete := checker.Outcome()
				require.True(t, complete)
				require.Equal(t, d.VersionCheckTimeout, outcome.FailureReason)
			})
		}
	}
}

// TestVersionCheckWiredWaiterCancellation proves a generated-client waiter can
// abandon the shared probe while another caller completes it for later requests.
func TestVersionCheckWiredWaiterCancellation(t *testing.T) {
	t.Parallel()
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var probes, operations atomic.Int32
	cfg := versionCheckConfig(toolx.V89)
	client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/v2/topology" {
			probes.Add(1)
			close(entered)
			<-release
			return versionCheckResponse(req, 200, `{"gatewayVersion":"8.9"}`), nil
		}
		operations.Add(1)
		return versionCheckResponse(req, 200, `{}`), nil
	})}
	checker := NewVersionCheck(cfg, client, versionCheckLogger(), nil)
	ctx := httpc.WithBeforeRequestGate(context.Background(), checker)
	api, err := New(cfg, client, versionCheckLogger())
	require.NoError(t, err)
	winner := make(chan error, 1)
	go func() { _, err := api.GetClusterLicense(ctx); winner <- err }()
	awaitVersionCheck(t, entered)
	parent, cancel := context.WithCancel(ctx)
	defer cancel()
	waiter := &versionCheckWaitContext{Context: parent, entered: make(chan struct{})}
	waiterResult := make(chan error, 1)
	go func() { _, err := api.GetClusterLicense(waiter); waiterResult <- err }()
	awaitVersionCheck(t, waiter.entered)
	cancel()
	select {
	case err := <-waiterResult:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("waiter did not cancel")
	}
	require.Zero(t, operations.Load())
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-winner:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("winner did not complete")
	}
	_, err = api.GetClusterLicense(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, probes.Load())
	require.EqualValues(t, 2, operations.Load())
}

// TestVersionCheckWiredRetryAndRedirect keeps both controls on ordinary generated
// reads while the topology probe refuses redirects and never retries a 503.
func TestVersionCheckWiredRetryAndRedirect(t *testing.T) {
	t.Parallel()
	for _, probeStatus := range []int{302, 503} {
		t.Run(strconv.Itoa(probeStatus), func(t *testing.T) {
			t.Parallel()
			cfg := versionCheckConfig(toolx.V89)
			calls := []string{}
			retries, redirects := 0, 0
			svc, err := httpc.New(cfg, versionCheckLogger(), func(s *httpc.Service) {
				s.Client().CheckRedirect = func(*http.Request, []*http.Request) error { redirects++; return nil }
				s.Client().Transport = versionCheckTransport(func(req *http.Request) (*http.Response, error) {
					calls = append(calls, req.URL.Path)
					status, body := 200, `{}`
					switch req.URL.Path {
					case "/v2/topology":
						status = probeStatus
					case "/v2/license":
						retries++
						if retries == 1 {
							status = 503
						} else {
							status = 302
						}
					case "/final":
					default:
						t.Errorf("unexpected path %s", req.URL.Path)
					}
					resp := versionCheckResponse(req, status, body)
					resp.Header.Set("Location", "/final")
					resp.Header.Set("Retry-After", "0")
					return resp, nil
				})
			})
			require.NoError(t, err)
			checker := NewVersionCheck(cfg, svc.Client(), versionCheckLogger(), nil)
			ctx := httpc.WithBeforeRequestGate(svc.ToContext(context.Background()), checker)
			api, err := New(cfg, svc.Client(), versionCheckLogger())
			require.NoError(t, err)
			_, err = api.GetClusterLicense(ctx)
			require.NoError(t, err)
			require.Equal(t, []string{"/v2/topology", "/v2/license", "/v2/license", "/final"}, calls)
			require.Equal(t, 1, redirects)
			outcome, _ := checker.Outcome()
			require.Equal(t, d.VersionCheckUnavailable, outcome.FailureReason)
		})
	}
}

// TestVersionCheckWiredMutationEvidence exercises the real limiter and mutation
// recorder: a locally unadmitted probe cannot classify a sent operation as unsent.
func TestVersionCheckWiredMutationEvidence(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"accepted", "ambiguous", "retry unadmitted"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			cfg := versionCheckConfig(toolx.V89)
			cfg.HTTP.Timeout = "3s"
			// The next slot is beyond the probe's two-second cap but within the
			// operation's independent three-second budget. Admission rejects immediately.
			cfg.HTTP.RequestRate = 0.45
			var output bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
			calls := []string{}
			operationErr := errors.New("ambiguous submitted operation")
			svc, err := httpc.New(cfg, logger, func(s *httpc.Service) {
				s.Client().Transport = versionCheckTransport(func(req *http.Request) (*http.Response, error) {
					calls = append(calls, req.URL.Path)
					if req.URL.Path == "/warmup" {
						return versionCheckResponse(req, 200, `{}`), nil
					}
					require.Equal(t, "/v2/element-instances/123/variables", req.URL.Path)
					body, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					require.NoError(t, req.Body.Close())
					require.JSONEq(t, `{"local":true,"variables":{"sentinel":"original"}}`, string(body))
					if mode == "ambiguous" {
						return nil, operationErr
					}
					if mode == "retry unadmitted" {
						response := versionCheckResponse(req, 503, `{"title":"RESOURCE_EXHAUSTED"}`)
						response.Header.Set("Retry-After", "0")
						return response, nil
					}
					return versionCheckResponse(req, 204, ""), nil
				})
			}, httpc.WithDiagnostics())
			require.NoError(t, err)
			parent, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			ctx, earlier := httpc.BeginRequestRateAttempt(svc.ToContext(parent))
			require.NotNil(t, earlier)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://isolated.invalid/warmup", nil)
			require.NoError(t, err)
			resp, err := svc.Client().Do(req)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			checker := NewVersionCheck(cfg, svc.Client(), logger, nil)
			ctx = httpc.WithBeforeRequestGate(ctx, checker)
			api, err := variable.New(cfg, svc.Client(), logger)
			require.NoError(t, err)
			result, err := api.UpdateScopeVariables(ctx, "123", map[string]any{"sentinel": "original"})
			if mode == "ambiguous" {
				require.ErrorIs(t, err, operationErr)
				require.Equal(t, d.ScopeVariableSubmissionUnknown, result.SubmissionStatus)
				require.False(t, result.Accepted)
			} else if mode == "retry unadmitted" {
				require.True(t, httpc.IsRequestRateWaitError(err))
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Equal(t, d.ScopeVariableSubmissionUnknown, result.SubmissionStatus)
				require.Equal(t, 503, result.StatusCode)
				require.False(t, result.Accepted)
			} else {
				require.NoError(t, err)
				require.Equal(t, d.ScopeVariableSubmissionAccepted, result.SubmissionStatus)
				require.True(t, result.Accepted)
			}
			require.NoError(t, earlier.Failure())
			outcome, complete := checker.Outcome()
			require.True(t, complete)
			require.Equal(t, d.VersionCheckTimeout, outcome.FailureReason)
			require.Equal(t, []string{"/warmup", "/v2/element-instances/123/variables"}, calls)
			require.Equal(t, 2, strings.Count(output.String(), "api #"), "unadmitted probe must not allocate a diagnostic network start")
			require.Contains(t, output.String(), "api #2 PUT")
			require.Equal(t, mode == "retry unadmitted", httpc.IsRequestRateWaitError(err))
		})
	}
}

// TestVersionCheckSharedWorkflows keeps one completed decision across concurrent
// batch workers, subsequent pages, nested variable calls and a polling lifecycle.
func TestVersionCheckSharedWorkflows(t *testing.T) {
	t.Parallel()
	cfg := versionCheckConfig(toolx.V89)
	var probes, instances, pages, variables, polls atomic.Int32
	client := &http.Client{Transport: versionCheckTransport(func(req *http.Request) (*http.Response, error) {
		body := `{}`
		switch {
		case req.URL.Path == "/v2/topology":
			probes.Add(1)
			body = `{"gatewayVersion":"8.9"}`
		case strings.HasPrefix(req.URL.Path, "/v2/process-instances/"):
			instances.Add(1)
			body = fmt.Sprintf(`{"processInstanceKey":%q,"state":"COMPLETED"}`, strings.TrimPrefix(req.URL.Path, "/v2/process-instances/"))
		case req.URL.Path == "/v2/process-definitions/search":
			page := pages.Add(1)
			body = fmt.Sprintf(`{"items":[{"processDefinitionKey":"123"}],"page":{"endCursor":"cursor-%d","hasMoreTotalItems":%t,"totalItems":2}}`, page, page == 1)
		case req.URL.Path == "/v2/variables/search":
			variables.Add(1)
			body = `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
		case req.URL.Path == "/v2/batch-operations/123":
			state := "ACTIVE"
			if polls.Add(1) > 1 {
				state = "COMPLETED"
			}
			body = fmt.Sprintf(`{"batchOperationKey":"123","batchOperationType":"CANCEL_PROCESS_INSTANCE","state":%q}`, state)
		default:
			return nil, fmt.Errorf("unexpected fixture path %s", req.URL.Path)
		}
		return versionCheckResponse(req, 200, body), nil
	})}
	checker := NewVersionCheck(cfg, client, versionCheckLogger(), nil)
	parent, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx := httpc.WithBeforeRequestGate(parent, checker)
	pi, err := processinstance.New(cfg, client, versionCheckLogger())
	require.NoError(t, err)
	items, err := pi.GetProcessInstances(ctx, typex.Keys{"1", "2", "3", "4"}, 4)
	require.NoError(t, err)
	require.Len(t, items, 4)
	pd, err := processdefinition.New(cfg, client, versionCheckLogger())
	require.NoError(t, err)
	first, err := pd.SearchProcessDefinitionsPage(ctx, d.ProcessDefinitionFilter{}, d.ProcessDefinitionPageRequest{Size: 1})
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	second, err := pd.SearchProcessDefinitionsPage(ctx, d.ProcessDefinitionFilter{}, d.ProcessDefinitionPageRequest{Size: 1, After: first.EndCursor})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	values, err := pi.SearchProcessInstanceVariables(ctx, "1")
	require.NoError(t, err)
	require.Empty(t, values)
	batch, err := batchoperation.New(cfg, client, versionCheckLogger())
	require.NoError(t, err)
	_, err = batch.WaitForCompletion(ctx, "123")
	require.NoError(t, err)
	require.EqualValues(t, 1, probes.Load())
	require.EqualValues(t, 4, instances.Load())
	require.EqualValues(t, 2, pages.Load())
	require.EqualValues(t, 1, variables.Load())
	require.EqualValues(t, 2, polls.Load())
}
