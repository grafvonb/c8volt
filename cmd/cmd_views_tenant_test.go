// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/stretchr/testify/require"
)

// TestListTenantsView_HumanJSONAndKeysOnly keeps tenant collection rendering
// compatible for aligned human rows and machine-oriented output modes.
func TestListTenantsView_HumanJSONAndKeysOnly(t *testing.T) {
	resp := tenantRendererFixture()

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("tenant")

		require.NoError(t, listTenantsView(cmd, resp))

		require.Equal(t, ""+
			"<default> Default\n"+
			"tenant-a  Alpha   primary tenant\n"+
			"tenant-b  Beta\n"+
			"found: 3\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})

	t.Run("json", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewAsJson = true
		cmd := newGetViewTestCommand("tenant")
		setContractSupport(cmd, ContractSupportFull)

		require.NoError(t, listTenantsView(cmd, resp))

		var envelope map[string]any
		require.NoError(t, json.Unmarshal(cmd.OutOrStdout().(*bytes.Buffer).Bytes(), &envelope))
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		require.Equal(t, "get tenant", envelope["command"])
		payload := requireJSONObject(t, envelope["payload"])
		require.Equal(t, float64(3), payload["total"])
		items := requireJSONItems(t, payload["items"], 3)
		first := requireJSONObject(t, items[0])
		second := requireJSONObject(t, items[1])
		require.Equal(t, "<default>", first["tenantId"])
		require.Equal(t, "Default", first["name"])
		require.Equal(t, "tenant-a", second["tenantId"])
		require.Equal(t, "primary tenant", second["description"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("tenant")

		require.NoError(t, listTenantsView(cmd, resp))

		require.Equal(t, "<default>\ntenant-a\ntenant-b\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})
}

// TestTenantView_HumanJSONAndKeysOnly keeps keyed tenant rendering compatible
// across the same modes used by other get command item views.
func TestTenantView_HumanJSONAndKeysOnly(t *testing.T) {
	item := tenantRendererFixture().Items[1]

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("tenant")

		require.NoError(t, tenantView(cmd, item))

		require.Equal(t, "tenant-a Alpha primary tenant\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})

	t.Run("json", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewAsJson = true
		cmd := newGetViewTestCommand("tenant")
		setContractSupport(cmd, ContractSupportFull)

		require.NoError(t, tenantView(cmd, item))

		var envelope map[string]any
		require.NoError(t, json.Unmarshal(cmd.OutOrStdout().(*bytes.Buffer).Bytes(), &envelope))
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		require.Equal(t, "get tenant", envelope["command"])
		payload := requireJSONObject(t, envelope["payload"])
		require.Equal(t, "tenant-a", payload["tenantId"])
		require.Equal(t, "Alpha", payload["name"])
		require.Equal(t, "primary tenant", payload["description"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("tenant")

		require.NoError(t, tenantView(cmd, item))

		require.Equal(t, "tenant-a\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})
}

// tenantRendererFixture keeps tenant renderer tests on one list with an absent
// description row to protect sparse-column compaction.
func tenantRendererFixture() tenant.Tenants {
	return tenant.Tenants{
		Total: 3,
		Items: []tenant.Tenant{
			{TenantId: "<default>", Name: "Default"},
			{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"},
			{TenantId: "tenant-b", Name: "Beta"},
		},
	}
}
