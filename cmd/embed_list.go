// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
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

type embeddedListFunc func() ([]string, error)

var embedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bundled BPMN fixture files",
	Long: `List bundled BPMN fixture files for the configured Camunda version.

Use before embed deploy or embed export to find exact file names. The selection matches embed deploy --all.`,
	Example: `  ./c8volt embed list
  ./c8volt embed list --details
  ./c8volt --json embed list`,
	Aliases: []string{"ls"},
	Run:     runEmbedList,
}

// runEmbedList retains bootstrap handling while wiring the compiled embedded
// filesystem into the command-local execution seam.
func runEmbedList(cmd *cobra.Command, _ []string) {
	log, _ := logging.FromContext(cmd.Context())
	cfg, err := config.FromContext(cmd.Context())
	if err != nil {
		_, noErrCodes := bootstrapFailureContext(cmd)
		ferrors.HandleAndExit(log, noErrCodes, normalizeBootstrapError(err))
	}
	runEmbedListWithListing(cmd, log, cfg, embedded.List)
}

// runEmbedListWithListing isolates ordinary listing behavior so compiled-in
// filesystem failures can be exercised without mutable global injection.
func runEmbedListWithListing(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, list embeddedListFunc) {
	files, err := list()
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
	}
	files = embeddedFilesForCamundaVersion(files, cfg.App.CamundaVersion)
	if len(files) == 0 {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no embedded files found for Camunda version %q", cfg.App.CamundaVersion.String())))
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
}

func init() {
	embedCmd.AddCommand(embedListCmd)
	embedListCmd.Flags().BoolVar(&flagEmbedListDetails, "details", false, "show full embedded file paths")
	setCommandMutation(embedListCmd, CommandMutationReadOnly)
	setContractSupport(embedListCmd, ContractSupportFull)
	setOutputModes(embedListCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
		},
	)
}
