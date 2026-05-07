// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestUpdatePICommand_SubmitsV88UpdateAndConfirmsVariables(t *testing.T) {
	var sawUpdate bool
	var sawConfirmation bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables":
			require.Equal(t, http.MethodPut, r.Method)
			sawUpdate = true
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"foo": "bar"}, body["variables"])
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			sawConfirmation = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"update", "pi",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
	)

	require.True(t, sawUpdate)
	require.True(t, sawConfirmation)
	envelope := requireUpdateProcessInstanceEnvelope(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "update process-instance", envelope["command"])
	item := firstUpdateResultItem(t, envelope)
	require.Equal(t, "2251799813711967", item["key"])
	require.Equal(t, "confirmed", item["status"])
	require.Equal(t, true, item["mutationAccepted"])
	require.Equal(t, "confirmed", item["confirmationStatus"])
}

func TestUpdateProcessInstanceCommand_MultipleRepeatedKeysApplyOneVarsPayloadToEachUniqueKey(t *testing.T) {
	var mu sync.Mutex
	updates := map[string]int{}
	confirmations := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables",
			"/v2/element-instances/2251799813711968/variables":
			require.Equal(t, http.MethodPut, r.Method)
			key := keyFromElementInstanceVariablesPath(t, r.URL.Path)
			mu.Lock()
			updates[key]++
			mu.Unlock()
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"foo": "bar"}, body["variables"])
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			mu.Lock()
			confirmations++
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"},{"name":"foo","value":"\"bar\"","variableKey":"902","processInstanceKey":"2251799813711968","scopeKey":"2251799813711968","tenantId":"<default>"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"update", "process-instance",
		"--key", "2251799813711967",
		"--key", "2251799813711968",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
	)

	mu.Lock()
	gotUpdates := map[string]int{}
	for key, count := range updates {
		gotUpdates[key] = count
	}
	gotConfirmations := confirmations
	mu.Unlock()
	require.Equal(t, map[string]int{
		"2251799813711967": 1,
		"2251799813711968": 1,
	}, gotUpdates)
	require.Equal(t, 2, gotConfirmations)
	envelope := requireUpdateProcessInstanceEnvelope(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	requireUpdateResultKeys(t, envelope, "2251799813711967", "2251799813711968")
}

func TestUpdateProcessInstanceCommand_StdinKeysMergeAndDeduplicateWithFlagKeys(t *testing.T) {
	var mu sync.Mutex
	updates := map[string]int{}
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables",
			"/v2/element-instances/2251799813711968/variables":
			key := keyFromElementInstanceVariablesPath(t, r.URL.Path)
			mu.Lock()
			updates[key]++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"},{"name":"foo","value":"\"bar\"","variableKey":"902","processInstanceKey":"2251799813711968","scopeKey":"2251799813711968","tenantId":"<default>"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout := executeRootForProcessInstanceTestWithStdin(t,
		"2251799813711967\n2251799813711968\n2251799813711968\n",
		"--config", cfgPath,
		"update", "pi",
		"--key", "2251799813711967",
		"-",
		"--vars", `{"foo":"bar"}`,
	)

	mu.Lock()
	gotUpdates := map[string]int{}
	for key, count := range updates {
		gotUpdates[key] = count
	}
	mu.Unlock()
	require.Equal(t, map[string]int{
		"2251799813711967": 1,
		"2251799813711968": 1,
	}, gotUpdates)
	require.Contains(t, stdout, "updated process-instance 2251799813711967: confirmed")
	require.Contains(t, stdout, "updated process-instance 2251799813711968: confirmed")
	require.Contains(t, stdout, "updated: 2")
}

func TestUpdateProcessInstanceCommand_NoWaitReturnsSubmittedWithoutConfirmationLookup(t *testing.T) {
	var sawUpdate bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables":
			require.Equal(t, http.MethodPut, r.Method)
			sawUpdate = true
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"foo": "bar"}, body["variables"])
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			t.Fatalf("no-wait update must not perform confirmation lookup")
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"update", "pi",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
		"--no-wait",
	)

	require.True(t, sawUpdate)
	require.Contains(t, output, "updated process-instance 2251799813711967: submitted")
	require.Contains(t, output, "updated: 1 (confirmed/submitted: 1, failed: 0)")
}

func TestUpdateProcessInstanceCommand_NoWaitJSONReportsSubmittedResults(t *testing.T) {
	var sawUpdate bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables":
			require.Equal(t, http.MethodPut, r.Method)
			sawUpdate = true
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			t.Fatalf("no-wait update must not perform confirmation lookup")
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"update", "pi",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
		"--no-wait",
	)

	require.True(t, sawUpdate)
	envelope := requireUpdateProcessInstanceEnvelope(t, stdout)
	require.Equal(t, string(OutcomeAccepted), envelope["outcome"])
	require.Equal(t, "update process-instance", envelope["command"])
	item := firstUpdateResultItem(t, envelope)
	require.Equal(t, "2251799813711967", item["key"])
	require.Equal(t, "submitted", item["status"])
	require.Equal(t, true, item["mutationAccepted"])
	require.Equal(t, "skipped", item["confirmationStatus"])
}

func TestUpdateProcessInstanceCommand_FullNameAndAliasBehaveIdenticallyForSingleKey(t *testing.T) {
	for _, leaf := range []string{"process-instance", "pi"} {
		t.Run(leaf, func(t *testing.T) {
			var requestedPath string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/element-instances/2251799813711967/variables":
					require.Equal(t, http.MethodPut, r.Method)
					requestedPath = r.URL.Path
					w.WriteHeader(http.StatusNoContent)
				case "/v2/variables/search":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"update", leaf,
				"--key", "2251799813711967",
				"--vars", `{"foo":"bar"}`,
			)

			require.Equal(t, "/v2/element-instances/2251799813711967/variables", requestedPath)
			require.Contains(t, output, "updated process-instance 2251799813711967: confirmed")
			require.Contains(t, output, "updated: 1")
		})
	}
}

func TestUpdateProcessInstanceCommand_InvalidVarsFailBeforeMutation(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	tests := []struct {
		name       string
		helperName string
		args       []string
		want       string
	}{
		{
			name:       "missing vars",
			helperName: "TestUpdateProcessInstanceCommand_MissingVarsHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967"},
			want:       "--vars is required and must be a JSON object",
		},
		{
			name:       "malformed json",
			helperName: "TestUpdateProcessInstanceCommand_MalformedVarsHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967", "--vars", `{"foo":`},
			want:       "--vars must be a valid JSON object",
		},
		{
			name:       "non object json",
			helperName: "TestUpdateProcessInstanceCommand_NonObjectVarsHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967", "--vars", `["foo"]`},
			want:       "--vars must be a valid JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, tt.helperName, map[string]string{
				"C8VOLT_TEST_CONFIG":      cfgPath,
				"C8VOLT_TEST_UPDATE_ARGS": marshalUpdateArgsForEnv(t, tt.args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "invalid input")
			require.Contains(t, string(output), tt.want)
		})
	}
}

func TestUpdateProcessInstanceCommand_MissingTargetsFailBeforeMutation(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	t.Run("no key", func(t *testing.T) {
		output, err := testx.RunCmdSubprocess(t, "TestUpdateProcessInstanceCommand_MissingKeyHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":      cfgPath,
			"C8VOLT_TEST_UPDATE_ARGS": marshalUpdateArgsForEnv(t, []string{"update", "pi", "--vars", `{"foo":"bar"}`}),
		})
		require.Error(t, err)

		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok)
		require.Equal(t, exitcode.Error, exitErr.ExitCode())
		require.Contains(t, string(output), "local precondition failed")
		require.Contains(t, string(output), "no process instance keys provided or found to update")
	})

	t.Run("empty stdin", func(t *testing.T) {
		output, err := testx.RunCmdSubprocessWithStdin(t, "TestUpdateProcessInstanceCommand_EmptyStdinHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":      cfgPath,
			"C8VOLT_TEST_UPDATE_ARGS": marshalUpdateArgsForEnv(t, []string{"update", "pi", "-", "--vars", `{"foo":"bar"}`}),
		}, "\n")
		require.Error(t, err)

		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok)
		require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
		require.Contains(t, string(output), "invalid input")
		require.Contains(t, string(output), "stdin contained no keys")
	})
}

func requireUpdateProcessInstanceEnvelope(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	return envelope
}

func firstUpdateResultItem(t *testing.T, envelope map[string]any) map[string]any {
	t.Helper()

	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	return item
}

func requireUpdateResultKeys(t *testing.T, envelope map[string]any, want ...string) {
	t.Helper()

	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, len(want))
	got := make([]string, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		require.True(t, ok)
		got = append(got, item["key"].(string))
		require.Equal(t, "confirmed", item["status"])
		require.Equal(t, true, item["mutationAccepted"])
		require.Equal(t, "confirmed", item["confirmationStatus"])
	}
	sort.Strings(got)
	sort.Strings(want)
	require.Equal(t, want, got)
}

func TestUpdateProcessInstanceCommand_MissingVarsHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_MalformedVarsHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_NonObjectVarsHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_MissingKeyHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_EmptyStdinHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func runUpdateProcessInstanceHelperFromEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var updateArgs []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_UPDATE_ARGS")), &updateArgs); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, updateArgs...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func marshalUpdateArgsForEnv(t *testing.T, args []string) string {
	t.Helper()
	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

func keyFromElementInstanceVariablesPath(t *testing.T, path string) string {
	t.Helper()

	const prefix = "/v2/element-instances/"
	const suffix = "/variables"
	require.True(t, len(path) > len(prefix)+len(suffix))
	require.Equal(t, prefix, path[:len(prefix)])
	require.Equal(t, suffix, path[len(path)-len(suffix):])
	return path[len(prefix) : len(path)-len(suffix)]
}

func executeRootForProcessInstanceTestWithStdin(t *testing.T, stdin string, args ...string) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	_, err = writer.WriteString(stdin)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	prevStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = prevStdin
		_ = reader.Close()
	})

	prevConfirm := confirmCmdOrAbortFn
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm
	t.Cleanup(resetProcessInstanceCommandGlobals)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm

	_, err = root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}
