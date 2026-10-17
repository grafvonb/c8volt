// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/stretchr/testify/require"
)

// TestGetFamilySuspendedState proves reads, exact selectors, expectations and fresh
// timestamp snapshots on 8.10.4; the prefix includes it in integration-cli-get.
func TestGetFamilySuspendedState(t *testing.T) {
	writeSuspendedRevisionEvidence(t)
	for _, profile := range requireSelectedProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			// Older profiles keep their baseline get slice; this skip is never live acceptance.
			if profile.ExpectedVersion != "8.10" {
				t.Skip("suspension scenario requires a disposable 8.10.4 server")
			}
			fixture := newSuspendedFixture(t, profile)
			keys := fixture.dataset.PositiveProcessInstanceKeys
			suspended, active, terminal := keys[0], keys[1], keys[2]
			result := fixture.run(t, "terminal-control", "cancel", "pi", "--key", terminal, "--automation", "--json")
			require.NoError(t, result.Err, result.Stderr)
			require.NoError(t, requireProcessInstanceStateOneOf(t, fixture.profile, terminal, "CANCELED", "TERMINATED"))
			require.NoError(t, fixture.transition(t, suspended, true))
			observed, err := fixture.observe(t, suspended, "SUSPENDED", true)
			require.NoError(t, err)
			_, err = fixture.observe(t, active, "ACTIVE", false)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			report, instance, err := fixture.client.api.LookupProcessInstanceStateByKey(ctx, suspended)
			require.NoError(t, err)
			require.Equal(t, process.StateSuspended, report.State)
			require.Equal(t, process.StateSuspended, instance.State)
			require.False(t, instance.State.IsTerminal())
			require.Equal(t, observed.SuspendedDate.UTC().Format(time.RFC3339Nano), instance.SuspendedDate)
			writeDataEvidence(t, "suspended-public-lookup-"+profile.Name+".json", map[string]any{"report": report, "instance": instance})

			for _, tc := range []struct {
				state string
				keys  []string
			}{
				{"suspended", []string{suspended}}, {"ACTIVE", []string{active}}, {"all", keys},
			} {
				t.Run("filter-"+tc.state, func(t *testing.T) {
					result := fixture.run(t, "filter-"+tc.state, "get", "pi", "--pd-key", fixture.dataset.PositiveProcessDefinitionKeys[0], "--state", tc.state, "--json", "--automation")
					require.NoError(t, result.Err, result.Stderr)
					require.Empty(t, result.Stderr)
					var instances process.ProcessInstances
					decodeSuspendedLiveEnvelope(t, result.Stdout, "succeeded", &instances)
					actual := make([]string, 0, len(instances.Items))
					for _, item := range instances.Items {
						actual = append(actual, item.Key)
						if item.Key == suspended {
							require.Equal(t, process.StateSuspended, item.State)
							require.Equal(t, instance.SuspendedDate, item.SuspendedDate)
						}
					}
					require.ElementsMatch(t, tc.keys, actual)
				})
			}
			t.Run("state-expectation", func(t *testing.T) {
				result := fixture.run(t, "expect-state", "expect", "pi", "--key", suspended, "--state", "sUsPeNdEd", "--json")
				require.NoError(t, result.Err, result.Stderr)
				var reports process.StateReports
				decodeSuspendedLiveEnvelope(t, result.Stdout, "succeeded", &reports)
				require.Len(t, reports.Items, 1)
				require.Equal(t, process.StateSuspended, reports.Items[0].State)
			})
			t.Run("combined-expectation", func(t *testing.T) {
				result := fixture.run(t, "expect-combined", "expect", "pi", "--key", suspended, "--state", "suspended", "--incident", "false", "--json")
				require.NoError(t, result.Err, result.Stderr)
				var reports process.ProcessInstanceExpectationReports
				decodeSuspendedLiveEnvelope(t, result.Stdout, "succeeded", &reports)
				require.Len(t, reports.Items, 1)
				require.True(t, reports.Items[0].Ok)
				require.Equal(t, process.StateSuspended, reports.Items[0].State)
				require.NotNil(t, reports.Items[0].Incident)
				require.False(t, reports.Items[0].Incident.Bool())
			})
			t.Run("terminal-not-satisfied", func(t *testing.T) {
				result := fixture.run(t, "expect-terminal", "expect", "pi", "--key", suspended, "--state", "completed,canceled,terminated,absent", "--backoff-timeout", "1s", "--json")
				require.Error(t, result.Err)
				require.Equal(t, exitcode.Timeout, result.ExitCode)
				require.Less(t, result.FinishedAt.Sub(result.StartedAt), 15*time.Second)
				var envelope map[string]any
				require.NoError(t, json.Unmarshal([]byte(result.Stdout), &envelope))
				require.Equal(t, "failed", envelope["outcome"])
				require.Equal(t, "timeout", envelope["class"])
				// Confirm the failed expectation did not change or lose the held state.
				_, err := fixture.observe(t, suspended, "SUSPENDED", true)
				require.NoError(t, err)
			})

			requireSuspendedLiveSnapshot(t, fixture, suspended, process.StateSuspended, instance.SuspendedDate)
			require.NoError(t, fixture.transition(t, suspended, false))
			_, err = fixture.observe(t, suspended, "ACTIVE", false)
			require.NoError(t, err)
			fresh, err := fixture.client.api.GetProcessInstance(ctx, suspended)
			require.NoError(t, err)
			require.Equal(t, process.StateActive, fresh.State)
			require.Empty(t, fresh.SuspendedDate)
			writeDataEvidence(t, "suspended-public-resumed-"+profile.Name+".json", fresh)
			requireSuspendedLiveSnapshot(t, fixture, suspended, process.StateActive, "")
		})
	}
}

// decodeSuspendedLiveEnvelope requires one command result followed by EOF.
func decodeSuspendedLiveEnvelope(t *testing.T, output, outcome string, payload any) {
	t.Helper()
	var envelope struct {
		Outcome string          `json:"outcome"`
		Payload json.RawMessage `json:"payload"`
	}
	decoder := json.NewDecoder(strings.NewReader(output))
	require.NoError(t, decoder.Decode(&envelope))
	require.Equal(t, io.EOF, decoder.Decode(new(any)))
	require.Equal(t, outcome, envelope.Outcome)
	require.NoError(t, json.Unmarshal(envelope.Payload, payload))
}

// requireSuspendedLiveSnapshot checks lookup and search JSON fields rather than
// trusting a zero-valued Go field to prove omission after resumption.
func requireSuspendedLiveSnapshot(t *testing.T, fixture *suspendedFixture, key string, state process.State, date string) {
	t.Helper()
	for _, search := range []bool{false, true} {
		name := "snapshot-" + string(state) + "-lookup"
		args := []string{"get", "pi", "--key", key, "--json"}
		if search {
			name = "snapshot-" + string(state) + "-search"
			args = []string{"get", "pi", "--pd-key", fixture.dataset.PositiveProcessDefinitionKeys[0], "--state", "all", "--json", "--automation"}
		}
		result := fixture.run(t, name, args...)
		require.NoError(t, result.Err, result.Stderr)
		require.Empty(t, result.Stderr)
		var payload struct {
			Items []map[string]any `json:"items"`
		}
		decodeSuspendedLiveEnvelope(t, result.Stdout, "succeeded", &payload)
		found := false
		for _, item := range payload.Items {
			if item["key"] != key {
				continue
			}
			found = true
			require.Equal(t, string(state), item["state"])
			if date == "" {
				require.NotContains(t, item, "suspendedDate")
			} else {
				require.Equal(t, date, item["suspendedDate"])
			}
		}
		require.True(t, found, "owned instance missing from snapshot")
	}
}
