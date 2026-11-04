// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestJobTimeoutUpdatedSelection exercises the CLI parser, adapter and shared
// coordinator against test-owned HTTP fakes, with the actual 8.10 job state.
func TestJobTimeoutUpdatedSelection(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(fmt.Sprint("complete=", complete), func(t *testing.T) {
			searches, writes := 0, 0
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}
				require.Equal(t, http.MethodPost, r.Method)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/jobs/search":
					searches++
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					filter := requireJSONObject(t, body["filter"])
					if filter["jobKey"] == nil {
						require.Equal(t, "TIMEOUT_UPDATED", filter["state"])
					} else {
						require.Equal(t, "2251799813711967", filter["jobKey"])
					}
					io.WriteString(w, `{"items":[{"jobKey":"2251799813711967","state":"TIMEOUT_UPDATED","retries":3,"type":"work","kind":"BPMN_ELEMENT","processInstanceKey":"2251799813711000","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
				case "/v2/jobs/2251799813711967/completion":
					writes++
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, "ORDER-42", body["businessId"])
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected fake request %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			t.Cleanup(server.Close)
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.10"), "--json", "--tenant", "tenant-a"}
			if complete {
				args = append(args, "--automation", "update", "job", "--state", "timeout_updated", "--type", "work", "--complete", "--set-bid", "ORDER-42")
			} else {
				args = append(args, "get", "job", "--state", "timeout_updated", "--limit", "1")
			}
			stdout, stderr := executeRootForJobTestWithSeparateOutputs(t, args...)
			require.Empty(t, stderr)
			var envelope struct {
				Outcome string `json:"outcome"`
				Payload struct {
					Counts struct {
						Accepted int `json:"accepted"`
					} `json:"counts"`
				} `json:"payload"`
			}
			require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
			if complete {
				require.Equal(t, "succeeded", envelope.Outcome)
				require.Equal(t, 1, envelope.Payload.Counts.Accepted)
				require.Equal(t, 1, writes)
				require.Equal(t, 2, searches, "frozen discovery and execution recheck")
			} else {
				require.Equal(t, "succeeded", envelope.Outcome)
				require.Equal(t, 0, writes)
				require.Equal(t, 1, searches)
			}
		})
	}
}
