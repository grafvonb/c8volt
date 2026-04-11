package cmd

import (
	"path/filepath"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/embedded"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var (
	flagEmbedListDetails bool
)

var embedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List embedded (virtual) files containing process definitions",
	Example: `  ./c8volt embed list
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

		for _, f := range files {
			view := f
			if !flagEmbedListDetails {
				view = filepath.Base(f)
			}
			if flagViewAsJson {
				cmd.Println(toolx.ToJSONString(view))
			} else {
				cmd.Println(view)
			}
		}
	},
}

func init() {
	embedCmd.AddCommand(embedListCmd)
	embedListCmd.Flags().BoolVar(&flagEmbedListDetails, "details", false, "show full embedded file paths")
}
