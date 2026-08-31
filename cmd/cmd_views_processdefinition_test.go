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

		require.NoError(t, processDefinitionWatchView(cmd, process.ProcessDefinitionWatchSnapshot{
			Items: resp.Items,
			Total: resp.Total,
		}))

		require.Equal(t, want, processDefinitionKeysFromHumanOutput(cmd.OutOrStdout().(*bytes.Buffer).String()))
	})
}

// TestProcessDefinitionWatchView_UsesListRenderingParity verifies watch refresh
// bodies stay byte-for-byte aligned with ordinary process-definition lists.
func TestProcessDefinitionWatchView_UsesListRenderingParity(t *testing.T) {
	resetViewModeFlags(t)
	resp := processDefinitionRendererFixture()
	watchCmd := newGetViewTestCommand("process-definition")
	listCmd := newGetViewTestCommand("process-definition")

	err := processDefinitionWatchView(watchCmd, process.ProcessDefinitionWatchSnapshot{
		Items: resp.Items,
		Total: resp.Total,
	})
	require.NoError(t, err)
	err = listProcessDefinitionsView(listCmd, resp)
	require.NoError(t, err)

	require.Equal(t, listCmd.OutOrStdout().(*bytes.Buffer).String(), watchCmd.OutOrStdout().(*bytes.Buffer).String())
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
		if line == "" || strings.HasPrefix(line, "found:") {
			continue
		}
		fields := strings.Fields(line)
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
