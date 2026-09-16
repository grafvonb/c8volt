// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsAnalyseSlowProcessInstancesBuildsExplicitKeyRequests verifies flags and stdin normalize to one keyed request.
func TestOpsAnalyseSlowProcessInstancesBuildsExplicitKeyRequests(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		keys      []string
		stdinKeys typex.Keys
		wantKeys  typex.Keys
	}{
		{name: "repeated --key", keys: []string{"2251799813685249", "2251799813685250"}, wantKeys: typex.Keys{"2251799813685249", "2251799813685250"}},
		{name: "stdin dash", args: []string{"-"}, stdinKeys: typex.Keys{"2251799813685251"}, wantKeys: typex.Keys{"2251799813685251"}},
		{name: "mixed flag and stdin keys", args: []string{"-"}, keys: []string{"2251799813685249"}, stdinKeys: typex.Keys{"2251799813685250"}, wantKeys: typex.Keys{"2251799813685249", "2251799813685250"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			flagOpsAnalyseSlowProcessInstanceKeys = append([]string(nil), tc.keys...)
			keys := append(typex.Keys{}, flagOpsAnalyseSlowProcessInstanceKeys...)
			keys = append(keys, tc.stdinKeys...)

			got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, tc.args, keys.Unique())

			require.NoError(t, err)
			require.Equal(t, ops.SlowProcessAnalysisSelectionModeExplicitKeys, got.Request.SelectionMode)
			require.Equal(t, tc.wantKeys, got.Request.InputKeys)
			require.Equal(t, len(tc.args) == 1 && tc.args[0] == "-", got.StdinRequested)
			require.NotZero(t, got.Request.CapturedNow)
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesKeyFlagHasShortAlias protects the documented -k shorthand.
func TestOpsAnalyseSlowProcessInstancesKeyFlagHasShortAlias(t *testing.T) {
	require.NotNil(t, opsAnalyseSlowProcessInstancesCmd.Flags().ShorthandLookup("k"))
	require.Equal(t, "key", opsAnalyseSlowProcessInstancesCmd.Flags().ShorthandLookup("k").Name)
}

// TestOpsAnalyseSlowProcessInstancesWithFullTimelineFlagIsCommandLocal verifies the full detail switch stays out of facade input.
func TestOpsAnalyseSlowProcessInstancesWithFullTimelineFlagIsCommandLocal(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
		flagOpsAnalyseSlowProcessInstanceWithFullTimeline = false
	})

	flag := opsAnalyseSlowProcessInstancesCmd.Flags().Lookup("with-full-timeline")
	require.NotNil(t, flag)
	require.Equal(t, "false", flag.DefValue)
	require.Empty(t, flag.Shorthand)
	require.Contains(t, flag.Usage, "complete chronological element and transition detail")

	aliasCmd, remaining, err := root.Find([]string{"ops", "analyze", "spi", "--with-full-timeline"})
	require.NoError(t, err)
	require.Equal(t, []string{"--with-full-timeline"}, remaining)
	require.Same(t, opsAnalyseSlowProcessInstancesCmd, aliasCmd)
	require.NoError(t, aliasCmd.Flags().Set("with-full-timeline", "true"))

	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
	flagOpsAnalyseSlowProcessInstanceWithFullTimeline = true

	got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

	require.NoError(t, err)
	require.True(t, got.WithFullTimeline)
	require.Equal(t, ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch, got.Request.SelectionMode)
	require.Equal(t, "one-line", got.Request.OutputMode)
}

// TestOpsAnalyseSlowProcessInstancesWithFullTimelineAllowsMachineModes verifies parse output mode remains independent.
func TestOpsAnalyseSlowProcessInstancesWithFullTimelineAllowsMachineModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
		json bool
		keys bool
	}{
		{name: "json", mode: "json", json: true},
		{name: "keys-only", mode: "keys-only", keys: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			prevJSON := flagViewAsJson
			prevKeysOnly := flagViewKeysOnly
			t.Cleanup(func() {
				flagViewAsJson = prevJSON
				flagViewKeysOnly = prevKeysOnly
			})
			flagViewAsJson = tc.json
			flagViewKeysOnly = tc.keys
			flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"

			withoutFullTimeline, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)
			require.NoError(t, err)
			flagOpsAnalyseSlowProcessInstanceWithFullTimeline = true
			withFullTimeline, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

			require.NoError(t, err)
			require.False(t, withoutFullTimeline.WithFullTimeline)
			require.True(t, withFullTimeline.WithFullTimeline)
			require.Equal(t, tc.mode, withFullTimeline.Request.OutputMode)
			require.Equal(t, withoutFullTimeline.Request.SelectionMode, withFullTimeline.Request.SelectionMode)
			require.Equal(t, withoutFullTimeline.Request.ProcessDefinitionSelector, withFullTimeline.Request.ProcessDefinitionSelector)
			require.Equal(t, withoutFullTimeline.Request.ProcessInstanceFilters, withFullTimeline.Request.ProcessInstanceFilters)
			require.Equal(t, withoutFullTimeline.Request.DetailFilters, withFullTimeline.Request.DetailFilters)
			require.Equal(t, withoutFullTimeline.Request.RootDurationLonger, withFullTimeline.Request.RootDurationLonger)
			require.Equal(t, withoutFullTimeline.Request.BatchSize, withFullTimeline.Request.BatchSize)
			require.Equal(t, withoutFullTimeline.Request.Limit, withFullTimeline.Request.Limit)
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesWithListenersMapsRequest verifies listener enrichment is an explicit facade request.
func TestOpsAnalyseSlowProcessInstancesWithListenersMapsRequest(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagOpsAnalyseSlowProcessInstanceKeys = []string{"2251799813685249"}
	flagOpsAnalyseSlowProcessInstanceWithListeners = true

	got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, typex.Keys{"2251799813685249"})

	require.NoError(t, err)
	require.True(t, got.Request.WithListeners)
	require.Equal(t, ops.SlowProcessAnalysisSelectionModeExplicitKeys, got.Request.SelectionMode)
	require.Equal(t, typex.Keys{"2251799813685249"}, got.Request.InputKeys)
}

func TestOpsAnalyseSlowProcessInstancesHelpDocumentsListenerTimestampGrammar(t *testing.T) {
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Long, "s: for job creation (not worker execution start), e: for job end")
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Long, "d: for an available deadline only while the state is exactly ACTIVATED")
}

func TestOpsAnalyseSlowProcessInstancesWithListenersCommandRendersLifecycleTimestamps(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "normal"},
		{name: "full timeline", args: []string{"--with-full-timeline"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			const key = "2251799813685249"
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Append(r.Method + " " + r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/process-instances/" + key:
					_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813685249","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				case "/v2/process-instances/search":
					_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813685249","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				case "/v2/element-instances/search":
					_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:00Z","processInstanceKey":"2251799813685249","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				case "/v2/jobs/search":
					_, _ = w.Write([]byte(`{"items":[{"jobKey":"job-exec-1","kind":"EXECUTION_LISTENER","listenerEventType":"END","type":"audit-end","state":"COMPLETED","retries":0,"creationTime":"2026-09-16T13:07:16.359+02:00","endTime":"2026-09-16T13:07:16.842+02:00","deadline":"2026-09-16T13:08:00+02:00","processInstanceKey":"2251799813685249","elementInstanceKey":"element-1","elementId":"task-a","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.8"), "--no-indicator", "ops", "analyse", "slow-process-instances", "--key", key, "--with-listeners"}
			args = append(args, tc.args...)
			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)

			require.Contains(t, stdout, "job-exec-1 EXECUTION_LISTENER lsnr:END COMPLETED tp:audit-end r:0 s:2026-09-16T13:07:16.359 e:2026-09-16T13:07:16.842")
			require.NotContains(t, stdout, "d:2026-09-16T13:08:00.000")
			require.Empty(t, stderr)
			require.ElementsMatch(t, []string{
				"POST /v2/process-instances/search",
				"POST /v2/element-instances/search",
				"POST /v2/jobs/search",
				"POST /v2/jobs/search",
			}, requests.Snapshot())
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesWithListenersCommandJSONPreservesTimestamps verifies
// the execution path emits one clean envelope with optional times and retained deadlines.
func TestOpsAnalyseSlowProcessInstancesWithListenersCommandJSONPreservesTimestamps(t *testing.T) {
	var requests testx.SafeSlice[string]
	const key = "2251799813685249"
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/" + key:
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813685249","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/search":
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813685249","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/element-instances/search":
			_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:00Z","processInstanceKey":"2251799813685249","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/jobs/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			if filter["kind"] == "EXECUTION_LISTENER" {
				_, _ = w.Write([]byte(`{"items":[{"jobKey":"job-exec-1","kind":"EXECUTION_LISTENER","listenerEventType":"END","type":"audit-end","state":"COMPLETED","retries":0,"creationTime":"2026-09-16T13:07:16.359+05:30","endTime":"2026-09-16T13:07:16.842+05:30","deadline":"2026-09-16T13:08:00+05:30","processInstanceKey":"2251799813685249","elementInstanceKey":"element-1","elementId":"task-a","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				break
			}
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"), "--no-indicator", "--json",
		"ops", "analyse", "slow-process-instances", "--key", key, "--with-listeners",
	)

	require.Empty(t, stderr)
	require.ElementsMatch(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/element-instances/search",
		"POST /v2/jobs/search",
		"POST /v2/jobs/search",
	}, requests.Snapshot())
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops analyse slow-process-instances", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	items := requireJSONItems(t, payload["items"], 1)
	timeline := requireJSONItems(t, requireJSONObject(t, items[0])["timeline"], 1)
	listeners := requireJSONItems(t, requireJSONObject(t, timeline[0])["listeners"], 1)
	listener := requireJSONObject(t, listeners[0])
	require.Equal(t, "2026-09-16T13:07:16.359+05:30", listener["creationTime"])
	require.Equal(t, "2026-09-16T13:07:16.842+05:30", listener["endTime"])
	require.Equal(t, "2026-09-16T13:08:00+05:30", listener["deadline"])
}

// TestOpsAnalyseSlowProcessInstancesBuildsProcessDefinitionSearchRequests verifies selector and discovery flags normalize to search mode.
func TestOpsAnalyseSlowProcessInstancesBuildsProcessDefinitionSearchRequests(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*cobra.Command)
		wantBPMN   string
		wantPDKey  string
		wantState  process.State
		wantAfter  string
		wantBefore string
	}{
		{
			name: "bpmn selector with all state and date filters",
			setup: func(cmd *cobra.Command) {
				flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
				flagOpsAnalyseSlowProcessInstanceState = "all"
				require.NoError(t, cmd.Flags().Set("state", "all"))
				flagOpsAnalyseSlowProcessInstanceStartDateAfter = "2026-07-18T10:00:00Z"
				flagOpsAnalyseSlowProcessInstanceStartDateBefore = "2026-07-19"
				flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly = true
				flagOpsAnalyseSlowProcessInstanceBatchSize = 25
				require.NoError(t, cmd.Flags().Set("batch-size", "25"))
				flagOpsAnalyseSlowProcessInstanceLimit = 10
				require.NoError(t, cmd.Flags().Set("limit", "10"))
			},
			wantBPMN:   "OrderProcess",
			wantState:  "",
			wantAfter:  "2026-07-18T10:00:00Z",
			wantBefore: "2026-07-19T23:59:59.999999999Z",
		},
		{
			name: "process-definition-key selector with completed state",
			setup: func(cmd *cobra.Command) {
				flagOpsAnalyseSlowProcessInstancePDKey = "2251799813687001"
				flagOpsAnalyseSlowProcessInstanceState = "completed"
				require.NoError(t, cmd.Flags().Set("state", "completed"))
				flagOpsAnalyseSlowProcessInstanceEndDateAfter = "2026-07-18T10:00:00.123"
			},
			wantPDKey: "2251799813687001",
			wantState: process.StateCompleted,
			wantAfter: "2026-07-18T10:00:00.123Z",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			tc.setup(cmd)

			got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

			require.NoError(t, err)
			require.Equal(t, ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch, got.Request.SelectionMode)
			require.Equal(t, tc.wantBPMN, got.Request.ProcessDefinitionSelector.BpmnProcessID)
			require.Equal(t, tc.wantPDKey, got.Request.ProcessDefinitionSelector.ProcessDefinitionKey)
			require.Equal(t, tc.wantState, got.Request.ProcessInstanceFilters.State)
			if tc.wantBefore != "" {
				require.Equal(t, tc.wantAfter, got.Request.ProcessInstanceFilters.StartDateAfter)
				require.Equal(t, tc.wantBefore, got.Request.ProcessInstanceFilters.StartDateBefore)
				require.True(t, got.Request.ProcessInstanceFilters.NoIncidentsOnly)
				require.EqualValues(t, 25, got.Request.BatchSize)
				require.EqualValues(t, 10, got.Request.Limit)
			} else {
				require.Equal(t, tc.wantAfter, got.Request.ProcessInstanceFilters.EndDateAfter)
			}
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesBuildsDetailFilters verifies timeline filters normalize into facade input.
func TestOpsAnalyseSlowProcessInstancesBuildsDetailFilters(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
	flagOpsAnalyseSlowProcessInstanceElementID = "ReserveStock"
	flagOpsAnalyseSlowProcessInstanceType = "service_task"
	flagOpsAnalyseSlowProcessInstanceElementState = "active"
	flagOpsAnalyseSlowProcessInstanceElementDurationLonger = "2m"

	got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

	require.NoError(t, err)
	require.Equal(t, "ReserveStock", got.Request.DetailFilters.ElementID)
	require.Equal(t, "SERVICE_TASK", got.Request.DetailFilters.Type)
	require.Equal(t, "ACTIVE", got.Request.DetailFilters.ElementState)
	require.Equal(t, 2*time.Minute, got.Request.DetailFilters.DurationAfter)
}

// TestOpsAnalyseSlowProcessInstancesBuildsRootDurationFilter verifies root duration filters are separate from detail filters.
func TestOpsAnalyseSlowProcessInstancesBuildsRootDurationFilter(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
	flagOpsAnalyseSlowProcessInstanceDurationLonger = "5m"
	flagOpsAnalyseSlowProcessInstanceElementDurationLonger = "30s"

	got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

	require.NoError(t, err)
	require.Equal(t, 5*time.Minute, got.Request.RootDurationLonger)
	require.Equal(t, 30*time.Second, got.Request.DetailFilters.DurationAfter)
}

// TestOpsAnalyseSlowProcessInstancesDoesNotExposeIncidentsOnly verifies the unsupported positive incident filter is absent.
func TestOpsAnalyseSlowProcessInstancesDoesNotExposeIncidentsOnly(t *testing.T) {
	require.Nil(t, opsAnalyseSlowProcessInstancesCmd.Flags().Lookup("incidents-only"))
	require.NotContains(t, strings.ReplaceAll(opsAnalyseSlowProcessInstancesCmd.Flags().FlagUsages(), "--no-incidents-only", ""), "--incidents-only")
}

// TestOpsAnalyseSlowProcessInstancesDoesNotExposeDurationAfter verifies the old alias is not user-facing.
func TestOpsAnalyseSlowProcessInstancesDoesNotExposeDurationAfter(t *testing.T) {
	require.Nil(t, opsAnalyseSlowProcessInstancesCmd.Flags().Lookup("duration-after"))
	require.NotContains(t, opsAnalyseSlowProcessInstancesCmd.Long, "--duration-after")
	require.NotContains(t, opsAnalyseSlowProcessInstancesCmd.Example, "--duration-after")
	require.NotContains(t, opsAnalyseSlowProcessInstancesCmd.Flags().FlagUsages(), "--duration-after")
}

// resetOpsSlowProcessAnalysisTestFlags restores command globals and returns a flag-aware test command.
func resetOpsSlowProcessAnalysisTestFlags(t *testing.T) *cobra.Command {
	t.Helper()
	flagOpsAnalyseSlowProcessInstanceKeys = nil
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID = ""
	flagOpsAnalyseSlowProcessInstancePDKey = ""
	flagOpsAnalyseSlowProcessInstanceState = "all"
	flagOpsAnalyseSlowProcessInstanceStartDateAfter = ""
	flagOpsAnalyseSlowProcessInstanceStartDateBefore = ""
	flagOpsAnalyseSlowProcessInstanceEndDateAfter = ""
	flagOpsAnalyseSlowProcessInstanceEndDateBefore = ""
	flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly = false
	flagOpsAnalyseSlowProcessInstanceBatchSize = consts.MaxPISearchSize
	flagOpsAnalyseSlowProcessInstanceLimit = 0
	flagOpsAnalyseSlowProcessInstanceElementID = ""
	flagOpsAnalyseSlowProcessInstanceType = ""
	flagOpsAnalyseSlowProcessInstanceElementState = ""
	flagOpsAnalyseSlowProcessInstanceDurationLonger = ""
	flagOpsAnalyseSlowProcessInstanceElementDurationLonger = ""
	flagOpsAnalyseSlowProcessInstanceWithFullTimeline = false
	flagOpsAnalyseSlowProcessInstanceWithListeners = false
	flagCmdAutomation = false
	flagVerbose = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagQuiet = false
	flagDebug = false

	cmd := &cobra.Command{Use: "slow-process-instances"}
	cmd.SetContext(context.Background())
	flags := cmd.Flags()
	flags.StringVar(&flagOpsAnalyseSlowProcessInstanceState, "state", "all", "")
	flags.Int32Var(&flagOpsAnalyseSlowProcessInstanceBatchSize, "batch-size", consts.MaxPISearchSize, "")
	flags.Int32Var(&flagOpsAnalyseSlowProcessInstanceLimit, "limit", 0, "")
	flags.BoolVar(&flagOpsAnalyseSlowProcessInstanceWithFullTimeline, "with-full-timeline", false, "")
	flags.BoolVar(&flagOpsAnalyseSlowProcessInstanceWithListeners, "with-listeners", false, "")
	return cmd
}
