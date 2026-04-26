// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"path/filepath"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/embedded"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var (
	flagEmbedListDetails bool
)

var embedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bundled BPMN fixture files",
	Long: "List bundled BPMN fixture files.\n\n" +
		"Run this before `embed deploy` or `embed export` when you need the exact file names. " +
		"Use `--details` to show the full embedded paths.",
	Example: `  ./c8volt embed list
  ./c8volt embed list --details
  ./c8volt --json embed list`,
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			_, noErrCodes := bootstrapFailureContext(cmd)
			ferrors.HandleAndExit(log, noErrCodes, normalizeBootstrapError(err))
		}

		files, err := embedded.List()
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}

		viewItems := make([]string, 0, len(files))
		for _, f := range files {
			view := f
			if !flagEmbedListDetails {
				view = filepath.Base(f)
			}
			viewItems = append(viewItems, view)
		}
		if flagViewAsJson {
			if err := renderJSONPayload(cmd, RenderModeJSON, viewItems); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
			}
			return
		}
		for _, view := range viewItems {
			renderOutputLine(cmd, "%s", view)
		}
	},
}

func init() {
	embedCmd.AddCommand(embedListCmd)
	embedListCmd.Flags().BoolVar(&flagEmbedListDetails, "details", false, "show full embedded file paths")
	setCommandMutation(embedListCmd, CommandMutationReadOnly)
	setContractSupport(embedListCmd, ContractSupportLimited)
	setOutputModes(embedListCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
		},
	)
}
