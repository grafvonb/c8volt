package cmd

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/embedded"
	"github.com/spf13/cobra"
)

var (
	flagEmbedDeployFileNames []string
)

var embedDeployCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploy embedded (virtual) resources",
	Aliases: []string{"dep"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}

		all, err := embedded.List()
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}
		if len(flagEmbedDeployFileNames) == 0 {
			ferrors.HandleAndExit(log, fmt.Errorf("at least one --file is required"))
		}
		for _, f := range flagEmbedDeployFileNames {
			if !slices.Contains(all, f) {
				ferrors.HandleAndExit(log, fmt.Errorf("embedded file %q not found, run command 'embed list' to see all available embedded files, no deployment done", f))
			}
		}
		var units []resource.DeploymentUnitData
		for _, f := range flagEmbedDeployFileNames {
			data, err := fs.ReadFile(embedded.FS, f)
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("read embedded %q: %w", f, err))
			}
			log.Debug(fmt.Sprintf("deploying embedded process definition %q to tenant %s", f, cfg.App.ViewTenant()))
			units = append(units, resource.DeploymentUnitData{Name: f, Data: data})
		}

		_, err = cli.DeployProcessDefinition(cmd.Context(), cfg.App.Tenant, units, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("deploying embedded process definitions: %w", err))
		}
		log.Info(fmt.Sprintf("deployed %d embedded process definition(s) to tenant %q", len(units), cfg.App.ViewTenant()))
	},
}

func init() {
	embedCmd.AddCommand(embedDeployCmd)
	embedDeployCmd.Flags().StringSliceVarP(&flagEmbedDeployFileNames, "file", "f", nil, "embedded file(s) to deploy (repeatable)")
	_ = embedDeployCmd.MarkFlagRequired("file")
}
