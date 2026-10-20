// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBulkCompletionAcceptanceListenerHierarchy represents the incident's three
// listener owners with a test-owned HTTP fake. It proves CLI scope and submission
// reporting, without deploying BPMN or claiming that a real hierarchy terminated.
func TestBulkCompletionAcceptanceListenerHierarchy(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		t.Run(version, func(t *testing.T) {
			for _, mode := range []string{"preview", "execute", "no wait"} {
				t.Run(mode, func(t *testing.T) {
					keys := []string{completionAcceptanceKey(10), completionAcceptanceKey(11), completionAcceptanceKey(12)}
					processKeys := map[string]string{}
					control := completionAcceptanceKey(13)
					// Return a control row despite the requested filter to exercise
					// defensive matching before any completion can be submitted.
					rows := []string{strings.ReplaceAll(bulkCompletionRow(control, "CREATED"), `"cleanup"`, `"other"`)}
					for i, key := range keys {
						processKeys[key] = completionAcceptanceKey(100 + i)
						rows = append(rows, strings.ReplaceAll(bulkCompletionRow(key, "CREATED"), `"2251799813711000"`, strconv.Quote(processKeys[key])))
					}
					f, server := newCompletionAcceptanceServer(t, completionAcceptancePages(rows, 2), nil)
					// These inputs are immutable before any command request starts.
					f.processInstanceKeys = processKeys
					f.extraFilters = map[string]string{"state": "CREATED", "kind": "TASK_LISTENER", "listenerEventType": "CANCELING"}
					flags := []string{"--type", "cleanup", "--state", "created", "--kind", "TASK_LISTENER", "--listener-event-type", "canceling", "--batch-size", "2", "--json"}
					if mode == "preview" {
						flags = append(flags, "--dry-run")
					} else {
						flags = append(flags, "--auto-confirm")
						if mode == "no wait" {
							flags = append(flags, "--no-wait")
						}
					}
					stdout, stderr, err := runCompletionAcceptance(t, server, version, flags, "", false)
					require.NoError(t, err, stdout, stderr)
					require.Empty(t, stderr)
					envelope := decodeBulkCompletionEnvelope(t, stdout)
					wantEnvelope := "succeeded"
					if mode == "no wait" {
						wantEnvelope = "accepted"
					}
					require.Equal(t, wantEnvelope, envelope["outcome"])
					payload := requireJSONObject(t, envelope["payload"])
					counts := requireJSONObject(t, payload["counts"])
					require.Equal(t, float64(3), counts["selected"])
					require.Zero(t, counts["completed"], "accepted responses do not prove observed completion")
					if mode == "preview" {
						require.Equal(t, "preview", payload["status"])
						require.NotContains(t, payload, "outcomes")
						candidates := payload["candidates"].([]any)
						require.Len(t, candidates, len(keys))
						for i, raw := range candidates {
							candidate := requireJSONObject(t, raw)
							require.Equal(t, keys[i], candidate["key"])
							require.Equal(t, processKeys[keys[i]], candidate["processInstanceKey"])
						}
						require.Zero(t, counts["accepted"])
					} else {
						require.Equal(t, "finished", payload["status"])
						outcomes := requireCompletionAcceptancePartition(t, payload, keys)
						require.Equal(t, float64(3), counts["accepted"])
						for i, raw := range outcomes {
							out := requireJSONObject(t, raw)
							require.Equal(t, processKeys[keys[i]], requireJSONObject(t, out["candidate"])["processInstanceKey"])
							require.Equal(t, "accepted", out["disposition"])
							require.Equal(t, "accepted", requireJSONObject(t, out["submission"])["status"])
							require.Equal(t, "skipped", requireJSONObject(t, out["observation"])["status"])
						}
					}
					f.mu.Lock()
					defer f.mu.Unlock()
					require.Equal(t, 2, f.discovery)
					require.Equal(t, []string{"search:0", "search:1"}, f.calls[:2])
					require.Zero(t, f.reads[control])
					require.Zero(t, f.writes[control])
					require.NotContains(t, stdout, control)
					if mode == "preview" {
						require.Empty(t, f.reads)
						require.Empty(t, f.writes)
						require.Len(t, f.calls, 2)
					} else {
						require.Len(t, f.reads, 3)
						require.Len(t, f.writes, 3)
						require.Len(t, f.calls, 8)
						for _, key := range keys {
							require.Equal(t, 1, f.reads[key])
							require.Equal(t, 1, f.writes[key])
						}
					}
				})
			}
		})
	}
}
