package cmd

import (
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var capabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Describe machine-readable CLI capabilities",
	Long: "Describe the machine-readable c8volt command surface for automation.\n" +
		"Use this command to discover command paths, flags, output modes, mutation behavior, and contract support without scraping prose help.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(toolx.ToJSONString(capabilityDocumentForRoot(Root())))
	},
}

func init() {
	rootCmd.AddCommand(capabilitiesCmd)

	setCommandMutation(capabilitiesCmd, CommandMutationReadOnly)
	setContractSupport(capabilitiesCmd, ContractSupportLimited)
	setOutputModes(capabilitiesCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
			Notes:            "canonical discovery format",
		},
		OutputModeContract{
			Name:      RenderModeOneLine.String(),
			Supported: false,
			Notes:     "use JSON discovery output",
		},
	)
}
