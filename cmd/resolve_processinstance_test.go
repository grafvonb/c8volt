// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestResolveProcessInstanceCommand_DiscoversResolvesAndWaitsForConfirmation(t *testing.T) {
	var searches int
	var sawResolve bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searches++
			if searches == 1 {
				_, _ = w.Write([]byte(incidentSearchJSON(
					incidentSearchItemJSON("2251799813685249", "2251799813685250", "ACTIVE"),
					incidentSearchItemJSON("2251799813685248", "2251799813685250", "RESOLVED"),
					incidentSearchItemJSON("2251799813685247", "2251799813685260", "ACTIVE"),
				)))
				return
			}
			_, _ = w.Write([]byte(incidentSearchJSON(
				incidentSearchItemJSON("2251799813685248", "2251799813685250", "RESOLVED"),
			)))
		case "/v2/incidents/2251799813685249/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			sawResolve = true
			w.WriteHeader(http.StatusNoContent)
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"resolve", "process-instance",
		"--key", "2251799813685250",
	)

	require.True(t, sawResolve)
	require.Equal(t, 2, searches)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "resolved process-instance 2251799813685250: confirmed (1 incident(s))")
	require.Contains(t, stderr, "resolved process-instances: 1")
}

func TestResolveProcessInstancesWithPlan_ExpandsFamilyScopeAndPrompts(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var prompt string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, got string) error {
		require.True(t, autoConfirm)
		prompt = got
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	var resolvedKeys types.Keys
	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys types.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, types.Keys{"2251799813735367"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     types.Keys{"2251799813735367"},
				Collected: types.Keys{"2251799813735367", "2251799813735372"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		resolveProcessInstancesIncidents: func(_ context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (incident.ProcessInstanceResolutionResults, error) {
			resolvedKeys = append(types.Keys(nil), keys...)
			require.Zero(t, wantedWorkers)
			cfg := options.ApplyFacadeOptions(opts)
			require.False(t, cfg.DryRun)
			require.Equal(t, 2, cfg.AffectedProcessInstanceCount)
			return incident.ProcessInstanceResolutionResults{
				Operation: incident.ResolutionOperationProcessInstance,
				Total:     2,
				Confirmed: 1,
				Skipped:   1,
				Items: []incident.ProcessInstanceResolutionResult{
					{ProcessInstanceKey: "2251799813735367", Status: incident.ProcessInstanceResolutionStatusSkipped, ConfirmationStatus: "no_active_incidents"},
					{ProcessInstanceKey: "2251799813735372", Status: incident.ProcessInstanceResolutionStatusConfirmed, ResolvedIncidentKeys: []string{"2251799813735377"}},
				},
				MutationSubmitted: true,
			}, nil
		},
	}

	got, err := resolveProcessInstancesWithPlan(cmd, cli, types.Keys{"2251799813735367"}, true)

	require.NoError(t, err)
	require.Equal(t, types.Keys{"2251799813735367", "2251799813735372"}, resolvedKeys)
	require.Contains(t, prompt, "requested to resolve incidents for 1 process instance(s)")
	require.Contains(t, prompt, "2 instance(s) with 1 root instance(s) will be inspected")
	require.Contains(t, buf.String(), "resolved process-instance 2251799813735372: confirmed (1 incident(s))")
	require.Equal(t, 2, got.Total)
	require.Equal(t, 1, got.Confirmed)
}

func TestResolveProcessInstanceCommand_ParentFamilyScopeResolvesChildIncident(t *testing.T) {
	searchCounts := map[string]int{}
	var sawResolve bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813735367/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searchCounts["2251799813735367"]++
			_, _ = w.Write([]byte(incidentSearchJSON()))
		case "/v2/process-instances/2251799813735372/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searchCounts["2251799813735372"]++
			if searchCounts["2251799813735372"] == 1 {
				_, _ = w.Write([]byte(incidentSearchJSON(
					incidentSearchItemJSON("2251799813735377", "2251799813735372", "ACTIVE"),
				)))
				return
			}
			_, _ = w.Write([]byte(incidentSearchJSON()))
		case "/v2/incidents/2251799813735377/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			sawResolve = true
			w.WriteHeader(http.StatusNoContent)
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r,
				map[string]string{"2251799813735372": "2251799813735367"},
				map[string][]string{"2251799813735367": []string{"2251799813735372"}},
			) {
				return
			}
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"resolve", "pi",
		"--key", "2251799813735367",
		"--auto-confirm",
	)

	require.True(t, sawResolve)
	require.Equal(t, map[string]int{"2251799813735367": 1, "2251799813735372": 2}, searchCounts)
	require.Contains(t, output, "resolved process-instance 2251799813735367: skipped (no_active_incidents)")
	require.Contains(t, output, "resolved process-instance 2251799813735372: confirmed (1 incident(s))")
	require.Contains(t, output, "resolved process-instances: 2")
}

func TestResolveProcessInstanceCommand_AliasRepeatedKeysAndStdinDeduplicate(t *testing.T) {
	searchCounts := map[string]int{}
	resolveCounts := map[string]int{}
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searchCounts["2251799813685250"]++
			if searchCounts["2251799813685250"] == 1 {
				_, _ = w.Write([]byte(incidentSearchJSON(incidentSearchItemJSON("2251799813685249", "2251799813685250", "ACTIVE"))))
				return
			}
			_, _ = w.Write([]byte(incidentSearchJSON()))
		case "/v2/process-instances/2251799813685260/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searchCounts["2251799813685260"]++
			if searchCounts["2251799813685260"] == 1 {
				_, _ = w.Write([]byte(incidentSearchJSON(incidentSearchItemJSON("2251799813685259", "2251799813685260", "ACTIVE"))))
				return
			}
			_, _ = w.Write([]byte(incidentSearchJSON()))
		case "/v2/incidents/2251799813685249/resolution",
			"/v2/incidents/2251799813685259/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			key := incidentKeyFromPath(t, r.URL.Path, "/v2/incidents/", "/resolution")
			resolveCounts[key]++
			w.WriteHeader(http.StatusNoContent)
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTestWithStdin(t,
		"2251799813685250\n2251799813685260\n2251799813685260\n",
		"--config", cfgPath,
		"resolve", "pi",
		"--workers", "1",
		"--key", "2251799813685250",
		"-",
	)

	require.Equal(t, map[string]int{"2251799813685249": 1, "2251799813685259": 1}, resolveCounts)
	require.Equal(t, map[string]int{"2251799813685250": 2, "2251799813685260": 2}, searchCounts)
	require.Contains(t, output, "resolved process-instance 2251799813685250: confirmed")
	require.Contains(t, output, "resolved process-instance 2251799813685260: confirmed")
	require.Contains(t, output, "resolved process-instances: 2")
}

func TestResolveProcessInstanceCommand_NoActiveIncidentsReportsSkipped(t *testing.T) {
	var sawResolve bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(incidentSearchJSON(
				incidentSearchItemJSON("2251799813685249", "2251799813685250", "RESOLVED"),
			)))
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			if r.URL.Path == "/v2/incidents/2251799813685249/resolution" ||
				r.URL.Path == "/v2/process-instances/2251799813685250/incident-resolution" {
				sawResolve = true
			}
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"resolve", "pi",
		"--key", "2251799813685250",
	)

	require.False(t, sawResolve)
	require.Contains(t, output, "resolved process-instance 2251799813685250: skipped (no_active_incidents)")
	require.Contains(t, output, "resolved process-instances: 1")
}

func TestResolveProcessInstanceCommand_DryRunDiscoversIncidentsAndDoesNotMutate(t *testing.T) {
	var searches int
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searches++
			_, _ = w.Write([]byte(incidentSearchJSON(
				incidentSearchItemJSON("2251799813685249", "2251799813685250", "ACTIVE"),
				incidentSearchItemJSON("2251799813685248", "2251799813685250", "RESOLVED"),
			)))
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			t.Fatalf("unexpected request path during dry-run: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"resolve", "pi",
		"--key", "2251799813685250",
		"--dry-run",
	)

	require.Equal(t, 1, searches)
	require.Contains(t, output, "dry run: process-instance 2251799813685250 would resolve 1 incident(s)")
	require.Contains(t, output, "no changes applied")
}

func TestResolveProcessInstanceCommand_DryRunNoActiveIncidentsReportsSkipped(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(incidentSearchJSON(
				incidentSearchItemJSON("2251799813685248", "2251799813685250", "RESOLVED"),
			)))
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			t.Fatalf("unexpected request path during dry-run: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"resolve", "process-instance",
		"--key", "2251799813685250",
		"--dry-run",
	)

	require.Contains(t, output, "resolved process-instance 2251799813685250: skipped (no_active_incidents)")
	require.Contains(t, output, "no changes applied")
}

func TestResolveProcessInstanceCommand_JSONDryRunReturnsStablePlanPayload(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(incidentSearchJSON(
				incidentSearchItemJSON("2251799813685249", "2251799813685250", "ACTIVE"),
			)))
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			t.Fatalf("unexpected request path during JSON dry-run: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--json",
		"--config", cfgPath,
		"resolve", "pi",
		"--key", "2251799813685250",
		"--dry-run",
	)

	require.Empty(t, stderr)
	payload := requireDryRunEnvelopePayload(t, stdout)
	require.Equal(t, "resolveProcessInstance", payload["operation"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, false, payload["mutationSubmitted"])
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813685250", item["processInstanceKey"])
	require.Equal(t, "planned", item["status"])
	require.Equal(t, []any{"2251799813685249"}, item["attemptedIncidentKeys"])
	require.Equal(t, false, item["mutationSubmitted"])
}

func TestResolveProcessInstanceCommand_InvalidKeysAndLookupFailure(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	t.Run("invalid flag key", func(t *testing.T) {
		output, err := testx.RunCmdSubprocess(t, "TestResolveProcessInstanceCommand_InvalidFlagKeyHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":       cfgPath,
			"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{"resolve", "process-instance", "--key", "bad-key"}),
		})
		require.Error(t, err)
		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok)
		require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
		require.Contains(t, string(output), "invalid input")
		require.Contains(t, string(output), "process instance key \"bad-key\" is not a valid key")
	})

	t.Run("empty stdin", func(t *testing.T) {
		output, err := testx.RunCmdSubprocessWithStdin(t, "TestResolveProcessInstanceCommand_EmptyStdinHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":       cfgPath,
			"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{"resolve", "pi", "-"}),
		}, "\n")
		require.Error(t, err)
		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok)
		require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
		require.Contains(t, string(output), "invalid input")
		require.Contains(t, string(output), "stdin contained no keys")
	})

	t.Run("lookup failure", func(t *testing.T) {
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/process-instances/2251799813685250/incidents/search":
				require.Equal(t, http.MethodPost, r.Method)
				http.Error(w, `{"message":"lookup failed"}`, http.StatusInternalServerError)
			default:
				if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
					return
				}
				t.Fatalf("unexpected request path: %s", r.URL.Path)
			}
		}))
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output, err := testx.RunCmdSubprocess(t, "TestResolveProcessInstanceCommand_LookupFailureHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":       cfgPath,
			"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{"resolve", "pi", "--key", "2251799813685250"}),
		})
		require.Error(t, err)
		require.Contains(t, string(output), "resolve process-instance incidents")
		require.Contains(t, string(output), "lookup failed")
	})
}

func TestResolveProcessInstanceCommand_JSONRejectsVerbose(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestResolveProcessInstanceCommand_JSONRejectsVerboseHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":       cfgPath,
		"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{"--json", "resolve", "process-instance", "--key", "2251799813685250", "--dry-run", "--verbose"}),
	})
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "--json cannot be combined with --verbose for resolve process-instance")
}

func TestResolveProcessInstanceCommand_NoWaitPartialFailureRendersSuccessfulTargets(t *testing.T) {
	searchCounts := map[string]int{}
	resolveCounts := map[string]int{}
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searchCounts["2251799813685250"]++
			_, _ = w.Write([]byte(incidentSearchJSON(
				incidentSearchItemJSON("2251799813685249", "2251799813685250", "ACTIVE"),
			)))
		case "/v2/process-instances/2251799813685260/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searchCounts["2251799813685260"]++
			http.Error(w, `{"message":"lookup failed"}`, http.StatusInternalServerError)
		case "/v2/incidents/2251799813685249/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			resolveCounts["2251799813685249"]++
			w.WriteHeader(http.StatusNoContent)
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestResolveProcessInstanceCommand_PartialFailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
		"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{
			"resolve", "pi",
			"--no-wait",
			"--workers", "1",
			"--no-worker-limit",
			"--key", "2251799813685250",
			"--key", "2251799813685260",
		}),
	})
	require.Error(t, err)
	require.Equal(t, map[string]int{"2251799813685250": 1, "2251799813685260": 1}, searchCounts)
	require.Equal(t, map[string]int{"2251799813685249": 1}, resolveCounts)
	require.Contains(t, string(output), "resolved process-instance 2251799813685250: submitted (1 incident(s))")
	require.Contains(t, string(output), "resolved process-instance 2251799813685260: failed")
	require.Contains(t, string(output), "resolved process-instances: 2 (confirmed/submitted/skipped: 1, failed: 1)")
}

func TestResolveProcessInstanceCommand_FailFastStopsAfterFirstLookupFailure(t *testing.T) {
	var searches int
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searches++
			http.Error(w, `{"message":"lookup failed"}`, http.StatusInternalServerError)
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestResolveProcessInstanceCommand_FailFastHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
		"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{
			"resolve", "process-instance",
			"--workers", "1",
			"--fail-fast",
			"--key", "2251799813685250",
			"--key", "2251799813685260",
		}),
	})
	require.Error(t, err)
	require.Equal(t, 1, searches)
	require.Contains(t, string(output), "resolved process-instances: 1 (confirmed/submitted/skipped: 0, failed: 1)")
	require.NotContains(t, string(output), "2251799813685260")
}

func TestResolveProcessInstanceCommand_TimeoutReportsConfirmationFailure(t *testing.T) {
	var searches int
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685250/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			searches++
			_, _ = w.Write([]byte(incidentSearchJSON(
				incidentSearchItemJSON("2251799813685249", "2251799813685250", "ACTIVE"),
			)))
		case "/v2/incidents/2251799813685249/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusNoContent)
		default:
			if handleResolveProcessInstanceFamilyFixture(t, w, r, nil, nil) {
				return
			}
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestResolveProcessInstanceCommand_TimeoutHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
		"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{
			"resolve", "--backoff-timeout", "1ns",
			"pi",
			"--key", "2251799813685250",
		}),
	})
	require.Error(t, err)
	require.GreaterOrEqual(t, searches, 1)
	require.Contains(t, string(output), "partial failure")
	require.Contains(t, string(output), "context deadline exceeded")
}

func TestResolveProcessInstanceCommand_LookupFailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	cfgPath := os.Getenv("C8VOLT_TEST_CONFIG")
	args := unmarshalResolveArgsFromEnv(t, "C8VOLT_TEST_RESOLVE_ARGS")
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs(append([]string{"--config", cfgPath}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func TestResolveProcessInstanceCommand_InvalidFlagKeyHelper(t *testing.T) {
	runResolveProcessInstanceHelperFromEnv(t)
}

func TestResolveProcessInstanceCommand_EmptyStdinHelper(t *testing.T) {
	runResolveProcessInstanceHelperFromEnv(t)
}

func TestResolveProcessInstanceCommand_JSONRejectsVerboseHelper(t *testing.T) {
	runResolveProcessInstanceHelperFromEnv(t)
}

func TestResolveProcessInstanceCommand_PartialFailureHelper(t *testing.T) {
	runResolveProcessInstanceHelperFromEnv(t)
}

func TestResolveProcessInstanceCommand_FailFastHelper(t *testing.T) {
	runResolveProcessInstanceHelperFromEnv(t)
}

func TestResolveProcessInstanceCommand_TimeoutHelper(t *testing.T) {
	runResolveProcessInstanceHelperFromEnv(t)
}

func runResolveProcessInstanceHelperFromEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	cfgPath := os.Getenv("C8VOLT_TEST_CONFIG")
	args := unmarshalResolveArgsFromEnv(t, "C8VOLT_TEST_RESOLVE_ARGS")
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs(append([]string{"--config", cfgPath}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func incidentSearchJSON(items ...string) string {
	out := `{"items":[`
	for i, item := range items {
		if i > 0 {
			out += ","
		}
		out += item
	}
	out += `],"page":{"totalItems":`
	out += strconv.Itoa(len(items))
	out += `,"hasMoreTotalItems":false}}`
	return out
}

func incidentSearchItemJSON(incidentKey string, processInstanceKey string, state string) string {
	return `{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"2251799813685300","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"` + incidentKey + `","processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processInstanceKey":"` + processInstanceKey + `","state":"` + state + `","tenantId":"<default>"}`
}

func handleResolveProcessInstanceFamilyFixture(t *testing.T, w http.ResponseWriter, r *http.Request, parents map[string]string, children map[string][]string) bool {
	t.Helper()
	if parents == nil {
		parents = map[string]string{}
	}
	if children == nil {
		children = map[string][]string{}
	}

	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/") {
		key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
		if key == "" || strings.Contains(key, "/") {
			return false
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resolveProcessInstanceJSON(key, parents[key])))
		return true
	}

	if r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search" {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		payload := string(body)
		if !strings.Contains(payload, `"parentProcessInstanceKey"`) {
			return false
		}
		for parentKey, childKeys := range children {
			if strings.Contains(payload, fmt.Sprintf(`"parentProcessInstanceKey":"%s"`, parentKey)) {
				items := make([]string, 0, len(childKeys))
				for _, childKey := range childKeys {
					items = append(items, resolveProcessInstanceJSON(childKey, parentKey))
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(resolveProcessInstanceSearchJSON(items...)))
				return true
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resolveProcessInstanceSearchJSON()))
		return true
	}

	return false
}

func resolveProcessInstanceSearchJSON(items ...string) string {
	out := `{"items":[`
	for i, item := range items {
		if i > 0 {
			out += ","
		}
		out += item
	}
	out += `],"page":{"totalItems":`
	out += strconv.Itoa(len(items))
	out += `,"hasMoreTotalItems":false}}`
	return out
}

func resolveProcessInstanceJSON(key string, parentKey string) string {
	parentField := ""
	if parentKey != "" {
		parentField = fmt.Sprintf(`,"parentProcessInstanceKey":"%s"`, parentKey)
	}
	return fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processDefinitionName":"demo","processDefinitionVersion":1,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"<default>"%s}`, key, parentField)
}
