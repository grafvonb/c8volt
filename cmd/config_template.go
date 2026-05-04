// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var configTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Print a blank configuration template",
	Long: `Print a blank configuration template.

Renders the same blank configuration template as ` + "`config show --template`" + `.`,
	Example: `  ./c8volt config template`,
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		templateCfg, yCfg, err := renderBlankConfigTemplateYAML()
		if err != nil {
			ferrors.HandleAndExit(log, templateCfg.App.NoErrCodes, err)
		}
		cmd.Println(yCfg)
	},
}

func init() {
	configCmd.AddCommand(configTemplateCmd)

	setCommandMutation(configTemplateCmd, CommandMutationReadOnly)
}
