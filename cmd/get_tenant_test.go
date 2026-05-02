// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
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

func TestGetTenantListOutput_AlignsTenantIDAndNameColumns(t *testing.T) {
	resetTenantRenderFlags(t)
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{searchTenants: func(context.Context, tenant.TenantFilter, ...foptions.FacadeOption) (tenant.Tenants, error) {
		return tenant.Tenants{
			Total: 3,
			Items: []tenant.Tenant{
				{TenantId: "<default>", Name: "Default"},
				{TenantId: "dev01", Name: "Dev 01 - Development Stage 01", Description: "shared development stage"},
				{TenantId: "dev02", Name: "Dev 02"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true, tenant.TenantFilter{})
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Equal(t, strings.Join([]string{
		"<default>                Default",
		"dev01                    Dev 01 - Development Stage 01    shared development stage",
		"dev02                    Dev 02",
		"found: 3",
		"",
	}, "\n"), output)
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
	require.NotContains(t, output, "secret")
	require.NotContains(t, output, "authorization")
	require.NotContains(t, output, "members")
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

func TestGetTenantListOutput_FilterPassesTenantIDOrNameContainsAndRendersMatches(t *testing.T) {
	resetTenantRenderFlags(t)
	flagGetTenantFilter = "dev"
	cmd := newTenantListTestCommand()
	api := tenantCommandAPI{searchTenants: func(_ context.Context, filter tenant.TenantFilter, opts ...foptions.FacadeOption) (tenant.Tenants, error) {
		require.Equal(t, tenant.TenantFilter{NameContains: "dev"}, filter)
		return tenant.Tenants{
			Total: 2,
			Items: []tenant.Tenant{
				{TenantId: "dev01", Name: "Alpha"},
				{TenantId: "tenant-a", Name: "Development"},
			},
		}, nil
	}}

	runSearchTenants(cmd, api, tenantTestLogger(), true, tenantFilterFromFlags())
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Contains(t, output, "dev01")
	require.Contains(t, output, "Alpha")
	require.Contains(t, output, "tenant-a")
	require.Contains(t, output, "Development")
	require.Contains(t, output, "found: 2")
}

func TestGetTenantListOutput_FilterEmptyResults(t *testing.T) {
	resetTenantRenderFlags(t)
	flagGetTenantFilter = "missing"
	cmd := newTenantListTestCommand()
	logBuf := &bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(logBuf, nil))
	api := tenantCommandAPI{searchTenants: func(_ context.Context, filter tenant.TenantFilter, opts ...foptions.FacadeOption) (tenant.Tenants, error) {
		require.Equal(t, tenant.TenantFilter{NameContains: "missing"}, filter)
		return tenant.Tenants{}, nil
	}}

	runSearchTenants(cmd, api, log, true, tenantFilterFromFlags())
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Equal(t, "found: 0\n", output)
	require.NotContains(t, logBuf.String(), "tenant search returned no tenants")
}

func TestGetTenantListOutput_UnfilteredEmptyResultsWarnsAboutTenantAccess(t *testing.T) {
	resetTenantRenderFlags(t)
	cmd := newTenantListTestCommand()
	logBuf := &bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(logBuf, nil))
	api := tenantCommandAPI{searchTenants: func(_ context.Context, filter tenant.TenantFilter, opts ...foptions.FacadeOption) (tenant.Tenants, error) {
		require.Equal(t, tenant.TenantFilter{}, filter)
		return tenant.Tenants{}, nil
	}}

	runSearchTenants(cmd, api, log, true, tenant.TenantFilter{})
	output := cmd.OutOrStdout().(*bytes.Buffer).String()

	require.Equal(t, "found: 0\n", output)
	require.Contains(t, logBuf.String(), "level=WARN")
	require.Contains(t, logBuf.String(), "tenant search returned no tenants")
	require.Contains(t, logBuf.String(), "<default>")
	require.Contains(t, logBuf.String(), "configured client may not have access to tenant resources")
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
	require.NotContains(t, output, "secret")
	require.NotContains(t, output, "authorization")
	require.NotContains(t, output, "members")
}

func TestGetTenantCommand_V87ListReportsUnsupported(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.7
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
`)

	output, err := testx.RunCmdSubprocess(t, "TestGetTenantCommand_V87ListReportsUnsupportedHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "tenant search requires Camunda 8.8 or newer")
}

func TestGetTenantCommand_V87KeyedReportsUnsupported(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.7
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
`)

	output, err := testx.RunCmdSubprocess(t, "TestGetTenantCommand_V87KeyedReportsUnsupportedHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "tenant lookup requires Camunda 8.8 or newer")
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

func TestGetTenantHelp_DocumentsListKeyFilterAndJSONExamples(t *testing.T) {
	output := executeRootForTest(t, "get", "tenant", "--help")

	require.Contains(t, output, "./c8volt get tenant")
	require.Contains(t, output, "./c8volt get tenant --key <tenant-id>")
	require.Contains(t, output, "./c8volt get tenant --filter demo")
	require.Contains(t, output, "./c8volt get tenant --json")
	require.Contains(t, output, "./c8volt get tenant --key <tenant-id> --json")
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

func TestGetTenantCommand_V87ListReportsUnsupportedHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "tenant"}

	Execute()
}

func TestGetTenantCommand_V87KeyedReportsUnsupportedHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "tenant", "--key", "tenant-a"}

	Execute()
}
