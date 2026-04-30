// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/spf13/cobra"
)

var flagGetResourceID string

var getResourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "Get a resource by id",
	Long: "Get a single resource by id.\n\n" +
		"Requires --id.",
	Example: `  ./c8volt get resource --id <resource-key>
  ./c8volt --json get resource --id <resource-key>
  ./c8volt --keys-only get resource --id <resource-key>`,
	Aliases: []string{"r"},
	Args: func(cmd *cobra.Command, args []string) error {
		_, err := validatedResourceID()
		return err
	},
	Run: runGetResource,
}

func runGetResource(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}

	id, err := validatedResourceID()
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
	}

	runGetResourceByID(cmd, cli, log, cfg.App.NoErrCodes, id)
}

func runGetResourceByID(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, id string) {
	log.Debug(fmt.Sprintf("fetching resource by id: %s", id))
	resource, err := cli.GetResource(cmd.Context(), id, collectOptions()...)
	if err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("get resource: %w", err))
	}
	if err := resourceView(cmd, resource); err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("error rendering resource view: %w", err))
	}
}

func init() {
	getCmd.AddCommand(getResourceCmd)

	fs := getResourceCmd.Flags()
	fs.StringVarP(&flagGetResourceID, "id", "i", "", "resource id to fetch")
	_ = getResourceCmd.MarkFlagRequired("id")

	setCommandMutation(getResourceCmd, CommandMutationReadOnly)
	setContractSupport(getResourceCmd, ContractSupportFull)
}

func validatedResourceID() (string, error) {
	id := strings.TrimSpace(flagGetResourceID)
	if id == "" {
		return "", invalidFlagValuef("resource lookup requires a non-empty --id")
	}
	return id, nil
}
