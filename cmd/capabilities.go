// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var capabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Describe commands for scripts and agents",
	Long: "Describe the public c8volt command contract for scripts, CI jobs, and agents.\n\n" +
		"Use --json for command paths, flags, output modes, mutation behavior, contract support, and automation support.",
	Example: `  ./c8volt capabilities
  ./c8volt capabilities --json`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doc := capabilityDocumentForRoot(Root())
		if flagViewAsJson {
			if err := renderJSONPayload(cmd, RenderModeJSON, doc); err != nil {
				handleCommandError(cmd, nil, flagNoErrCodes, err)
			}
			return
		}
		renderCapabilitySummary(cmd, doc)
	},
}

func renderCapabilitySummary(cmd *cobra.Command, doc CapabilityDocument) {
	renderHumanLine(cmd, "Machine-readable public CLI capabilities")
	renderHumanLine(cmd, "Use --json for the full discovery document. Inspect automationSupport before using --automation.")
	renderHumanLine(cmd, "Hidden and shell-internal commands are excluded.")
	renderHumanLine(cmd, "")
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
	renderHumanLine(cmd, "%s- %s [%s, %s] modes:%s",
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
	setAutomationSupport(capabilitiesCmd, AutomationSupportFull, "discovery command for automation support")
	setOutputModes(capabilitiesCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
			Notes:            "full discovery format",
		},
		OutputModeContract{
			Name:      RenderModeOneLine.String(),
			Supported: true,
			Notes:     "human-readable summary",
		},
	)
}
