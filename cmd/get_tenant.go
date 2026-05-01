// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/spf13/cobra"
)

var getTenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "List tenants",
	Long: "List tenants visible to the configured environment.\n\n" +
		"Human output includes tenant ID, name, and description when available.",
	Example: `  ./c8volt get tenant
  ./c8volt get tenant --json
  ./c8volt get tenant --keys-only`,
	Aliases: []string{"tenants"},
	Run:     runGetTenant,
}

func runGetTenant(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}
	runSearchTenants(cmd, cli, log, cfg.App.NoErrCodes)
}

func runSearchTenants(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool) {
	log.Debug("searching tenants")
	tenants, err := cli.SearchTenants(cmd.Context(), collectOptions()...)
	if err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("search tenants: %w", err))
	}
	if err := listTenantsView(cmd, tenants); err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("render tenants: %w", err))
	}
	log.Debug(fmt.Sprintf("fetched tenants, found: %d items", tenants.Total))
}

func init() {
	getCmd.AddCommand(getTenantCmd)

	setCommandMutation(getTenantCmd, CommandMutationReadOnly)
	setContractSupport(getTenantCmd, ContractSupportFull)
	setAutomationSupport(getTenantCmd, AutomationSupportFull, "supports shared machine output")
}
