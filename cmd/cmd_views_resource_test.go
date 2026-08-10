// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/stretchr/testify/require"
)

// TestResourceView_HumanJSONAndKeysOnly keeps resource lookup rendering
// compatible across terminal, structured, and script-oriented modes.
func TestResourceView_HumanJSONAndKeysOnly(t *testing.T) {
	item := resourceRendererFixture()

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("resource")

		require.NoError(t, resourceView(cmd, item))

		require.Equal(t, "resource-id-123 k:resource-key-123 tenant-a order-process.bpmn v7/stable\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})

	t.Run("json", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewAsJson = true
		cmd := newGetViewTestCommand("resource")
		setContractSupport(cmd, ContractSupportFull)

		require.NoError(t, resourceView(cmd, item))

		var envelope map[string]any
		require.NoError(t, json.Unmarshal(cmd.OutOrStdout().(*bytes.Buffer).Bytes(), &envelope))
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		require.Equal(t, "get resource", envelope["command"])
		payload := requireJSONObject(t, envelope["payload"])
		require.Equal(t, "resource-id-123", payload["id"])
		require.Equal(t, "resource-key-123", payload["key"])
		require.Equal(t, "order-process.bpmn", payload["name"])
		require.Equal(t, "tenant-a", payload["tenantId"])
		require.Equal(t, float64(7), payload["version"])
		require.Equal(t, "stable", payload["versionTag"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("resource")

		require.NoError(t, resourceView(cmd, item))

		require.Equal(t, "resource-id-123\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})
}

// resourceRendererFixture keeps resource renderer tests on one payload that
// exercises ID, key, tenant, name, version, and version-tag formatting.
func resourceRendererFixture() resource.Resource {
	return resource.Resource{
		ID:         "resource-id-123",
		Key:        "resource-key-123",
		Name:       "order-process.bpmn",
		TenantId:   "tenant-a",
		Version:    7,
		VersionTag: "stable",
	}
}
