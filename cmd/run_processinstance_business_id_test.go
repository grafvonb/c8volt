// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// runBusinessIDFixture owns every request and records submitted inputs and reads.
type runBusinessIDFixture struct {
	inputs   testx.SafeSlice[map[string]any]
	reads    atomic.Int32
	searches atomic.Int32
	writes   atomic.Int32
}

// newRunBusinessIDFixture supplies independent native/observed identities; negative successes means all succeed.
func newRunBusinessIDFixture(t *testing.T, version string, successes int, observed *string) (*runBusinessIDFixture, *httptest.Server) {
	t.Helper()
	f := &runBusinessIDFixture{}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			f.searches.Add(1)
			fmt.Fprint(w, `{"items":[{"processDefinitionId":"order","processDefinitionKey":"9001","tenantId":"<default>","version":1},{"processDefinitionId":"other","processDefinitionKey":"9002","tenantId":"<default>","version":1}],"page":{"totalItems":2}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			f.inputs.Append(body)
			n := int(f.writes.Add(1))
			if successes >= 0 && n > successes {
				w.WriteHeader(http.StatusConflict)
				fmt.Fprint(w, `{"title":"Conflict","detail":"POST https://private.invalid/secret body-only-secret resource-key-secret leaseToken"}`)
				return
			}
			fmt.Fprintf(w, `{"processInstanceKey":%q,"processDefinitionKey":"9001","processDefinitionId":"order","processDefinitionVersion":1,"tenantId":"<default>","businessId":"native,ID"}`, strconv.Itoa(100+n))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			f.reads.Add(1)
			result := map[string]any{"processInstanceKey": strings.TrimPrefix(r.URL.Path, "/v2/process-instances/"), "processDefinitionKey": "9001", "processDefinitionId": "order", "processDefinitionVersion": 1, "tenantId": "<default>", "state": "ACTIVE"}
			if observed != nil {
				result["businessId"] = *observed
			}
			require.NoError(t, json.NewEncoder(w).Encode(result))
		default:
			t.Errorf("unexpected fixture request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(server.Close)
	return f, server
}

// runBusinessIDCommand checks terminal stdin, redirected stdout and both logger destinations.
func runBusinessIDCommand(t *testing.T, args []string, destination string, failWriter bool, exchanges ...testx.CmdTerminalExchange) (string, string, error) {
	t.Helper()
	dir := t.TempDir()
	env := map[string]string{"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, args), "C8VOLT_BID_DEST": destination, "C8VOLT_BID_ERR": filepath.Join(dir, "stderr"), "C8VOLT_BID_OUT": filepath.Join(dir, "stdout")}
	if failWriter {
		env["C8VOLT_BID_FAIL_WRITER"] = "1"
	}
	result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestRunProcessInstanceBusinessIDHelper", Env: env, Exchanges: exchanges})
	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	require.Empty(t, result.Stdout, "result bypassed configured output")
	captured, err := os.ReadFile(env["C8VOLT_BID_ERR"])
	require.NoError(t, err)
	require.Equal(t, string(captured), result.Stderr, "diagnostic bypassed configured logger destination")
	output, err := os.ReadFile(env["C8VOLT_BID_OUT"])
	require.NoError(t, err)
	return string(output), result.Stderr, result.Err
}

// runBusinessIDFailWriter records attempts to detect duplicate envelope writes.
type runBusinessIDFailWriter struct{ io.Writer }

// Write records an attempt without pretending any result bytes were delivered.
func (w runBusinessIDFailWriter) Write(p []byte) (int, error) {
	_, _ = io.WriteString(w.Writer, "attempt\n")
	return 0, io.ErrClosedPipe
}

// TestRunProcessInstanceBusinessIDHelper isolates globals and exits with real terminal eligibility.
func TestRunProcessInstanceBusinessIDHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	require.False(t, term.IsTerminal(int(os.Stdout.Fd())))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_ROOT_ARGS")), &args))
	root := Root()
	out, err := os.Create(os.Getenv("C8VOLT_BID_OUT"))
	require.NoError(t, err)
	capture, err := os.Create(os.Getenv("C8VOLT_BID_ERR"))
	require.NoError(t, err)
	root.SetOut(out)
	if os.Getenv("C8VOLT_BID_FAIL_WRITER") == "1" {
		root.SetOut(runBusinessIDFailWriter{out})
	}
	writer := io.MultiWriter(os.Stderr, capture)
	if os.Getenv("C8VOLT_BID_DEST") == "configured" {
		selected, _, findErr := root.Find(args)
		require.NoError(t, findErr)
		selected.SetErr(writer)
	} else {
		root.SetErr(writer)
	}
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	os.Exit(0)
}

// assertRunBusinessIDExit checks original failure exits and no-error-codes overrides.
func assertRunBusinessIDExit(t *testing.T, err error, code int) {
	t.Helper()
	if code == 0 {
		require.NoError(t, err)
		return
	}
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, code, exitErr.ExitCode())
}

// TestRunProcessInstanceBusinessIDInputs covers presence, selectors, repeated inputs and observation source.
func TestRunProcessInstanceBusinessIDInputs(t *testing.T) {
	observed := "observed\n\"ID"
	for _, tc := range []struct {
		name, version, command string
		selector, flags        []string
		bid                    *string
		observed               *string
		want                   string
		count, reads, exit     int
	}{
		{name: "key native opaque", version: "8.10", command: "pi", selector: []string{"--pd-key", "9001"}, flags: []string{"--no-wait"}, bid: toolx.Ptr("  order,AbC  "), want: "native,ID", count: 1},
		{name: "BPMN observed", version: "8.10", command: "process-instance", selector: []string{"-b", "order"}, bid: toolx.Ptr(" "), observed: &observed, want: observed, count: 1, reads: 1},
		{name: "absent observation", version: "8.10", command: "pi", selector: []string{"--pd-key", "9001"}, bid: toolx.Ptr("input"), count: 1, reads: 1},
		{name: "multiple keys", version: "8.10", command: "pi", selector: []string{"--pd-key", "9001,9002"}, flags: []string{"--no-wait"}, bid: toolx.Ptr("same,ID"), want: "native,ID", count: 2},
		{name: "multiple BPMN", version: "8.10", command: "pi", selector: []string{"-b", "order,other"}, flags: []string{"--no-wait"}, bid: toolx.Ptr("same,ID"), want: "native,ID", count: 2},
		{name: "count", version: "8.10", command: "pi", selector: []string{"--pd-key", "9001"}, flags: []string{"--no-wait", "--count", "3", "--workers", "2"}, bid: toolx.Ptr("same,ID"), want: "native,ID", count: 3},
		{name: "omitted", version: "8.10", command: "pi", selector: []string{"--pd-key", "9001"}, flags: []string{"--no-wait"}, want: "native,ID", count: 1},
		{name: "empty", version: "8.10", command: "pi", selector: []string{"--pd-key", "9001"}, bid: toolx.Ptr(""), exit: 2},
		{name: "empty count", version: "8.10", command: "pi", selector: []string{"--pd-key", "9001"}, flags: []string{"--count", "4", "--workers", "2"}, bid: toolx.Ptr(""), exit: 2},
		{name: "UTF-16 boundary", version: "8.10", command: "pi", selector: []string{"-b", "order"}, flags: []string{"--no-wait"}, bid: toolx.Ptr(strings.Repeat("😀", 128)), want: "native,ID", count: 1},
		{name: "too long", version: "8.10", command: "pi", selector: []string{"-b", "order"}, bid: toolx.Ptr(strings.Repeat("😀", 129)), exit: 2},
		{name: "v88 unsupported empty", version: "8.8", command: "pi", selector: []string{"--pd-key", "9001"}, bid: toolx.Ptr(""), exit: 1},
		{name: "v89 unsupported", version: "8.9", command: "pi", selector: []string{"-b", "order"}, bid: toolx.Ptr("id"), exit: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, srv := newRunBusinessIDFixture(t, tc.version, -1, tc.observed)
			args := []string{"--config", writeTestConfigForVersion(t, srv.URL, tc.version), "--json", "--quiet", "run", tc.command}
			args = append(args, tc.selector...)
			args = append(args, tc.flags...)
			if tc.bid != nil {
				args = append(args, "--bid", *tc.bid)
			}
			stdout, stderr, err := runBusinessIDCommand(t, args, "configured", false)
			assertRunBusinessIDExit(t, err, tc.exit)
			require.Empty(t, stderr)
			envelope := decodeRunBusinessIDEnvelope(t, stdout)
			require.Equal(t, int32(tc.count), f.writes.Load())
			require.Equal(t, int32(tc.reads), f.reads.Load())
			for _, input := range f.inputs.Snapshot() {
				if tc.bid == nil {
					require.NotContains(t, input, "businessId")
				} else {
					require.Equal(t, *tc.bid, input["businessId"])
				}
				require.Empty(t, input["variables"])
			}
			if tc.exit != 0 {
				class := "invalid_input"
				if tc.version != "8.10" {
					class = "unsupported"
				}
				require.Equal(t, class, envelope["class"])
				return
			}
			outcome := "succeeded"
			if tc.reads == 0 {
				outcome = "accepted"
			}
			require.Equal(t, outcome, envelope["outcome"])
			payload := requireJSONObject(t, envelope["payload"])
			require.Equal(t, float64(tc.count), payload["total"])
			items := payload["items"].([]any)
			require.Len(t, items, tc.count)
			for _, raw := range items {
				item := requireJSONObject(t, raw)
				if tc.want == "" {
					require.NotContains(t, item, "businessId")
				} else {
					require.Equal(t, tc.want, item["businessId"])
				}
			}
		})
	}
}

// decodeRunBusinessIDEnvelope requires exactly one creation envelope and EOF.
func decodeRunBusinessIDEnvelope(t *testing.T, stdout string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope), stdout)
	var extra any
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
	require.Equal(t, "run process-instance", envelope["command"])
	return envelope
}
