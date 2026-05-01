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
	api := tenantCommandAPI{searchTenants: func(context.Context, tenant.TenantFilter, ...foptions.FacadeOption) (tenant.Tenants, error) {
		return tenant.Tenants{
			Total: 2,
			Items: []tenant.Tenant{
				{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"},
				{TenantId: "tenant-b", Name: "Beta"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true, tenant.TenantFilter{})
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
	api := tenantCommandAPI{searchTenants: func(context.Context, tenant.TenantFilter, ...foptions.FacadeOption) (tenant.Tenants, error) {
		return tenant.Tenants{
			Total: 2,
			Items: []tenant.Tenant{
				{TenantId: "tenant-a", Name: "Alpha"},
				{TenantId: "tenant-b", Name: "Beta"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true, tenant.TenantFilter{})
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Equal(t, "tenant-a\ntenant-b\n", output)
}

func TestGetTenantListOutput_JSONUsesTenantPayload(t *testing.T) {
	resetTenantRenderFlags(t)
	flagViewAsJson = true
	cmd := newTenantListTestCommand()
	setContractSupport(cmd, ContractSupportFull)
	api := tenantCommandAPI{searchTenants: func(context.Context, tenant.TenantFilter, ...foptions.FacadeOption) (tenant.Tenants, error) {
		return tenant.Tenants{
			Total: 2,
			Items: []tenant.Tenant{
				{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"},
				{TenantId: "tenant-b", Name: "Beta"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true, tenant.TenantFilter{})
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

func TestGetTenantByKeyOutput_RendersSingleTenant(t *testing.T) {
	resetTenantRenderFlags(t)
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{getTenant: func(_ context.Context, tenantID string, opts ...foptions.FacadeOption) (tenant.Tenant, error) {
		require.Equal(t, "tenant-a", tenantID)
		return tenant.Tenant{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"}, nil
	}}

	runGetTenantByKey(cmd, api, tenantTestLogger(), true, "tenant-a")
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Contains(t, output, "tenant-a")
	require.Contains(t, output, "Alpha")
	require.Contains(t, output, "primary tenant")
	require.NotContains(t, output, "found:")
}

func TestGetTenantListOutput_FilterPassesNameContainsAndRendersMatches(t *testing.T) {
	resetTenantRenderFlags(t)
	flagGetTenantFilter = "demo"
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{searchTenants: func(_ context.Context, filter tenant.TenantFilter, opts ...foptions.FacadeOption) (tenant.Tenants, error) {
		require.Equal(t, tenant.TenantFilter{NameContains: "demo"}, filter)
		return tenant.Tenants{
			Total: 1,
			Items: []tenant.Tenant{
				{TenantId: "tenant-demo", Name: "demo tenant"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true, tenantFilterFromFlags())
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Contains(t, output, "tenant-demo")
	require.Contains(t, output, "demo tenant")
	require.Contains(t, output, "found: 1")
}

func TestGetTenantListOutput_FilterEmptyResults(t *testing.T) {
	resetTenantRenderFlags(t)
	flagGetTenantFilter = "missing"
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{searchTenants: func(_ context.Context, filter tenant.TenantFilter, opts ...foptions.FacadeOption) (tenant.Tenants, error) {
		require.Equal(t, tenant.TenantFilter{NameContains: "missing"}, filter)
		return tenant.Tenants{}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true, tenantFilterFromFlags())
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Equal(t, "found: 0\n", output)
}

func TestGetTenantListOutput_FilterKeepsPatternTextLiteral(t *testing.T) {
	resetTenantRenderFlags(t)
	flagGetTenantFilter = ".*"
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{searchTenants: func(_ context.Context, filter tenant.TenantFilter, opts ...foptions.FacadeOption) (tenant.Tenants, error) {
		require.Equal(t, tenant.TenantFilter{NameContains: ".*"}, filter)
		return tenant.Tenants{
			Total: 1,
			Items: []tenant.Tenant{
				{TenantId: "tenant-literal", Name: "demo.*"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true, tenantFilterFromFlags())
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Contains(t, output, "tenant-literal")
	require.Contains(t, output, "demo.*")
	require.Contains(t, output, "found: 1")
}

func TestGetTenantByKeyOutput_KeysOnlyUsesTenantID(t *testing.T) {
	resetTenantRenderFlags(t)
	flagViewKeysOnly = true
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{getTenant: func(_ context.Context, tenantID string, opts ...foptions.FacadeOption) (tenant.Tenant, error) {
		require.Equal(t, "tenant-a", tenantID)
		return tenant.Tenant{TenantId: "tenant-a", Name: "Alpha"}, nil
	}}

	runGetTenantByKey(cmd, api, tenantTestLogger(), true, "tenant-a")
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Equal(t, "tenant-a\n", output)
}

func TestGetTenantByKeyOutput_JSONUsesTenantPayload(t *testing.T) {
	resetTenantRenderFlags(t)
	flagViewAsJson = true
	cmd := newTenantListTestCommand()
	setContractSupport(cmd, ContractSupportFull)
	api := tenantCommandAPI{getTenant: func(_ context.Context, tenantID string, opts ...foptions.FacadeOption) (tenant.Tenant, error) {
		require.Equal(t, "tenant-a", tenantID)
		return tenant.Tenant{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"}, nil
	}}

	runGetTenantByKey(cmd, api, tenantTestLogger(), true, "tenant-a")
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get tenant", got["command"])

	payload, ok := got["payload"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "tenant-a", payload["tenantId"])
	require.Equal(t, "Alpha", payload["name"])
	require.Equal(t, "primary tenant", payload["description"])
}

func TestGetTenantCommand_RejectsWhitespaceKey(t *testing.T) {
	resetTenantRenderFlags(t)
	flagGetTenantKey = "   "
	cmd := newTenantListTestCommand()
	cmd.Flags().String("key", "", "")
	require.NoError(t, cmd.Flags().Set("key", "   "))

	err := validateTenantLookupFlags(cmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant lookup requires a non-empty --key")
}

func TestGetTenantCommand_RejectsKeyPlusFilter(t *testing.T) {
	resetTenantRenderFlags(t)
	flagGetTenantKey = "tenant-a"
	flagGetTenantFilter = "demo"
	cmd := newTenantListTestCommand()
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("filter", "", "")
	require.NoError(t, cmd.Flags().Set("key", "tenant-a"))
	require.NoError(t, cmd.Flags().Set("filter", "demo"))

	err := validateTenantLookupFlags(cmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--key cannot be combined with --filter")
}

type tenantCommandAPI struct {
	c8volt.API
	searchTenants func(context.Context, tenant.TenantFilter, ...foptions.FacadeOption) (tenant.Tenants, error)
	getTenant     func(context.Context, string, ...foptions.FacadeOption) (tenant.Tenant, error)
}

func (a tenantCommandAPI) SearchTenants(ctx context.Context, filter tenant.TenantFilter, opts ...foptions.FacadeOption) (tenant.Tenants, error) {
	return a.searchTenants(ctx, filter, opts...)
}

func (a tenantCommandAPI) GetTenant(ctx context.Context, tenantID string, opts ...foptions.FacadeOption) (tenant.Tenant, error) {
	return a.getTenant(ctx, tenantID, opts...)
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
	prevTenantKey := flagGetTenantKey
	prevTenantFilter := flagGetTenantFilter
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
		flagViewAsTree = prevTree
		flagGetTenantKey = prevTenantKey
		flagGetTenantFilter = prevTenantFilter
	})
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagViewAsTree = false
	flagGetTenantKey = ""
	flagGetTenantFilter = ""
}

func tenantTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
