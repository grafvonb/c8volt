// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestResolveIncidentCommand_SubmitsResolutionAndWaitsForConfirmation(t *testing.T) {
	var sawResolve bool
	var sawWait bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/incidents/2251799813685249/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			sawResolve = true
			w.WriteHeader(http.StatusNoContent)
		case "/v2/incidents/2251799813685249":
			require.Equal(t, http.MethodGet, r.Method)
			sawWait = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(incidentResultJSON("2251799813685249", "2251799813685250", "RESOLVED")))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"resolve", "incident",
		"--key", "2251799813685249",
	)

	require.True(t, sawResolve)
	require.True(t, sawWait)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "resolved incident 2251799813685249: confirmed")
	require.Contains(t, stderr, "resolved: 1")
}

func TestResolveIncidentCommand_AliasRepeatedKeysAndStdinDeduplicate(t *testing.T) {
	resolveCounts := map[string]int{}
	waitCounts := map[string]int{}
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/incidents/2251799813685249/resolution",
			"/v2/incidents/2251799813685250/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			key := incidentKeyFromPath(t, r.URL.Path, "/v2/incidents/", "/resolution")
			resolveCounts[key]++
			w.WriteHeader(http.StatusNoContent)
		case "/v2/incidents/2251799813685249",
			"/v2/incidents/2251799813685250":
			require.Equal(t, http.MethodGet, r.Method)
			key := incidentKeyFromPath(t, r.URL.Path, "/v2/incidents/", "")
			waitCounts[key]++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(incidentResultJSON(key, "2251799813685260", "RESOLVED")))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTestWithStdin(t,
		"2251799813685249\n2251799813685250\n2251799813685250\n",
		"--config", cfgPath,
		"resolve", "inc",
		"--workers", "1",
		"--key", "2251799813685249",
		"-",
	)

	require.Equal(t, map[string]int{
		"2251799813685249": 1,
		"2251799813685250": 1,
	}, resolveCounts)
	require.Equal(t, map[string]int{
		"2251799813685249": 1,
		"2251799813685250": 1,
	}, waitCounts)
	require.Contains(t, output, "resolved incident 2251799813685249: confirmed")
	require.Contains(t, output, "resolved incident 2251799813685250: confirmed")
	require.Contains(t, output, "resolved: 2")
}

func TestResolveIncidentCommand_DryRunLoadsKeysAndDoesNotMutate(t *testing.T) {
	getCounts := map[string]int{}
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/incidents/2251799813685249",
			"/v2/incidents/2251799813685250":
			require.Equal(t, http.MethodGet, r.Method)
			key := incidentKeyFromPath(t, r.URL.Path, "/v2/incidents/", "")
			getCounts[key]++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(incidentResultJSON(key, "2251799813685260", "ACTIVE")))
		default:
			t.Fatalf("unexpected request path during dry-run: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTestWithStdin(t,
		"2251799813685249\n2251799813685250\n2251799813685250\n",
		"--config", cfgPath,
		"resolve", "inc",
		"--dry-run",
		"--workers", "1",
		"--key", "2251799813685249",
		"-",
	)

	require.Equal(t, map[string]int{
		"2251799813685249": 1,
		"2251799813685250": 1,
	}, getCounts)
	require.Contains(t, output, "dry run: incident 2251799813685249 would be resolved")
	require.Contains(t, output, "dry run: incident 2251799813685250 would be resolved")
	require.Contains(t, output, "no changes applied")
}

func TestResolveIncidentCommand_JSONDryRunReturnsStablePlanPayload(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/incidents/2251799813685249", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(incidentResultJSON("2251799813685249", "2251799813685250", "ACTIVE")))
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--json",
		"--config", cfgPath,
		"resolve", "incident",
		"--key", "2251799813685249",
		"--dry-run",
	)

	require.Empty(t, stderr)
	payload := requireDryRunEnvelopePayload(t, stdout)
	require.Equal(t, "resolveIncident", payload["operation"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, false, payload["mutationSubmitted"])
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813685249", item["incidentKey"])
	require.Equal(t, "planned", item["status"])
	require.Equal(t, true, item["wouldResolve"])
	require.Equal(t, false, item["mutationSubmitted"])
}

func TestResolveIncidentCommand_InvalidKeysFailBeforeMutation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	t.Run("invalid flag key", func(t *testing.T) {
		output, err := testx.RunCmdSubprocess(t, "TestResolveIncidentCommand_InvalidFlagKeyHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":       cfgPath,
			"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{"resolve", "incident", "--key", "bad-key"}),
		})
		require.Error(t, err)
		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok)
		require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
		require.Contains(t, string(output), "invalid input")
		require.Contains(t, string(output), "incident key \"bad-key\" is not a valid key")
	})

	t.Run("empty stdin", func(t *testing.T) {
		output, err := testx.RunCmdSubprocessWithStdin(t, "TestResolveIncidentCommand_EmptyStdinHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":       cfgPath,
			"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{"resolve", "inc", "-"}),
		}, "\n")
		require.Error(t, err)
		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok)
		require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
		require.Contains(t, string(output), "invalid input")
		require.Contains(t, string(output), "stdin contained no keys")
	})
}

func TestResolveIncidentCommand_JSONRejectsVerbose(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestResolveIncidentCommand_JSONRejectsVerboseHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":       cfgPath,
		"C8VOLT_TEST_RESOLVE_ARGS": marshalResolveArgsForEnv(t, []string{"--json", "resolve", "incident", "--key", "2251799813685249", "--dry-run", "--verbose"}),
	})
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "--json cannot be combined with --verbose for resolve incident")
}

func TestResolveIncidentCommand_InvalidFlagKeyHelper(t *testing.T) {
	runResolveIncidentHelperFromEnv(t)
}

func TestResolveIncidentCommand_EmptyStdinHelper(t *testing.T) {
	runResolveIncidentHelperFromEnv(t)
}

func TestResolveIncidentCommand_JSONRejectsVerboseHelper(t *testing.T) {
	runResolveIncidentHelperFromEnv(t)
}

func runResolveIncidentHelperFromEnv(t *testing.T) {
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

func marshalResolveArgsForEnv(t *testing.T, args []string) string {
	t.Helper()
	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

func unmarshalResolveArgsFromEnv(t *testing.T, name string) []string {
	t.Helper()
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv(name)), &args))
	return args
}

func incidentResultJSON(incidentKey string, processInstanceKey string, state string) string {
	return `{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"2251799813685300","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"` + incidentKey + `","processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processInstanceKey":"` + processInstanceKey + `","state":"` + state + `","tenantId":"<default>"}`
}

func incidentKeyFromPath(t *testing.T, path string, prefix string, suffix string) string {
	t.Helper()
	require.Contains(t, path, prefix)
	key := path[len(prefix):]
	if suffix != "" {
		require.Contains(t, key, suffix)
		key = key[:len(key)-len(suffix)]
	}
	return key
}
