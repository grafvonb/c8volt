// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

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
