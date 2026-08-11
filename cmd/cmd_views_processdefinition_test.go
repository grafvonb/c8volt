// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
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
		"1  tenant Short                v1\n"+
		"22 tenant MuchLongerDefinition v12/stable [ac:4 cp:9 cx:2 inc:3]\n"+
		"found: 2\n", cmd.OutOrStdout().(*bytes.Buffer).String())
}

// TestProcessDefinitionView_HumanJSONAndKeysOnly keeps single process-definition
// output compatible across the supported renderer modes.
func TestProcessDefinitionView_HumanJSONAndKeysOnly(t *testing.T) {
	item := processDefinitionRendererFixture().Items[1]

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("process-definition")

		require.NoError(t, processDefinitionView(cmd, item))

		require.Equal(t, "22 tenant MuchLongerDefinition v12/stable [ac:4 cp:9 cx:2 inc:3]\n", cmd.OutOrStdout().(*bytes.Buffer).String())
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
		require.Equal(t, "22", payload["key"])
		require.Equal(t, "MuchLongerDefinition", payload["bpmnProcessId"])
		require.Equal(t, float64(12), payload["processVersion"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("process-definition")

		require.NoError(t, processDefinitionView(cmd, item))

		require.Equal(t, "22\n", cmd.OutOrStdout().(*bytes.Buffer).String())
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
		require.Equal(t, float64(2), payload["total"])
		items := requireJSONItems(t, payload["items"], 2)
		first := requireJSONObject(t, items[0])
		second := requireJSONObject(t, items[1])
		require.Equal(t, "1", first["key"])
		require.Equal(t, "22", second["key"])
		require.Equal(t, "stable", second["processVersionTag"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("process-definition")

		require.NoError(t, listProcessDefinitionsView(cmd, resp))

		require.Equal(t, "1\n22\n", cmd.OutOrStdout().(*bytes.Buffer).String())
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

// processDefinitionRendererFixture keeps renderer tests on one stable payload
// with version tags and statistics that exercise optional tail columns.
func processDefinitionRendererFixture() process.ProcessDefinitions {
	return process.ProcessDefinitions{
		Total: 2,
		Items: []process.ProcessDefinition{
			{
				Key:            "1",
				TenantId:       "tenant",
				BpmnProcessId:  "Short",
				ProcessVersion: 1,
			},
			{
				Key:               "22",
				TenantId:          "tenant",
				BpmnProcessId:     "MuchLongerDefinition",
				ProcessVersion:    12,
				ProcessVersionTag: "stable",
				Statistics: &process.ProcessDefinitionStatistics{
					Active:                 4,
					Completed:              9,
					Canceled:               2,
					Incidents:              3,
					IncidentCountSupported: true,
				},
			},
		},
	}
}
