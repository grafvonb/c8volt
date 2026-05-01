// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGetTenantListOutput_SortsAndRendersCompactRows(t *testing.T) {
	resetTenantRenderFlags(t)
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{searchTenants: func(context.Context, ...foptions.FacadeOption) (tenant.Tenants, error) {
		return tenant.Tenants{
			Total: 2,
			Items: []tenant.Tenant{
				{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"},
				{TenantId: "tenant-b", Name: "Beta"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true)
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Contains(t, output, "tenant-a")
	require.Contains(t, output, "Alpha")
	require.Contains(t, output, "primary tenant")
	require.Contains(t, output, "tenant-b")
	require.Contains(t, output, "Beta")
	require.Contains(t, output, "found: 2")
	require.Less(t, strings.Index(output, "tenant-a"), strings.Index(output, "tenant-b"))
}

func TestGetTenantListOutput_KeysOnlyUsesTenantID(t *testing.T) {
	resetTenantRenderFlags(t)
	flagViewKeysOnly = true
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{searchTenants: func(context.Context, ...foptions.FacadeOption) (tenant.Tenants, error) {
		return tenant.Tenants{
			Total: 2,
			Items: []tenant.Tenant{
				{TenantId: "tenant-a", Name: "Alpha"},
				{TenantId: "tenant-b", Name: "Beta"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true)
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Equal(t, "tenant-a\ntenant-b\n", output)
}

func TestGetTenantListOutput_JSONUsesTenantPayload(t *testing.T) {
	resetTenantRenderFlags(t)
	flagViewAsJson = true
	cmd := newTenantListTestCommand()
	setContractSupport(cmd, ContractSupportFull)
	api := tenantCommandAPI{searchTenants: func(context.Context, ...foptions.FacadeOption) (tenant.Tenants, error) {
		return tenant.Tenants{
			Total: 2,
			Items: []tenant.Tenant{
				{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"},
				{TenantId: "tenant-b", Name: "Beta"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true)
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get tenant", got["command"])

	payload, ok := got["payload"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), payload["total"])
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "tenant-a", first["tenantId"])
	require.Equal(t, "Alpha", first["name"])
	require.Equal(t, "primary tenant", first["description"])
}

type tenantCommandAPI struct {
	c8volt.API
	searchTenants func(context.Context, ...foptions.FacadeOption) (tenant.Tenants, error)
}

func (a tenantCommandAPI) SearchTenants(ctx context.Context, opts ...foptions.FacadeOption) (tenant.Tenants, error) {
	return a.searchTenants(ctx, opts...)
}

func newTenantListTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "tenant"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	parent := &cobra.Command{Use: "get"}
	parent.AddCommand(cmd)
	return cmd
}

func resetTenantRenderFlags(t *testing.T) {
	t.Helper()
	prevJSON := flagViewAsJson
	prevKeysOnly := flagViewKeysOnly
	prevTree := flagViewAsTree
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
		flagViewAsTree = prevTree
	})
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagViewAsTree = false
}

func tenantTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
