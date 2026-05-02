// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/spf13/cobra"
)

var flagGetTenantKey string
var flagGetTenantFilter string

var getTenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "List tenants",
	Long: "List tenants visible to the configured environment.\n\n" +
		"Human output includes tenant ID, name, and description when available.",
	Example: `  ./c8volt get tenant
  ./c8volt get tenant --key <tenant-id>
  ./c8volt get tenant --filter demo
  ./c8volt get tenant --json
  ./c8volt get tenant --key <tenant-id> --json
  ./c8volt get tenant --keys-only`,
	Aliases: []string{"tenants"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateTenantLookupFlags(cmd)
	},
	Run: runGetTenant,
}

func runGetTenant(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}
	if key := strings.TrimSpace(flagGetTenantKey); key != "" {
		runGetTenantByKey(cmd, cli, log, cfg.App.NoErrCodes, key)
		return
	}
	runSearchTenants(cmd, cli, log, cfg.App.NoErrCodes, tenantFilterFromFlags())
}

func runGetTenantByKey(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, tenantID string) {
	log.Debug(fmt.Sprintf("getting tenant by id: %s", tenantID))
	tenant, err := cli.GetTenant(cmd.Context(), tenantID, collectOptions()...)
	if err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("get tenant: %w", err))
	}
	if err := tenantView(cmd, tenant); err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("render tenant: %w", err))
	}
}

func runSearchTenants(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, filter tenant.TenantFilter) {
	log.Debug(fmt.Sprintf("searching tenants with filter: %+v", filter))
	tenants, err := cli.SearchTenants(cmd.Context(), filter, collectOptions()...)
	if err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("search tenants: %w", err))
	}
	warnIfUnfilteredTenantSearchReturnedEmpty(log, filter, tenants)
	if err := listTenantsView(cmd, tenants); err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("render tenants: %w", err))
	}
	log.Debug(fmt.Sprintf("fetched tenants, found: %d items", tenants.Total))
}

func warnIfUnfilteredTenantSearchReturnedEmpty(log *slog.Logger, filter tenant.TenantFilter, tenants tenant.Tenants) {
	if log == nil || strings.TrimSpace(filter.NameContains) != "" || len(tenants.Items) > 0 {
		return
	}
	log.Warn("tenant search returned no tenants; Camunda creates a reserved <default> tenant, so the configured client may not have access to tenant resources")
}

func init() {
	getCmd.AddCommand(getTenantCmd)

	fs := getTenantCmd.Flags()
	fs.StringVarP(&flagGetTenantKey, "key", "k", "", "tenant ID to fetch")
	fs.StringVarP(&flagGetTenantFilter, "filter", "f", "", "literal tenant ID or name contains filter")

	setCommandMutation(getTenantCmd, CommandMutationReadOnly)
	setContractSupport(getTenantCmd, ContractSupportFull)
	setAutomationSupport(getTenantCmd, AutomationSupportFull, "supports shared machine output")
}

func validateTenantLookupFlags(cmd *cobra.Command) error {
	if cmd != nil && cmd.Flags().Changed("key") && strings.TrimSpace(flagGetTenantKey) == "" {
		return invalidFlagValuef("tenant lookup requires a non-empty --key")
	}
	if cmd != nil && cmd.Flags().Changed("key") && cmd.Flags().Changed("filter") {
		return mutuallyExclusiveFlagsf("--key cannot be combined with --filter")
	}
	return nil
}

func tenantFilterFromFlags() tenant.TenantFilter {
	return tenant.TenantFilter{NameContains: flagGetTenantFilter}
}
