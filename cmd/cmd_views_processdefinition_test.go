// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

// TestListProcessDefinitionsView_AlignsFlatRowsDynamically verifies
// process-definition scan output uses dynamic alignment without changing field order.
func TestListProcessDefinitionsView_AlignsFlatRowsDynamically(t *testing.T) {
	resetViewModeFlags(t)
	cmd := newGetViewTestCommand("process-definition")

	err := listProcessDefinitionsView(cmd, processDefinitionRendererFixture())

	require.NoError(t, err)
	require.Equal(t, ""+
		"10 <default> Order                v10\n"+
		"2  <default> Order                v10\n"+
		"21 TenantA   Invoice              v10/stable [ac:4 cp:9 cx:2 inc:3]\n"+
		"20 TenantA   Invoice              v9\n"+
		"30 tenantA   CaseSensitiveProcess v10\n"+
		"found: 5\n", cmd.OutOrStdout().(*bytes.Buffer).String())
}

// TestProcessDefinitionView_HumanJSONAndKeysOnly keeps single process-definition
// output compatible across the supported renderer modes.
func TestProcessDefinitionView_HumanJSONAndKeysOnly(t *testing.T) {
	item := processDefinitionRendererFixture().Items[2]

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("process-definition")

		require.NoError(t, processDefinitionView(cmd, item))

		require.Equal(t, "21 TenantA Invoice v10/stable [ac:4 cp:9 cx:2 inc:3]\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})

	t.Run("json", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewAsJson = true
		cmd := newGetViewTestCommand("process-definition")
		setContractSupport(cmd, ContractSupportFull)

		require.NoError(t, processDefinitionView(cmd, item))

		var envelope map[string]any
		require.NoError(t, json.Unmarshal(cmd.OutOrStdout().(*bytes.Buffer).Bytes(), &envelope))
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		payload := requireJSONObject(t, envelope["payload"])
		require.Equal(t, "21", payload["key"])
		require.Equal(t, "Invoice", payload["bpmnProcessId"])
		require.Equal(t, float64(10), payload["processVersion"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("process-definition")

		require.NoError(t, processDefinitionView(cmd, item))

		require.Equal(t, "21\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})
}

// TestListProcessDefinitionsView_JSONAndKeysOnly preserves collection payloads
// and one-key-per-line output for script consumers.
func TestListProcessDefinitionsView_JSONAndKeysOnly(t *testing.T) {
	resp := processDefinitionRendererFixture()

	t.Run("json", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewAsJson = true
		cmd := newGetViewTestCommand("process-definition")
		setContractSupport(cmd, ContractSupportFull)

		require.NoError(t, listProcessDefinitionsView(cmd, resp))

		var envelope map[string]any
		require.NoError(t, json.Unmarshal(cmd.OutOrStdout().(*bytes.Buffer).Bytes(), &envelope))
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		payload := requireJSONObject(t, envelope["payload"])
		require.Equal(t, float64(5), payload["total"])
		items := requireJSONItems(t, payload["items"], 5)
		first := requireJSONObject(t, items[0])
		third := requireJSONObject(t, items[2])
		require.Equal(t, "10", first["key"])
		require.Equal(t, "21", third["key"])
		require.Equal(t, "stable", third["processVersionTag"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("process-definition")

		require.NoError(t, listProcessDefinitionsView(cmd, resp))

		require.Equal(t, "10\n2\n21\n20\n30\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})
}

// TestListProcessDefinitionsView_PreservesCanonicalKeyOrderAcrossRenderers verifies
// every process-definition renderer consumes the same service-provided order.
func TestListProcessDefinitionsView_PreservesCanonicalKeyOrderAcrossRenderers(t *testing.T) {
	resp := processDefinitionRendererFixture()
	want := processDefinitionRendererFixtureKeys(resp)

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("process-definition")

		require.NoError(t, listProcessDefinitionsView(cmd, resp))

		require.Equal(t, want, processDefinitionKeysFromHumanOutput(cmd.OutOrStdout().(*bytes.Buffer).String()))
	})

	t.Run("json", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewAsJson = true
		cmd := newGetViewTestCommand("process-definition")
		setContractSupport(cmd, ContractSupportFull)

		require.NoError(t, listProcessDefinitionsView(cmd, resp))

		require.Equal(t, want, processDefinitionKeysFromJSONOutput(t, cmd.OutOrStdout().(*bytes.Buffer).Bytes(), len(want)))
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("process-definition")

		require.NoError(t, listProcessDefinitionsView(cmd, resp))

		require.Equal(t, want, processDefinitionKeysFromKeysOnlyOutput(cmd.OutOrStdout().(*bytes.Buffer).String()))
	})

	t.Run("watch", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("process-definition")
		candidate := (processDefinitionWatchBaseline{}).candidate(resp.Items)

		require.NoError(t, processDefinitionWatchView(cmd, candidate, false))

		require.Equal(t, want, processDefinitionKeysFromHumanOutput(cmd.OutOrStdout().(*bytes.Buffer).String()))
	})
}

// TestProcessDefinitionWatchView_AddsContractAroundOrdinaryRows verifies watch
// markers and summary wrap unchanged aligned process-definition row content.
func TestProcessDefinitionWatchView_AddsContractAroundOrdinaryRows(t *testing.T) {
	resetViewModeFlags(t)
	resp := processDefinitionRendererFixture()
	watchCmd := newGetViewTestCommand("process-definition")
	listCmd := newGetViewTestCommand("process-definition")

	err := processDefinitionWatchView(watchCmd, (processDefinitionWatchBaseline{}).candidate(resp.Items), false)
	require.NoError(t, err)
	err = listProcessDefinitionsView(listCmd, resp)
	require.NoError(t, err)

	watchOutput := watchCmd.OutOrStdout().(*bytes.Buffer).String()
	listLines := strings.Split(strings.TrimSuffix(listCmd.OutOrStdout().(*bytes.Buffer).String(), "\n"), "\n")
	expected := "Changes: 0 updated, 0 new, 0 removed   (* changed, + new)\n"
	for _, line := range listLines[:len(listLines)-1] {
		expected += "  " + line + "\n"
	}
	expected += listLines[len(listLines)-1] + "\n"
	require.Equal(t, expected, watchOutput)
}

// TestProcessDefinitionWatchView_BoldsOnlyChangedVisibleValues verifies ANSI
// styling leaves alignment, labels, padding, unchanged values, and new rows intact.
func TestProcessDefinitionWatchView_BoldsOnlyChangedVisibleValues(t *testing.T) {
	baselineItems := []process.ProcessDefinition{
		{
			Key:               "1",
			TenantId:          "old",
			BpmnProcessId:     "invoice",
			ProcessVersion:    1,
			ProcessVersionTag: "draft",
			Statistics: &process.ProcessDefinitionStatistics{
				Active:                 0,
				Completed:              2,
				Canceled:               3,
				Incidents:              0,
				IncidentCountSupported: true,
			},
		},
		{Key: "2", TenantId: "tenant-wide", BpmnProcessId: "order", ProcessVersion: 1},
	}
	currentItems := []process.ProcessDefinition{
		{
			Key:               "1",
			TenantId:          "new",
			BpmnProcessId:     "invoice-new",
			ProcessVersion:    2,
			ProcessVersionTag: "stable",
			Statistics: &process.ProcessDefinitionStatistics{
				Active:                 4,
				Completed:              2,
				Canceled:               0,
				Incidents:              1,
				IncidentCountSupported: true,
			},
		},
		baselineItems[1],
		{Key: "3", TenantId: "fresh", BpmnProcessId: "newbie", ProcessVersion: 1},
	}
	baseline := processDefinitionWatchBaseline{}
	baseline.commit(baseline.candidate(baselineItems))
	candidate := baseline.candidate(currentItems)

	plainCmd := newGetViewTestCommand("process-definition")
	styledCmd := newGetViewTestCommand("process-definition")
	require.NoError(t, processDefinitionWatchView(plainCmd, candidate, false))
	require.NoError(t, processDefinitionWatchView(styledCmd, candidate, true))

	plain := plainCmd.OutOrStdout().(*bytes.Buffer).String()
	styled := styledCmd.OutOrStdout().(*bytes.Buffer).String()
	require.Equal(t, plain, strings.NewReplacer("\x1b[1m", "", "\x1b[0m", "").Replace(styled))
	require.Contains(t, styled, "* 1 \x1b[1mnew\x1b[0m         \x1b[1minvoice-new\x1b[0m \x1b[1mv2/stable\x1b[0m [ac:\x1b[1m4\x1b[0m cp:2 cx:\x1b[1m-\x1b[0m inc:\x1b[1m1\x1b[0m]\n")
	require.Contains(t, styled, "  2 tenant-wide order       v1\n")
	require.Contains(t, styled, "+ 3 fresh       newbie      v1\n")
	require.NotContains(t, styled, "\x1b[1mcp:2")
	require.NotContains(t, styled, "+ \x1b[1m")
}

// TestProcessDefinitionWatchView_StylesOptionalTransitionsAndClears verifies
// present replacement tokens are bold while removals and identical refreshes are not.
func TestProcessDefinitionWatchView_StylesOptionalTransitionsAndClears(t *testing.T) {
	item := process.ProcessDefinition{Key: "1", TenantId: "tenant", BpmnProcessId: "invoice", ProcessVersion: 1}
	baseline := processDefinitionWatchBaseline{}
	baseline.commit(baseline.candidate([]process.ProcessDefinition{item}))

	withStats := item
	withStats.Statistics = &process.ProcessDefinitionStatistics{IncidentCountSupported: true}
	appeared := baseline.candidate([]process.ProcessDefinition{withStats})
	appearedCmd := newGetViewTestCommand("process-definition")
	require.NoError(t, processDefinitionWatchView(appearedCmd, appeared, true))
	require.Contains(t, appearedCmd.OutOrStdout().(*bytes.Buffer).String(), "[ac:\x1b[1m-\x1b[0m cp:\x1b[1m-\x1b[0m cx:\x1b[1m-\x1b[0m inc:\x1b[1m-\x1b[0m]")

	baseline.commit(appeared)
	cleared := baseline.candidate([]process.ProcessDefinition{withStats})
	clearedCmd := newGetViewTestCommand("process-definition")
	require.NoError(t, processDefinitionWatchView(clearedCmd, cleared, true))
	require.NotContains(t, clearedCmd.OutOrStdout().(*bytes.Buffer).String(), "\x1b[")

	baseline.commit(cleared)
	removed := baseline.candidate([]process.ProcessDefinition{item})
	removedCmd := newGetViewTestCommand("process-definition")
	require.NoError(t, processDefinitionWatchView(removedCmd, removed, true))
	require.Contains(t, removedCmd.OutOrStdout().(*bytes.Buffer).String(), "* 1 tenant invoice v1\n")
	require.NotContains(t, removedCmd.OutOrStdout().(*bytes.Buffer).String(), "\x1b[")
}

// TestProcessDefinitionWatchCandidate_ClassifiesDisplayedChanges verifies key
// matching, exclusive categories, removals, ordering, and display-only identity.
func TestProcessDefinitionWatchCandidate_ClassifiesDisplayedChanges(t *testing.T) {
	baselineItems := []process.ProcessDefinition{
		{Key: "a", TenantId: "tenant", BpmnProcessId: "invoice", ProcessVersion: 1, Statistics: &process.ProcessDefinitionStatistics{Active: 1}},
		{Key: "b", TenantId: "tenant", BpmnProcessId: "order", ProcessVersion: 1},
		{Key: "removed", TenantId: "tenant", BpmnProcessId: "gone", ProcessVersion: 1},
	}
	baseline := processDefinitionWatchBaseline{}
	first := baseline.candidate(baselineItems)
	baseline.commit(first)

	current := []process.ProcessDefinition{
		{Key: "b", TenantId: "tenant", BpmnProcessId: "order", ProcessVersion: 1},
		{Key: "a", TenantId: "tenant", BpmnProcessId: "invoice", ProcessVersion: 1, Statistics: &process.ProcessDefinitionStatistics{Active: 2, Completed: 3}},
		{Key: "new", TenantId: "tenant", BpmnProcessId: "wide-neighbor", ProcessVersion: 1},
	}
	candidate := baseline.candidate(current)

	require.Equal(t, 1, candidate.updated)
	require.Equal(t, 1, candidate.new)
	require.Equal(t, 1, candidate.removed)
	require.Equal(t, []byte{' ', '*', '+'}, []byte{candidate.rows[0].marker, candidate.rows[1].marker, candidate.rows[2].marker})
	require.True(t, candidate.rows[1].changed.active)
	require.True(t, candidate.rows[1].changed.completed)
}

// TestProcessDefinitionWatchCandidate_IgnoresUndisplayedAndOwnsBaselineValues
// guards against whole-model comparison and mutable statistics pointer reuse.
func TestProcessDefinitionWatchCandidate_IgnoresUndisplayedAndOwnsBaselineValues(t *testing.T) {
	stats := &process.ProcessDefinitionStatistics{Active: 1, Incidents: 9, IncidentCountSupported: false}
	item := process.ProcessDefinition{Key: "a", TenantId: "tenant", BpmnProcessId: "invoice", ProcessVersion: 1, Statistics: stats}
	baseline := processDefinitionWatchBaseline{}
	baseline.commit(baseline.candidate([]process.ProcessDefinition{item}))

	stats.Active = 2
	stats.Incidents = 10
	candidate := baseline.candidate([]process.ProcessDefinition{item})
	require.Equal(t, 1, candidate.updated, "owned active token must retain the prior value")
	require.True(t, candidate.rows[0].changed.active)
	require.False(t, candidate.rows[0].changed.incidents, "unsupported incident values are not displayed")
}

// TestProcessDefinitionWatchCandidate_TracksEmptyBaselines verifies successful
// empty results remain initialized and make later definitions new.
func TestProcessDefinitionWatchCandidate_TracksEmptyBaselines(t *testing.T) {
	baseline := processDefinitionWatchBaseline{}
	empty := baseline.candidate(nil)
	baseline.commit(empty)

	added := baseline.candidate([]process.ProcessDefinition{{Key: "a", TenantId: "tenant", BpmnProcessId: "invoice", ProcessVersion: 1}})
	require.Equal(t, 1, added.new)
	require.Equal(t, byte('+'), added.rows[0].marker)
	baseline.commit(added)

	removed := baseline.candidate(nil)
	require.Equal(t, 1, removed.removed)
	baseline.commit(removed)
	require.Zero(t, baseline.candidate(nil).removed)
}

// TestProcessDefinitionWatchCandidate_TracksOptionalDisplayTransitions verifies
// optional statistics and supported incident tokens participate in identity.
func TestProcessDefinitionWatchCandidate_TracksOptionalDisplayTransitions(t *testing.T) {
	base := process.ProcessDefinition{Key: "a", TenantId: "tenant", BpmnProcessId: "invoice", ProcessVersion: 1}
	baseline := processDefinitionWatchBaseline{}
	baseline.commit(baseline.candidate([]process.ProcessDefinition{base}))

	withStats := base
	withStats.Statistics = &process.ProcessDefinitionStatistics{}
	appeared := baseline.candidate([]process.ProcessDefinition{withStats})
	require.Equal(t, 1, appeared.updated)
	require.True(t, appeared.rows[0].changed.statisticsPresent)
	baseline.commit(appeared)

	withActive := withStats
	withActive.Statistics = &process.ProcessDefinitionStatistics{Active: 3}
	zeroTransition := baseline.candidate([]process.ProcessDefinition{withActive})
	require.Equal(t, 1, zeroTransition.updated)
	require.True(t, zeroTransition.rows[0].changed.active)
	baseline.commit(zeroTransition)

	withIncident := withActive
	withIncident.Statistics = &process.ProcessDefinitionStatistics{IncidentCountSupported: true}
	supported := baseline.candidate([]process.ProcessDefinition{withIncident})
	require.Equal(t, 1, supported.updated)
	require.True(t, supported.rows[0].changed.incidentCountSupported)
	baseline.commit(supported)

	withoutStats := baseline.candidate([]process.ProcessDefinition{base})
	require.Equal(t, 1, withoutStats.updated)
	require.True(t, withoutStats.rows[0].changed.statisticsPresent)
}

// TestProcessDefinitionWatchCandidate_IgnoresUndisplayedNameAndResetsPerSession
// proves metadata changes and another invocation's history cannot mark rows.
func TestProcessDefinitionWatchCandidate_IgnoresUndisplayedNameAndResetsPerSession(t *testing.T) {
	item := process.ProcessDefinition{Key: "a", Name: "old", TenantId: "tenant", BpmnProcessId: "invoice", ProcessVersion: 1}
	firstSession := processDefinitionWatchBaseline{}
	firstSession.commit(firstSession.candidate([]process.ProcessDefinition{item}))
	item.Name = "new"
	require.Zero(t, firstSession.candidate([]process.ProcessDefinition{item}).updated)

	secondSession := processDefinitionWatchBaseline{}
	firstRefresh := secondSession.candidate([]process.ProcessDefinition{item})
	require.Zero(t, firstRefresh.updated)
	require.Equal(t, byte(' '), firstRefresh.rows[0].marker)
}

// processDefinitionRendererFixture keeps renderer tests on one stable canonical
// payload with tenant, process, version, key, and statistics variation.
func processDefinitionRendererFixture() process.ProcessDefinitions {
	return process.ProcessDefinitions{
		Total: 5,
		Items: []process.ProcessDefinition{
			{
				Key:            "10",
				TenantId:       "<default>",
				BpmnProcessId:  "Order",
				ProcessVersion: 10,
			},
			{
				Key:            "2",
				TenantId:       "<default>",
				BpmnProcessId:  "Order",
				ProcessVersion: 10,
			},
			{
				Key:               "21",
				TenantId:          "TenantA",
				BpmnProcessId:     "Invoice",
				ProcessVersion:    10,
				ProcessVersionTag: "stable",
				Statistics: &process.ProcessDefinitionStatistics{
					Active:                 4,
					Completed:              9,
					Canceled:               2,
					Incidents:              3,
					IncidentCountSupported: true,
				},
			},
			{
				Key:            "20",
				TenantId:       "TenantA",
				BpmnProcessId:  "Invoice",
				ProcessVersion: 9,
			},
			{
				Key:            "30",
				TenantId:       "tenantA",
				BpmnProcessId:  "CaseSensitiveProcess",
				ProcessVersion: 10,
			},
		},
	}
}

// processDefinitionRendererFixtureKeys returns the fixture's process-definition
// keys in the exact sequence that renderers must preserve.
func processDefinitionRendererFixtureKeys(resp process.ProcessDefinitions) []string {
	keys := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		keys = append(keys, item.Key)
	}
	return keys
}

// processDefinitionKeysFromHumanOutput reads the leading key from each human
// process-definition row and ignores the list summary.
func processDefinitionKeysFromHumanOutput(output string) []string {
	var keys []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" || strings.HasPrefix(line, "found:") || strings.HasPrefix(line, "Changes:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 1 && (fields[0] == "*" || fields[0] == "+") {
			fields = fields[1:]
		}
		if len(fields) > 0 {
			keys = append(keys, fields[0])
		}
	}
	return keys
}

// processDefinitionKeysFromJSONOutput decodes the shared result envelope and
// returns item keys in array order.
func processDefinitionKeysFromJSONOutput(t *testing.T, output []byte, wantLen int) []string {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(output, &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	items := requireJSONItems(t, payload["items"], wantLen)
	keys := make([]string, 0, len(items))
	for _, item := range items {
		key, ok := requireJSONObject(t, item)["key"].(string)
		require.True(t, ok, "expected process-definition key")
		keys = append(keys, key)
	}
	return keys
}

// processDefinitionKeysFromKeysOnlyOutput returns one key per non-empty output
// line, matching the keys-only machine contract.
func processDefinitionKeysFromKeysOnlyOutput(output string) []string {
	var keys []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line != "" {
			keys = append(keys, line)
		}
	}
	return keys
}
