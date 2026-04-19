package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var capabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Describe machine-readable CLI capabilities",
	Long: "Describe the machine-readable c8volt command surface for automation.\n" +
		"Use this command to discover command paths, flags, output modes, mutation behavior, contract support, and automation-mode support without scraping prose help.\n\n" +
		"Prefer `c8volt capabilities --json` when driving the CLI from AI agents, scripts, or CI. " +
		"The human-facing command taxonomy and help output remain unchanged; plain output summarizes the command surface for humans, while JSON is the repository-native discovery surface for automation, including whether each command currently supports `--automation` as the canonical non-interactive contract.",
	Example: `  ./c8volt capabilities
  ./c8volt capabilities --json`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doc := capabilityDocumentForRoot(Root())
		if flagViewAsJson {
			cmd.Println(toolx.ToJSONString(doc))
			return
		}
		renderCapabilitySummary(cmd, doc)
	},
}

func renderCapabilitySummary(cmd *cobra.Command, doc CapabilityDocument) {
	cmd.Println("Machine-readable CLI capabilities")
	cmd.Println("Use --json for the canonical discovery document and inspect automationSupport for --automation readiness.")
	cmd.Println("Use --automation as the canonical non-interactive flag on commands that report automation:full.")
	cmd.Println("")
	for _, capability := range doc.Commands {
		renderCapabilitySummaryLine(cmd, capability, 0)
	}
}

func renderCapabilitySummaryLine(cmd *cobra.Command, capability CommandCapability, depth int) {
	indent := strings.Repeat("  ", depth)
	modes := make([]string, 0, len(capability.OutputModes))
	for _, mode := range capability.OutputModes {
		if mode.Supported {
			modes = append(modes, mode.Name)
		}
	}
	cmd.Printf("%s- %s [%s, %s] modes:%s\n",
		indent,
		capability.Path,
		capability.Mutation,
		fmt.Sprintf("%s, automation:%s", capability.ContractSupport, capability.AutomationSupport),
		formatCapabilityModes(modes),
	)
	for _, child := range capability.Children {
		renderCapabilitySummaryLine(cmd, child, depth+1)
	}
}

func formatCapabilityModes(modes []string) string {
	if len(modes) == 0 {
		return " none"
	}
	return fmt.Sprintf(" %s", strings.Join(modes, ", "))
}

func init() {
	rootCmd.AddCommand(capabilitiesCmd)

	setCommandMutation(capabilitiesCmd, CommandMutationReadOnly)
	setContractSupport(capabilitiesCmd, ContractSupportLimited)
	setAutomationSupport(capabilitiesCmd, AutomationSupportFull, "canonical discovery command for automation support")
	setOutputModes(capabilitiesCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
			Notes:            "canonical discovery format",
		},
		OutputModeContract{
			Name:      RenderModeOneLine.String(),
			Supported: true,
			Notes:     "human-readable summary",
		},
	)
}
