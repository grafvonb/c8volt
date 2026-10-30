// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// A null batch must reach normal definition confirmation; failures at either
// boundary retain a nonzero exit and exactly one machine-readable envelope.
func TestDeleteProcessDefinitionDeferredHistory(t *testing.T) {
	for _, tc := range []struct {
		name         string
		noWait       bool
		body         string
		lookupStatus int
		wantOutcome  string
	}{
		{name: "confirmed", body: `{"resourceKey":"123","batchOperation":null}`, lookupStatus: 404, wantOutcome: "succeeded"},
		{name: "accepted", noWait: true, body: `{"resourceKey":"123","batchOperation":null}`, wantOutcome: "accepted"},
		{name: "confirmation forbidden", body: `{"resourceKey":"123","batchOperation":null}`, lookupStatus: 403, wantOutcome: "failed"},
		{name: "malformed delete", body: `{}`, wantOutcome: "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var deletes, lookups atomic.Int32
			srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}

				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/batch-operations/search":
					_, _ = io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
				case "/v2/resources/123/deletion":
					deletes.Add(1)
					require.Equal(t, http.MethodPost, r.Method)
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, true, body["deleteHistory"])
					_, _ = io.WriteString(w, tc.body)
				case "/v2/process-definitions/123":
					lookups.Add(1)
					require.Equal(t, http.MethodGet, r.Method)
					require.NotZero(t, tc.lookupStatus)
					w.WriteHeader(tc.lookupStatus)
					_, _ = io.WriteString(w, `{"detail":"definition lookup rejected"}`)
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(srv.Close)
			args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "--json", "delete", "process-definition", "--key", "123", "--auto-confirm", "--no-state-check"}
			if tc.noWait {
				args = append(args, "--no-wait")
			}
			raw, err := json.Marshal(args)
			require.NoError(t, err)
			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestHelperDeleteProcessDefinitionDeferredHistory", map[string]string{"C8VOLT_TEST_DELETE_ARGS": string(raw)})
			if tc.wantOutcome == "failed" {
				require.Error(t, err, stdout+stderr)
			} else {
				require.NoError(t, err, stdout+stderr)
			}
			decoder := json.NewDecoder(strings.NewReader(stdout))
			var envelope map[string]any
			require.NoError(t, decoder.Decode(&envelope))
			require.Equal(t, tc.wantOutcome, envelope["outcome"])
			require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
			require.NotContains(t, stdout, "INFO")
			require.NotContains(t, stdout, "ERROR")
			require.Equal(t, int32(1), deletes.Load())
			wantLookups := int32(0)
			if tc.lookupStatus != 0 {
				wantLookups = 1
			}
			require.Equal(t, wantLookups, lookups.Load())
			if tc.wantOutcome == "failed" {
				require.NotNil(t, envelope["detail"])
			}
		})
	}
}

func TestHelperDeleteProcessDefinitionDeferredHistory(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_DELETE_ARGS")), &args))
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs(args)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
