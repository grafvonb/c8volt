// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestProcessDefinitionLifecycleWatchTransitions compares independent observations,
// clears omissions and markers, and distinguishes returned DELETED from absence.
func TestProcessDefinitionLifecycleWatchTransitions(t *testing.T) {
	t.Parallel()
	var baseline processDefinitionWatchBaseline
	for _, step := range []struct {
		state            string
		active           int64
		absent           bool
		updated, removed int
		styled           string
	}{
		{state: "ACTIVE", styled: "ACTIVE"},
		{state: "DRAINING", updated: 1, styled: "\x1b[1mDRAINING\x1b[0m"},
		{state: "DRAINING", active: 2, updated: 1, styled: "DRAINING"},
		{state: "future-state", active: 3, updated: 1, styled: "\x1b[1mfuture-state\x1b[0m"},
		{active: 3, updated: 1},
		{state: "ACTIVE", active: 3, updated: 1, styled: "\x1b[1mACTIVE\x1b[0m"},
		{state: "DELETED", active: 3, updated: 1, styled: "\x1b[1mDELETED\x1b[0m"},
		{state: "DELETED", active: 3, styled: "DELETED"},
		{absent: true, removed: 1},
	} {
		items := []process.ProcessDefinition{{Key: "123", BpmnProcessId: "invoice", ProcessVersion: 1, State: step.state, Statistics: &process.ProcessDefinitionStatistics{Active: step.active, Suspended: 2, SuspendedCountSupported: true}}}
		if step.absent {
			items = nil
		}
		candidate := baseline.candidate(items)
		require.Equal(t, step.updated, candidate.updated)
		require.Equal(t, step.removed, candidate.removed)
		rows := formatProcessDefinitionWatchRows(candidate.rows, true)
		if !step.absent {
			require.Len(t, rows, 1)
			require.Contains(t, rows[0], step.styled)
			require.NotContains(t, rows[0], "\x1b[1m\x1b[0m")
			if step.state == "" {
				require.NotContains(t, rows[0], "future-state")
			}
			if baseline.initialized {
				changedState := baseline.byKey["123"].state != step.state
				require.Equal(t, changedState, candidate.rows[0].changed.state)
				require.Equal(t, baseline.byKey["123"].active != newProcessDefinitionDisplay(items[0]).active, candidate.rows[0].changed.active)
			}
		}
		baseline.commit(candidate)
	}
}

// TestGetProcessDefinitionLifecycleWatchConfiguredLogging executes the real
// command against independent state/count fakes with configured logger routing.
func TestGetProcessDefinitionLifecycleWatchConfiguredLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, level := range []string{"info", "warn"} {
			for _, route := range []string{"configured", "inherited"} {
				for _, verbose := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/verbose=%v", format, level, route, verbose), func(t *testing.T) {
						resetGetProcessDefinitionCommandGlobals()
						t.Cleanup(resetGetProcessDefinitionCommandGlobals)
						fixture := &definitionStatisticsFixture{version: "8.10", lifecycle: func(read int32, _ string) string { return []string{"ACTIVE", "DRAINING", "DRAINING"}[read-1] }, population: func(int32) (int, int) { return 0, 2 }}
						srv := fixture.server(t)
						cfg := writeRawTestConfig(t, fmt.Sprintf("app:\n  camunda_version: \"8.10\"\nauth:\n  mode: none\napis:\n  camunda_api:\n    base_url: %s\nlog:\n  level: %s\n  format: %s\n", srv.URL, level, format))
						root := Root()
						resetCommandTreeFlags(root)
						var stdout, stderr, diagnostics bytes.Buffer
						root.SetOut(&stdout)
						root.SetErr(&stderr)
						command, _, err := root.Find([]string{"get", "pd"})
						require.NoError(t, err)
						command.SetErr(nil)
						if route == "configured" {
							command.SetErr(&diagnostics)
						} else {
							root.SetErr(&diagnostics)
						}
						t.Cleanup(func() { command.SetErr(nil) })
						oldSleep, oldNow := processDefinitionWatchSleep, processDefinitionWatchNow
						t.Cleanup(func() { processDefinitionWatchSleep, processDefinitionWatchNow = oldSleep, oldNow })
						sleeps := 0
						processDefinitionWatchSleep = func(context.Context, time.Duration) error {
							sleeps++
							if sleeps == 3 {
								return context.Canceled
							}
							return nil
						}
						processDefinitionWatchNow = newProcessDefinitionWatchClockForTest(time.Now(), 0, 2*time.Second, 3*time.Second, 5*time.Second, 6*time.Second, 6*time.Second)
						args := []string{"--config", cfg, "get", "pd", "--watch", "--stat"}
						if verbose {
							args = append(args, "--verbose")
						}
						root.SetArgs(args)
						require.NoError(t, root.Execute())
						require.EqualValues(t, 3, fixture.definitions.Load())
						require.EqualValues(t, 18, fixture.counts.Load())
						require.Empty(t, stderr.String())
						frames := strings.Split(stdout.String(), processDefinitionWatchRepaintControlSequenceForTest)[1:]
						require.Equal(t, []string{"123 tenant invoice v1 ACTIVE [ac:- cp:- cx:- inc:- su:2]\nfound: 1\n", "123 tenant invoice v1 DRAINING [ac:- cp:- cx:- inc:- su:2] *\nfound: 1\n", "123 tenant invoice v1 DRAINING [ac:- cp:- cx:- inc:- su:2]\nfound: 1\n"}, frames)
						records := diagnostics.String()
						require.Equal(t, 1, strings.Count(records, "slow process-definition watch refresh"))
						summaries, timings := 3, 0
						if verbose {
							timings = 3
						}
						if level == "warn" {
							summaries, timings = 0, 0
						}
						require.Equal(t, summaries, strings.Count(records, " new, "))
						require.Equal(t, timings, strings.Count(records, " completed in "))
						if format == "json" {
							decoder := json.NewDecoder(strings.NewReader(records))
							for {
								var record map[string]any
								err := decoder.Decode(&record)
								if err == io.EOF {
									break
								}
								require.NoError(t, err)
								if strings.HasPrefix(record["msg"].(string), "slow process-definition") {
									require.Equal(t, "WARN", record["level"])
								} else {
									require.Equal(t, "INFO", record["level"])
								}
							}
						} else {
							for _, line := range strings.Split(strings.TrimSuffix(records, "\n"), "\n") {
								require.Regexp(t, `^\d{2}:\d{2}:\d{2}\.\d{3} (INFO|WARN) `, line)
							}
							require.Contains(t, records, "WARN slow process-definition")
						}
					})
				}
			}
		}
	}
}

// TestProcessDefinitionLifecycleWatchSummaryContext preserves configured
// attributes, rejects missing context, and treats filtered INFO as success.
func TestProcessDefinitionLifecycleWatchSummaryContext(t *testing.T) {
	var stdout, diagnostics bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&stdout)
	command.SetErr(&bytes.Buffer{})
	baseline := processDefinitionWatchBaseline{}
	snapshot := process.ProcessDefinitionWatchSnapshot{Items: []process.ProcessDefinition{{Key: "123", State: "ACTIVE"}}}
	require.ErrorIs(t, renderProcessDefinitionWatchRefresh(command, &baseline, snapshot), logging.ErrNoLoggerInContext)
	require.False(t, baseline.initialized)
	command.SetContext(logging.ToContext(context.Background(), slog.New(slog.NewJSONHandler(&diagnostics, nil)).With("scope", "definition-watch")))
	require.NoError(t, renderProcessDefinitionWatchRefresh(command, &baseline, snapshot))
	require.Contains(t, diagnostics.String(), `"scope":"definition-watch"`)
	require.True(t, baseline.initialized)
	snapshot.Items[0].State = "DRAINING"
	command.SetContext(logging.ToContext(context.Background(), slog.New(slog.NewJSONHandler(&processDefinitionWatchFailWriter{}, nil))))
	require.ErrorContains(t, renderProcessDefinitionWatchRefresh(command, &baseline, snapshot), "injected watch output failure")
	require.Equal(t, "ACTIVE", baseline.byKey["123"].state)
	command.SetContext(logging.ToContext(context.Background(), slog.New(slog.NewJSONHandler(&processDefinitionWatchFailWriter{}, &slog.HandlerOptions{Level: slog.LevelWarn}))))
	require.NoError(t, renderProcessDefinitionWatchRefresh(command, &baseline, snapshot))
	require.Equal(t, "DRAINING", baseline.byKey["123"].state)
}

// TestGetProcessDefinitionLifecycleWatchOmissionAfterFailure ensures a failed
// partial response cannot supply the comparison baseline for a later omission.
func TestGetProcessDefinitionLifecycleWatchOmissionAfterFailure(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	calls := 0
	cli := processDefinitionWatchTestAPI{collect: func(context.Context, process.ProcessDefinitionWatchSnapshotRequest, ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
		calls++
		item := process.ProcessDefinition{Key: "123", BpmnProcessId: "invoice", ProcessVersion: 1}
		if calls == 1 {
			item.State = "ACTIVE"
		}
		snapshot := process.ProcessDefinitionWatchSnapshot{Items: []process.ProcessDefinition{item}, Total: 1}
		if calls == 2 {
			return snapshot, context.DeadlineExceeded
		}
		return snapshot, nil
	}}
	result := executeGetProcessDefinitionWatchHarnessForTest(t, processDefinitionWatchHarness{cli: cli, maxRetries: 1, sleep: func(context.Context, time.Duration) error {
		if calls == 4 {
			return context.Canceled
		}
		return nil
	}})
	require.NoError(t, result.err)
	require.Equal(t, 4, calls)
	frames := strings.Split(result.stdout, processDefinitionWatchRepaintControlSequenceForTest)[1:]
	require.Equal(t, []string{"123 invoice v1 ACTIVE\nfound: 1\n", "123 invoice v1 *\nfound: 1\n", "123 invoice v1\nfound: 1\n"}, frames)
	require.Equal(t, 1, strings.Count(result.stderr, "1 updated, 0 new, 0 removed"))
}
