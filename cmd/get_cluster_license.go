package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var getClusterLicenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Get the cluster license of the connected Camunda 8 cluster",
	Long: "Get the cluster license of the connected Camunda 8 cluster.\n\n" +
		"This read-only command requires a configured Camunda 8 connection. Prefer " +
		"`--json` when automation needs the raw license payload instead of the default rendered output.",
	Example: `  ./c8volt get cluster license
  ./c8volt get cluster license --json`,
	Run: runGetClusterLicense,
}

func init() {
	getClusterCmd.AddCommand(getClusterLicenseCmd)

	setCommandMutation(getClusterLicenseCmd, CommandMutationReadOnly)
	setContractSupport(getClusterLicenseCmd, ContractSupportLimited)
	setOutputModes(getClusterLicenseCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
		},
	)
}

func runGetClusterLicense(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}
	log.Debug("fetching cluster license")
	license, err := cli.GetClusterLicense(cmd.Context())
	if err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("get cluster license: %w", err))
	}
	cmd.Println(toolx.ToJSONString(license))
}
