package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestOutputModesForCommand_UsesConfiguredContractSupport(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "demo", Short: "Demo"}
	cmd.Flags().Bool("json", false, "output as JSON")
	cmd.Flags().Bool("keys-only", false, "keys only")
	setContractSupport(cmd, ContractSupportFull)

	modes := outputModesForCommand(cmd)

	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
		{Name: "keys-only", Supported: true},
	}, modes)
}

func TestCommandCapabilityForCommand_IncludesInheritedAndRequiredFlags(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getResourceCmd)

	require.Equal(t, "get resource", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportLimited, capability.ContractSupport)
	require.Contains(t, capability.Aliases, "r")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "id",
		Shorthand:   "i",
		Type:        "string",
		Required:    true,
		Repeated:    false,
		Description: "resource id to fetch",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "json",
		Shorthand:   "j",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "output as JSON (where applicable)",
	})
}

func TestCommandPath_TrimsRootName(t *testing.T) {
	require.Equal(t, "", commandPath(Root()))
	require.Equal(t, "version", commandPath(versionCmd))
	require.Equal(t, "walk process-instance", commandPath(walkProcessInstanceCmd))
}
