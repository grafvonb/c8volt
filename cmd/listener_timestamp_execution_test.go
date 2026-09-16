// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestListenerTimestampCommandsHonorTimezoneConfig exercises config loading,
// transport conversion, facade mapping, and rendering in all four commands.
func TestListenerTimestampCommandsHonorTimezoneConfig(t *testing.T) {
	const key = "2251799813685249"
	for _, command := range []struct {
		name string
		args []string
	}{
		{"element", []string{"get", "element", "--pi-key", key, "--with-listeners"}},
		{"process", []string{"get", "process-instance", "--key", key, "--with-elements", "--with-listeners"}},
		{"walk", []string{"walk", "process-instance", "--key", key, "--with-elements", "--with-listeners"}},
		{"analysis", []string{"ops", "analyse", "slow-process-instances", "--key", key, "--with-listeners"}},
	} {
		for _, showOffset := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/offset=%t", command.name, showOffset), func(t *testing.T) {
				var requests testx.SafeSlice[string]
				process := `{"processInstanceKey":"2251799813685249","processDefinitionKey":"9001","processDefinitionId":"demo","processDefinitionVersion":1,"state":"ACTIVE","startDate":"2026-09-16T12:00:00Z","tenantId":"tenant","hasIncident":false}`
				srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Append(r.Method + " " + r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/v2/process-instances/" + key:
						_, _ = w.Write([]byte(process))
					case "/v2/process-instances/search":
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						filter, _ := body["filter"].(map[string]any)
						if _, children := filter["parentProcessInstanceKey"]; children {
							_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
						} else {
							_, _ = fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, process)
						}
					case "/v2/element-instances/search":
						_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-09-16T12:00:00Z","processInstanceKey":"2251799813685249","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
					case "/v2/jobs/search":
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						filter := requireJSONObject(t, body["filter"])
						if filter["kind"] == "EXECUTION_LISTENER" {
							_, _ = w.Write([]byte(`{"items":[{"jobKey":"job-offset","kind":"EXECUTION_LISTENER","listenerEventType":"START","type":"audit","state":"ACTIVATED","retries":1,"creationTime":"2026-09-16T13:07:16.359+05:30","endTime":"2026-09-16T13:07:16.842+05:30","deadline":"2026-09-16T13:08:00+05:30","processInstanceKey":"2251799813685249","elementInstanceKey":"element-1"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
						} else {
							_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
						}
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				t.Cleanup(srv.Close)
				cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.8")
				cfg, err := os.ReadFile(cfgPath)
				require.NoError(t, err)
				cfg = []byte(strings.Replace(string(cfg), "app:\n", fmt.Sprintf("app:\n  show_timezone_offset: %t\n", showOffset), 1))
				require.NoError(t, os.WriteFile(cfgPath, cfg, 0600))
				args := append([]string{"--config", cfgPath, "--no-indicator"}, command.args...)
				stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
				require.Empty(t, stderr)
				suffix := ""
				if showOffset {
					suffix = "+05:30"
				}
				found := false
				for _, line := range strings.Split(stdout, "\n") {
					if !strings.Contains(line, "job-offset ") {
						continue
					}
					found = true
					fields := strings.Fields(line)
					require.Contains(t, fields, "s:2026-09-16T13:07:16.359"+suffix)
					require.Contains(t, fields, "e:2026-09-16T13:07:16.842"+suffix)
					require.Contains(t, fields, "d:2026-09-16T13:08:00.000"+suffix)
				}
				require.True(t, found, "listener row missing: %s", stdout)
				jobs := 0
				for _, request := range requests.Snapshot() {
					if request == "POST /v2/jobs/search" {
						jobs++
					}
				}
				require.Equal(t, 2, jobs)
			})
		}
	}
}
