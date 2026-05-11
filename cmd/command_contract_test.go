// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

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
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, capability.AutomationSupport)
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
		Name:        "automation",
		Shorthand:   "",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
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

func TestCommandCapabilityForCommand_UsesExplicitAutomationOutputModes(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(runProcessInstanceCmd)

	require.Equal(t, "run process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
		{Name: "keys-only", Supported: true},
	}, capability.OutputModes)
}

func TestCommandPath_TrimsRootName(t *testing.T) {
	require.Equal(t, "", commandPath(Root()))
	require.Equal(t, "version", commandPath(versionCmd))
	require.Equal(t, "walk process-instance", commandPath(walkProcessInstanceCmd))
}

func TestCommandCapabilityForCommand_IncludesExplicitAutomationMetadata(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "demo", Short: "Demo"}
	setAutomationSupport(cmd, AutomationSupportFull, "safe for unattended execution")

	capability := commandCapabilityForCommand(cmd)

	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, "safe for unattended execution", capability.AutomationNotes)
}

func TestIsDiscoverableCommand_FiltersHiddenAndInternalCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{
			name: "nil",
			cmd:  nil,
			want: false,
		},
		{
			name: "visible public command",
			cmd:  &cobra.Command{Use: "get", Short: "Get resources"},
			want: true,
		},
		{
			name: "hidden command",
			cmd: &cobra.Command{
				Use:    "completion",
				Short:  "Shell completion",
				Hidden: true,
			},
			want: false,
		},
		{
			name: "shell completion command",
			cmd:  &cobra.Command{Use: "completion", Short: "Shell completion"},
			want: false,
		},
		{
			name: "help command",
			cmd:  &cobra.Command{Use: "help", Short: "Help"},
			want: false,
		},
		{
			name: "shell completion plumbing",
			cmd:  &cobra.Command{Use: "__complete", Short: "internal"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isDiscoverableCommand(tt.cmd))
		})
	}
}

func TestContractSupportForCommand_IgnoresHiddenChildren(t *testing.T) {
	t.Parallel()

	parent := &cobra.Command{Use: "demo", Short: "Demo"}
	hiddenChild := &cobra.Command{Use: "completion", Short: "Hidden helper", Hidden: true}
	setContractSupport(hiddenChild, ContractSupportFull)
	parent.AddCommand(hiddenChild)

	require.Equal(t, ContractSupportUnsupported, contractSupportForCommand(parent))
}

func TestCapabilityDocumentForRoot_ExcludesHiddenAndShellInternalCommands(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	publicChild := &cobra.Command{Use: "discovery-fixture", Short: "Fixture"}
	hiddenChild := &cobra.Command{Use: "completion", Short: "Shell completion", Hidden: true}
	helpChild := &cobra.Command{Use: "help", Short: "Help"}
	internalChild := &cobra.Command{Use: "__complete", Short: "internal"}
	root.AddCommand(publicChild, hiddenChild, helpChild, internalChild)
	t.Cleanup(func() {
		root.RemoveCommand(publicChild, hiddenChild, helpChild, internalChild)
	})

	doc := capabilityDocumentForRoot(root)

	var paths []string
	for _, command := range doc.Commands {
		paths = append(paths, command.Path)
	}

	require.Contains(t, paths, "discovery-fixture")
	require.NotContains(t, paths, "completion")
	require.NotContains(t, paths, "help")
	require.NotContains(t, paths, "__complete")
}

// Protects the discovery contract after removing the direct topology command and aliases.
func TestCapabilityDocumentForRoot_ExcludesRemovedClusterTopologyCommand(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	paths := commandCapabilityPaths(doc.Commands)
	require.NotContains(t, paths, "get cluster-topology")
	require.NotContains(t, paths, "get ct")
	require.NotContains(t, paths, "get cluster-info")
	require.NotContains(t, paths, "get ci")
	require.Contains(t, paths, "get cluster topology")
	require.Contains(t, paths, "get cluster version")
}

func TestCapabilityDocumentForRoot_ConfigDiagnosticsContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	show, ok := findCommandCapability(doc.Commands, "config show")
	require.True(t, ok)
	require.Equal(t, CommandMutationReadOnly, show.Mutation)
	require.Contains(t, show.Flags, FlagContract{
		Name:        "validate",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "compatibility shortcut: validate the effective configuration and exit with an error code if invalid",
	})
	require.Contains(t, show.Flags, FlagContract{
		Name:        "template",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "compatibility shortcut: print a blank configuration template",
	})

	for _, path := range []string{
		"config validate",
		"config template",
		"config test-connection",
	} {
		capability, ok := findCommandCapability(doc.Commands, path)
		require.True(t, ok, "missing command capability for %s", path)
		require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	}
}

func TestCommandCapabilityForCommand_ProcessInstanceExpectIncidentFlag(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(expectProcessInstanceCmd)

	require.Equal(t, "expect process-instance", capability.Path)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "state",
		Shorthand:   "s",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "state expectation; valid values are: [active, completed, canceled, terminated, absent]",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "incident",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "incident expectation; valid values are: [true, false]",
	})
}

func TestCommandCapabilityForCommand_ProcessInstanceVariableFlags(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getProcessInstanceCmd)

	require.Equal(t, "get process-instance", capability.Path)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-vars",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include process-instance-scope variables for keyed or list/search process-instance output",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var-value-limit",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum characters to show for variable values when --with-vars is set; 0 disables truncation",
	})
}

func TestCommandCapabilityForCommand_UpdateProcessInstanceContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(updateProcessInstanceCmd)

	require.Equal(t, "update process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process instance key(s) to update; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "vars",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "JSON object with variables to set on each process instance",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "vars-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "path to JSON object file with variables to set on each process instance",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview variable updates without submitting mutation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after the update request is accepted without variable confirmation",
	})
}

func TestCommandCapabilityForCommand_GetAndUpdateJobContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	getCapability := commandCapabilityForCommand(getJobCmd)
	require.Equal(t, "get job", getCapability.Path)
	require.Equal(t, CommandMutationReadOnly, getCapability.Mutation)
	require.Equal(t, ContractSupportFull, getCapability.ContractSupport)
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "key",
		Type:        "string",
		Required:    true,
		Repeated:    false,
		Description: "job key to inspect",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "error-message-limit",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum characters to show for error messages; 0 keeps full messages",
	})

	updateCapability := commandCapabilityForCommand(updateJobCmd)
	require.Equal(t, "update job", updateCapability.Path)
	require.Equal(t, CommandMutationStateChanging, updateCapability.Mutation)
	require.Equal(t, ContractSupportFull, updateCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, updateCapability.AutomationSupport)
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "key",
		Type:        "string",
		Required:    true,
		Repeated:    false,
		Description: "job key to update",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "retries",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "retry count to set on the job",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "timeout",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "timeout duration to submit for the job, for example 60s, 5m, or 1h",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview job updates without submitting mutation",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after the update request is accepted without retry confirmation",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "auto-confirm",
		Shorthand:   "y",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "auto-confirm prompts for non-interactive use",
	})
}

func TestCommandCapabilityForCommand_GetIncidentContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(resetGetIncidentFlagState)

	capability := commandCapabilityForCommand(getIncidentCmd)
	require.Equal(t, "get incident", capability.Path)
	require.Contains(t, capability.Aliases, "incidents")
	require.Contains(t, capability.Aliases, "inc")
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to fetch; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "error-message-limit",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum characters to show for incident messages; 0 keeps full messages",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-no-error-message",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "omit error messages from incident output",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pi-keys-only",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return only process instance keys for matching incidents",
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
	})
}

func TestCommandCapabilityForCommand_OpsPurgeOrphanProcessInstancesContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsPurgeOrphanProcessInstancesCmd)

	require.Equal(t, "ops purge orphan-process-instances", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "auto-confirmed purges")
	require.Contains(t, capability.Aliases, "orphan-pi")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "discover and validate orphan process-instance cleanup without submitting deletion requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "auto-confirm",
		Shorthand:   "y",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "auto-confirm prompts for non-interactive use",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
	})
}

func TestCapabilityDocumentForRoot_UpdateCommandFamily(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	update, ok := findCommandCapability(doc.Commands, "update")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, update.Mutation)
	require.Equal(t, ContractSupportLimited, update.ContractSupport)
	require.Contains(t, update.Aliases, "u")

	updatePI, ok := findCommandCapability(doc.Commands, "update process-instance")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, updatePI.Mutation)
	require.Equal(t, ContractSupportFull, updatePI.ContractSupport)
	require.Equal(t, AutomationSupportFull, updatePI.AutomationSupport)

	updateJob, ok := findCommandCapability(doc.Commands, "update job")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, updateJob.Mutation)
	require.Equal(t, ContractSupportFull, updateJob.ContractSupport)
	require.Equal(t, AutomationSupportFull, updateJob.AutomationSupport)
}

func TestCommandCapabilityForCommand_ResolveIncidentContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(resolveIncidentCmd)

	require.Equal(t, "resolve incident", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "inc")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to resolve; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers when resolving multiple incidents (default: min(count, GOMAXPROCS))",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview incident resolutions without submitting mutation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after the resolution request is accepted without incident confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling new incident resolutions after the first error",
	})
}

func TestCommandCapabilityForCommand_ResolveProcessInstanceContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(resolveProcessInstanceCmd)

	require.Equal(t, "resolve process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process instance key(s) to resolve; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers when resolving multiple process instances (default: min(count, GOMAXPROCS))",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview process-instance incident resolutions without submitting mutation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after resolution requests are accepted without incident confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling new process-instance resolutions after the first error",
	})
}

func TestCapabilityDocumentForRoot_ResolveCommandFamily(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	resolve, ok := findCommandCapability(doc.Commands, "resolve")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, resolve.Mutation)
	require.Equal(t, ContractSupportLimited, resolve.ContractSupport)
	require.Contains(t, resolve.Aliases, "res")

	incident, ok := findCommandCapability(doc.Commands, "resolve incident")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, incident.Mutation)
	require.Equal(t, ContractSupportFull, incident.ContractSupport)
	require.Equal(t, AutomationSupportFull, incident.AutomationSupport)
	require.Contains(t, incident.Aliases, "inc")

	processInstance, ok := findCommandCapability(doc.Commands, "resolve process-instance")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, processInstance.Mutation)
	require.Equal(t, ContractSupportFull, processInstance.ContractSupport)
	require.Equal(t, AutomationSupportFull, processInstance.AutomationSupport)
	require.Contains(t, processInstance.Aliases, "pi")
}

func TestGetJobAndUpdateJobHelp_DocumentsDiscoveryAndMutationGuards(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"get"}, []string{
		"Inspect cluster, process, job, incident, tenant, and resource state",
		"./c8volt get job --key <job-key>",
	}, nil)
	require.Contains(t, output, "job")

	output = assertCommandHelpOutput(t, []string{"get", "job"}, []string{
		"Inspect a Camunda job by key",
		"Use the jobKey exposed by incident-aware process-instance output",
		"Use --json for the stable job payload",
		"--error-message-limit",
		"Camunda 8.8 and 8.9",
		"./c8volt get job --key <job-key>",
		"./c8volt --json get job --key <job-key>",
		"--key string",
		"--error-message-limit int",
	}, nil)

	output = assertCommandHelpOutput(t, []string{"update"}, []string{
		"Update existing resources",
		"job retries and timeout by key",
		"dry-run planning",
		"submitted output",
		"./c8volt update job --key <job-key> --retries 3 --dry-run",
		"./c8volt update job --key <job-key> --timeout 5m --auto-confirm",
	}, nil)
	require.Contains(t, output, "job")

	output = assertCommandHelpOutput(t, []string{"update", "job"}, []string{
		"Update a Camunda job by key",
		"supports retries and timeout updates",
		"pre-mutation plan",
		"--dry-run previews",
		"Retry updates are confirmed by reading the job by key by default",
		"timeout updates report submitted milliseconds without deadline confirmation",
		"JSON mutations require --dry-run, --auto-confirm, or --automation",
		"--json cannot be combined with --verbose",
		"Camunda 8.7 returns an unsupported-version error before mutation",
		"./c8volt update job --key <job-key> --retries 3 --dry-run",
		"./c8volt --json update job --key <job-key> --retries 3 --auto-confirm",
		"--key string",
		"--retries int32",
		"--timeout string",
		"--dry-run",
		"--auto-confirm",
	}, nil)
}

func TestGetIncidentHelp_DocumentsAliasesPipelinesAndInheritedOutputModes(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"get", "incident"}, []string{
		"Get Camunda incidents by key or by search criteria",
		"repeated --key values or newline-separated keys from stdin with '-'",
		"Search mode defaults to active incidents",
		"./c8volt get incident --key <incident-key>",
		"./c8volt get inc --key <incident-key> --key <another-incident-key>",
		"./c8volt get incident --state resolved --error-type io_mapping_error --limit 5",
		"./c8volt get pi --with-incidents --keys-only | ./c8volt get inc -",
		"./c8volt get incident --state active --error-type job_no_retries --pi-keys-only",
		"./c8volt get incident --state active --error-type job_no_retries --pi-keys-only | ./c8volt cancel pi --dry-run -",
		"./c8volt --json get incident --key <incident-key>",
		"./c8volt --keys-only get incident --key <incident-key>",
		"--key strings",
		"--pi-keys-only",
		"return only process instance keys for matching incidents",
		"--state string",
		"--error-type string",
		"--bpmn-process-id string",
		"--pd-key string",
		"--pi-key string",
		"--root-key string",
		"--flow-node-id string",
		"--fni-key string",
		"--batch-size int32",
		"--limit int32",
		"--error-message-limit int",
		"--json",
		"--keys-only",
	}, nil)
	require.Contains(t, output, "Aliases:")
	require.Contains(t, output, "incidents")
	require.Contains(t, output, "inc")
	require.NotContains(t, output, "AD_HOC_SUB_PROCESS_NO_RETRIES")
}

func TestUpdateProcessInstanceHelp_DocumentsVariableUpdateDiscovery(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"update"}, []string{
		"Update existing resources",
		"Camunda 8.8 and 8.9",
		"unsupported-version error before these mutations",
		"./c8volt update process-instance --key <process-instance-key> --vars",
		"./c8volt update pi --key <process-instance-key> --vars-file",
		"./c8volt --automation --json update pi --key <process-instance-key> --vars",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"update", "process-instance"}, []string{
		"Update process-instance variables by key",
		"Provide exactly one variable payload source",
		"--vars with a JSON object or --vars-file with a path",
		"repeated --key values or newline-separated keys from stdin with '-'",
		"loads current process-instance-scope variables",
		"Use --dry-run to preview without mutating",
		"--auto-confirm for unattended mutation",
		"Camunda 8.7 returns an unsupported-version error before mutation",
		"./c8volt update pi --key <process-instance-key> --vars '{\"customerTier\":\"gold\"}' --dry-run",
		"./c8volt update pi --key <process-instance-key-a> --key <process-instance-key-b> --vars",
		"printf '%s\\n' \"$PROCESS_INSTANCE_KEY_A\" | ./c8volt update pi --key \"$PROCESS_INSTANCE_KEY_B\" - --vars",
		"--workers",
		"--dry-run",
		"--fail-fast",
	}, nil)
	require.Contains(t, output, "Aliases:")
	require.Contains(t, output, "pi")

	aliasOutput := assertCommandHelpOutput(t, []string{"update", "pi"}, []string{
		"Update process-instance variables by key",
		"--vars string",
		"--vars-file string",
		"--dry-run",
		"--no-wait",
	}, nil)
	require.Contains(t, aliasOutput, "Aliases:")
	require.Contains(t, aliasOutput, "pi")
}

func TestProcessInstanceSelectorValidationHelpContract(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		wants []string
	}{
		{
			name: "get pi",
			args: []string{"get", "pi", "--help"},
			wants: []string{
				"When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances.",
				"A missing selector fails with a local diagnostic instead of looking like a valid empty result",
				"--json, --automation, --keys-only, and non-TTY runs never prompt for recovery output.",
			},
		},
		{
			name: "cancel pi",
			args: []string{"cancel", "pi", "--help"},
			wants: []string{
				"When --bpmn-process-id is set, c8volt applies the selector directly to the non-mutating process-instance search.",
				"If no matching instances are found, no cancellation request is submitted.",
			},
		},
		{
			name: "delete pi",
			args: []string{"delete", "pi", "--help"},
			wants: []string{
				"When --bpmn-process-id is set, c8volt applies the selector directly to the non-mutating process-instance search.",
				"If no matching instances are found, no deletion request is submitted.",
			},
		},
		{
			name: "run pi",
			args: []string{"run", "pi", "--help"},
			wants: []string{
				"When running by BPMN process ID, c8volt validates all requested process definitions before creating anything.",
				"Mixed visible and missing BPMN IDs fail as one request, so no partial process instances are started",
				"automation-oriented modes never prompt for recovery output.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeRootForTest(t, tt.args...)
			for _, want := range tt.wants {
				require.Contains(t, output, want)
			}
		})
	}
}

// commandCapabilityPaths flattens nested discovery output so removed aliases cannot hide under `get`.
func commandCapabilityPaths(commands []CommandCapability) []string {
	var paths []string
	for _, command := range commands {
		paths = append(paths, command.Path)
		paths = append(paths, commandCapabilityPaths(command.Children)...)
	}
	return paths
}

func findCommandCapability(commands []CommandCapability, path string) (CommandCapability, bool) {
	for _, command := range commands {
		if command.Path == path {
			return command, true
		}
		if child, ok := findCommandCapability(command.Children, path); ok {
			return child, true
		}
	}
	return CommandCapability{}, false
}
