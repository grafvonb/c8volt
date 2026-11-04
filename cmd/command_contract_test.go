// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/tenant"
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
		Description: "resource key to fetch",
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

// TestCommandCapabilityForCommand_UsesConsistentFlagShorthands guards the
// recurring public CLI shortcuts that operators rely on across command groups.
func TestCommandCapabilityForCommand_UsesConsistentFlagShorthands(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)
	expected := map[string]string{
		"batch-size":      "n",
		"bpmn-process-id": "b",
		"key":             "k",
		"limit":           "l",
		"state":           "s",
		"workers":         "w",
	}
	for _, command := range flattenCommandCapabilities(doc.Commands) {
		for _, flag := range command.Flags {
			want, ok := expected[flag.Name]
			if !ok {
				continue
			}
			require.Equalf(t, want, flag.Shorthand, "%s --%s shorthand", command.Path, flag.Name)
		}
	}

	incidentPurge, ok := findCommandCapability(doc.Commands, "ops purge process-instances-with-incidents")
	require.True(t, ok)
	require.Contains(t, incidentPurge.Flags, FlagContract{
		Name:        "inc-key",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to select for candidate discovery",
	})
	require.False(t, hasFlagContractNamed(incidentPurge.Flags, "key"))
}

// TestCommandCapabilityForCommand_DocumentsTenantContract keeps the machine
// contract aligned with operator-facing tenant help.
func TestCommandCapabilityForCommand_DocumentsTenantContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	const tenantDescription = "tenant for discovery and creation; explicit keys use backend authorization"
	capability := commandCapabilityForCommand(getProcessInstanceCmd)

	require.Contains(t, capability.Flags, FlagContract{
		Name:        "tenant",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: tenantDescription,
	})
}

// TestRenderSucceededResult_AttachesTenantContextBesidePayload verifies the
// shared envelope adds tenant context without wrapping or reshaping payloads.
func TestRenderSucceededResult_AttachesTenantContextBesidePayload(t *testing.T) {
	resetTenantContextRenderFlags(t)
	flagViewAsJson = true
	cmd, buf := newContractTenantContextTestCommand()
	setContractSupport(cmd, ContractSupportFull)
	attachTenantContext(cmd, withTenantContextEvidence(newExplicitKeysTenantContext("tenant-a"), []string{"tenant-b"}, 0))

	require.NoError(t, renderSucceededResult(cmd, map[string]any{"items": []string{"pi-1"}}))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "cancel process-instance", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, []any{"pi-1"}, payload["items"])
	require.NotContains(t, payload, "tenantContext")
	tenantContext := requireJSONObject(t, envelope["tenantContext"])
	require.Equal(t, string(tenant.ContextModeExplicitKeys), tenantContext["mode"])
	require.Equal(t, string(tenant.ContextFilterNotApplied), tenantContext["filter"])
	require.Equal(t, "tenant-a", tenantContext["configuredTenantId"])
	require.Equal(t, []any{"tenant-b"}, tenantContext["resolvedTenantIds"])
	require.Equal(t, float64(0), tenantContext["unknownTargetCount"])
	require.Equal(t, false, tenantContext["crossTenant"])
	require.NotContains(t, tenantContext, "payload")
}

// TestRenderSucceededResult_OmitsTenantContextWhenUnattached keeps commands
// without tenant semantics on the previous envelope shape.
func TestRenderSucceededResult_OmitsTenantContextWhenUnattached(t *testing.T) {
	resetTenantContextRenderFlags(t)
	flagViewAsJson = true
	cmd, buf := newContractTenantContextTestCommand()
	setContractSupport(cmd, ContractSupportFull)

	require.NoError(t, renderSucceededResult(cmd, map[string]any{"ok": true}))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.NotContains(t, envelope, "tenantContext")
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, true, payload["ok"])
}

// newContractTenantContextTestCommand builds a nested command so envelope tests
// exercise the same command-path logic as normal subcommands.
func newContractTenantContextTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	parent := &cobra.Command{Use: "cancel"}
	cmd := &cobra.Command{Use: "process-instance"}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	parent.AddCommand(cmd)
	return cmd, buf
}

// TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata keeps command
// discovery aligned with the repainting watch contract.
func TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getProcessDefinitionCmd)

	require.Equal(t, "get process-definition", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, capability.AutomationSupport)
	require.Empty(t, capability.AutomationNotes)
	require.Equal(t, "List or fetch deployed process definitions", capability.Summary)
	require.Contains(t, capability.Aliases, "pd")
	require.Contains(t, capability.Aliases, "pds")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "watch",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "repeat the process-definition lookup until stopped",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "watch-timeout",
		Type:        "duration",
		Description: "maximum watch session duration (0 = unlimited; requires --watch)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "watch-interval",
		Type:        "duration",
		Required:    false,
		Repeated:    false,
		Description: "interval between process-definition watch refreshes after the immediate first refresh",
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "one-line",
		Supported: true,
		Notes:     "watch refreshes log a timestamped INFO summary; trailing row markers; and selective bold on supported terminals; plain fallback remains available and --watch rejects JSON/keys-only/XML/quiet/automation combinations",
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
		Notes:            "preferred for automation when not using --xml or --watch",
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
		Notes:     "finite key stream for non-watch invocations; --watch rejects keys-only output",
	})
}

// TestCommandContractFocusedModeFilesOwnLifecycleDeclarations guards the file
// cohesion contract for modes that need a dedicated lifecycle owner.
func TestCommandContractFocusedModeFilesOwnLifecycleDeclarations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		baseFile              string
		focusedFile           string
		lifecycleDeclarations []string
	}{
		{
			name:        "process definition watch",
			baseFile:    "get_processdefinition.go",
			focusedFile: "get_processdefinition_watch.go",
			lifecycleDeclarations: []string{
				"defaultGetPDWatchInterval",
				"processDefinitionWatchSleep",
				"processDefinitionWatchNow",
				"runGetProcessDefinitionWatch",
				"executeGetProcessDefinitionWatch",
				"processDefinitionWatchSlowWarningState",
				"renderGetProcessDefinitionWatchRefreshStatus",
				"resolveGetProcessDefinitionWatchInterval",
				"isGetProcessDefinitionWatchRetryable",
				"renderGetProcessDefinitionWatchRetryStatus",
				"renderGetProcessDefinitionWatchStopStatus",
				"newGetProcessDefinitionWatchSnapshotRequest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDeclarations := readGoTopLevelDeclarations(t, tt.baseFile)
			focusedDeclarations, focusedExists := readGoTopLevelDeclarationsIfExists(t, tt.focusedFile)

			for _, declaration := range tt.lifecycleDeclarations {
				if focusedExists {
					require.Containsf(t, focusedDeclarations, declaration, "%s should own %s", tt.focusedFile, declaration)
					require.NotContainsf(t, baseDeclarations, declaration, "%s must leave %s after %s exists", declaration, tt.baseFile, tt.focusedFile)
					continue
				}
				require.Containsf(t, baseDeclarations, declaration, "%s should remain discoverable in %s until %s is created", declaration, tt.baseFile, tt.focusedFile)
			}
		})
	}
}

// TestCommandCapabilityForCommand_BasicPagedReadContracts pins the public
// meaning of page size, total limits, aliases, automation, and machine output.
func TestCommandCapabilityForCommand_BasicPagedReadContracts(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	tests := []struct {
		name      string
		cmd       *cobra.Command
		aliases   []string
		batchDesc string
		limitDesc string
	}{
		{
			name:      "job",
			cmd:       getJobCmd,
			batchDesc: "number of jobs to request per page; does not cap total results (max limit 1000 enforced by server)",
			limitDesc: "maximum number of matching jobs to return across all pages; omit to continue through all matches",
		},
		{
			name:      "element",
			cmd:       getElementCmd,
			aliases:   []string{"ei"},
			batchDesc: "number of elements to request per page; does not cap total results (max limit 1000 enforced by server)",
			limitDesc: "maximum number of matching elements to return across all pages; omit to continue through all matches",
		},
		{
			name:      "incident",
			cmd:       getIncidentCmd,
			aliases:   []string{"incidents", "inc"},
			batchDesc: "incidents per discovery page (server maximum 1000)",
			limitDesc: "maximum number of matching incidents to return across all pages; omit to continue through all matches",
		},
		{
			name:      "process instance",
			cmd:       getProcessInstanceCmd,
			aliases:   []string{"process-instances", "pi", "pis"},
			batchDesc: "process instances per discovery page (server maximum 1000)",
			limitDesc: "maximum matching instances to return; omit for all matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capability := commandCapabilityForCommand(tt.cmd)
			require.Equal(t, ContractSupportFull, capability.ContractSupport)
			require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
			require.Contains(t, capability.OutputModes, OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true})
			require.Contains(t, capability.OutputModes, OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true})
			for _, alias := range tt.aliases {
				require.Contains(t, capability.Aliases, alias)
			}
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "batch-size",
				Shorthand:   "n",
				Type:        "int32",
				Required:    false,
				Repeated:    false,
				Description: tt.batchDesc,
			})
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "limit",
				Shorthand:   "l",
				Type:        "int32",
				Required:    false,
				Repeated:    false,
				Description: tt.limitDesc,
			})
		})
	}
}

// TestCommandCapabilityForCommand_ProcessInstanceMutationPagingContracts keeps
// destructive search-selected scope wording aligned with the frozen plan model.
func TestCommandCapabilityForCommand_ProcessInstanceMutationPagingContracts(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	tests := []struct {
		name      string
		cmd       *cobra.Command
		batchDesc string
		limitDesc string
	}{
		{
			name:      "cancel",
			cmd:       cancelProcessInstanceCmd,
			batchDesc: "process instances per discovery page (server maximum 1000)",
			limitDesc: "maximum matching instances to select; omit for all matches",
		},
		{
			name:      "delete",
			cmd:       deleteProcessInstanceCmd,
			batchDesc: "process instances per discovery page (server maximum 1000)",
			limitDesc: "maximum matching instances to select; omit for all matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capability := commandCapabilityForCommand(tt.cmd)
			require.Equal(t, CommandMutationStateChanging, capability.Mutation)
			require.Equal(t, ContractSupportFull, capability.ContractSupport)
			require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
			require.Contains(t, capability.Aliases, "pi")
			require.Contains(t, capability.OutputModes, OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true})
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "batch-size",
				Shorthand:   "n",
				Type:        "int32",
				Required:    false,
				Repeated:    false,
				Description: tt.batchDesc,
			})
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "limit",
				Shorthand:   "l",
				Type:        "int32",
				Required:    false,
				Repeated:    false,
				Description: tt.limitDesc,
			})
		})
	}
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

// Repair capabilities expose submitted retry values without legacy aliases.
func TestCommandContractOpsRepairIncident(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsRepairIncidentCmd)

	require.Equal(t, "ops repair incident", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
	}, capability.OutputModes)
	require.Contains(t, capability.Aliases, "inc")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to repair; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "set-retries",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "retry count to set on related jobs; 0 skips retry restoration",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-instance-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key to filter incidents",
	})
	require.False(t, hasFlagContractNamed(capability.Flags, "flow-node-id"))
	require.False(t, hasFlagContractNamed(capability.Flags, "fni-key"))
}

// TestCommandContractOpsAnalyseSlowProcessInstances captures the read-only analysis machine contract.
func TestCommandContractOpsAnalyseSlowProcessInstances(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsAnalyseSlowProcessInstancesCmd)

	require.Equal(t, "ops analyse slow-process-instances", capability.Path)
	require.Contains(t, capability.Flags, FlagContract{
		Name: "bid", Type: "string", Required: false, Repeated: false,
		Description: "exact process business ID to select (only c8.10+)",
	})

	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "slow-pi")
	require.Contains(t, capability.Aliases, "spi")
	require.Contains(t, opsAnalyseCmd.Aliases, "analyze")
	aliasCmd, remaining, err := root.Find([]string{"ops", "analyze", "slow-process-instances"})
	require.NoError(t, err)
	require.Empty(t, remaining)
	require.Same(t, opsAnalyseSlowProcessInstancesCmd, aliasCmd)
	aliasCmd, remaining, err = root.Find([]string{"ops", "analyse", "spi"})
	require.NoError(t, err)
	require.Empty(t, remaining)
	require.Same(t, opsAnalyseSlowProcessInstancesCmd, aliasCmd)

	require.NotContains(t, opsAnalyseSlowProcessInstancesCmd.Long, "--duration-after")

	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true, Notes: "stdout remains one JSON document; preflight and frozen-scope metadata are exposed as result fields"},
		{Name: "keys-only", Supported: true, Notes: "stdout remains one process-instance key per line with no progress or preflight text"},
	}, capability.OutputModes)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process-instance key(s) to analyse; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to discover process instances",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process definition key to discover process instances",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "state",
		Shorthand:   "s",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "state to filter discovered process instances: all, active, suspended (only c8.10+), completed, canceled, terminated",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-incidents-only",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "only include process instances without incidents during discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "batch-size",
		Shorthand:   "n",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "process instances per discovery page (server maximum 1000)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "limit",
		Shorthand:   "l",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "maximum matching instances to analyse; omit for all matches",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dur-longer",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "only include process instances whose whole duration is longer than this duration, for example 5m or 1h30m",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dur-element-longer",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "only include process instances with elements or transitions longer than this duration, for example 30s or 2m",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-full-timeline",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "show complete chronological element and transition detail",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-listeners",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime listener jobs",
	})
	require.False(t, hasFlagContractNamed(capability.Flags, "duration-after"))
	require.False(t, hasFlagContractNamed(capability.Flags, "incidents-only"))
}

// TestCommandCapabilityForCommand_OpsPagedDiscoveryFlagContracts verifies discovery flags describe page size and explicit caps distinctly.
func TestCommandCapabilityForCommand_OpsPagedDiscoveryFlagContracts(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	resetOpsRepairIncidentFlagState()
	resetProcessInstanceCommandGlobals()
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)
	t.Cleanup(resetOpsRepairIncidentFlagState)
	t.Cleanup(resetProcessInstanceCommandGlobals)
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	tests := []struct {
		name      string
		cmd       *cobra.Command
		batchDesc string
		limitDesc string
	}{
		{
			name:      "incident purge",
			cmd:       opsPurgeProcessInstancesWithIncidentsCmd,
			batchDesc: "incidents per discovery page (server maximum 1000)",
			limitDesc: "maximum matching incidents to select; omit for all matches",
		},
		{
			name:      "repair incident",
			cmd:       opsRepairIncidentCmd,
			batchDesc: "incidents per discovery page (server maximum 1000)",
			limitDesc: "maximum number of matching incidents to freeze for repair; omit to discover all matches",
		},
		{
			name:      "repair process-instance",
			cmd:       opsRepairProcessInstanceCmd,
			batchDesc: "process instances per discovery page (server maximum 1000)",
			limitDesc: "maximum number of matching process instances to freeze for repair; omit to discover all matches",
		},
		{
			name:      "all process definitions purge",
			cmd:       opsPurgeAllProcessDefinitionsCmd,
			batchDesc: "process definitions per discovery page (server maximum 1000)",
			limitDesc: "maximum number of matching process definitions to freeze for purge; omit to discover all matches",
		},
		{
			name:      "retention policy",
			cmd:       opsExecuteRetentionPolicyCmd,
			batchDesc: "process instances per discovery page (server maximum 1000)",
			limitDesc: "maximum retention candidates to select; omit for all matches",
		},
		{
			name:      "slow process analysis",
			cmd:       opsAnalyseSlowProcessInstancesCmd,
			batchDesc: "process instances per discovery page (server maximum 1000)",
			limitDesc: "maximum matching instances to analyse; omit for all matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capability := commandCapabilityForCommand(tt.cmd)
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "batch-size",
				Shorthand:   "n",
				Type:        "int32",
				Required:    false,
				Repeated:    false,
				Description: tt.batchDesc,
			})
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "limit",
				Shorthand:   "l",
				Type:        "int32",
				Required:    false,
				Repeated:    false,
				Description: tt.limitDesc,
			})
		})
	}
}

// TestCommandContractOpsRepairProcessInstance verifies the process-instance repair target exposes automation metadata.
func TestCommandContractOpsRepairProcessInstance(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsRepairProcessInstanceCmd)

	require.Equal(t, "ops repair process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
	}, capability.OutputModes)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process-instance key(s) whose active incidents should be repaired; repeat or combine with stdin '-'",
	})
	for _, flag := range capability.Flags {
		require.NotEqual(t, "incidents-only", flag.Name)
	}
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "direct-incidents-only",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "select only process instances with direct active incidents",
	})
}

func TestCommandPath_TrimsRootName(t *testing.T) {
	require.Equal(t, "", commandPath(Root()))
	require.Equal(t, "version", commandPath(versionCmd))
	require.Equal(t, "walk process-instance", commandPath(walkProcessInstanceCmd))
}

// TestCommandCapabilityForCommand_WalkProcessInstanceElementFlagAndContract verifies walk element enrichment is discoverable as read-only command metadata.
func TestCommandCapabilityForCommand_WalkProcessInstanceElementFlagAndContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(walkProcessInstanceCmd)

	require.Equal(t, "walk process-instance", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Aliases, "pis")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-elements",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "show runtime element instances for selected process-instance walks",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-listeners",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime listener jobs; requires --with-elements",
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "one-line", Supported: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
}

// TestCommandContractWalkProcessInstanceWithListeners verifies listener
// enrichment is exposed in walk command metadata and user-facing examples.
func TestCommandContractWalkProcessInstanceWithListeners(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(walkProcessInstanceCmd)

	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-listeners",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime listener jobs; requires --with-elements",
	})
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

// TestAllTenantsSupportForCommand_DefaultsAccepted verifies unannotated commands
// inherit the safe parsing contract for the root override.
func TestAllTenantsSupportForCommand_DefaultsAccepted(t *testing.T) {
	t.Parallel()

	require.Equal(t, AllTenantsSupportAccepted, allTenantsSupportForCommand(nil))
	require.Equal(t, AllTenantsSupportAccepted, allTenantsSupportForCommand(&cobra.Command{Use: "demo"}))
}

// TestAllTenantsSupportForCommand_UsesExplicitAnnotation keeps runtime validation
// and future capability serialization on a single command annotation resolver.
func TestAllTenantsSupportForCommand_UsesExplicitAnnotation(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "deploy process-definition"}
	setAllTenantsSupport(cmd, AllTenantsSupportRejectedConcreteDestination)

	require.Equal(t, AllTenantsSupportRejectedConcreteDestination, allTenantsSupportForCommand(cmd))
	require.Equal(t, string(AllTenantsSupportRejectedConcreteDestination), cmd.Annotations[allTenantsSupportAnnotation])
}

// TestAllTenantsSupportForCommand_ConcreteDestinationInventory pins the
// complete command set whose creation semantics reject all-tenants at runtime.
func TestAllTenantsSupportForCommand_ConcreteDestinationInventory(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	tests := []struct {
		path string
		cmd  *cobra.Command
	}{
		{path: "deploy process-definition", cmd: deployProcessDefinitionCmd},
		{path: "embed deploy", cmd: embedDeployCmd},
		{path: "run process-instance", cmd: runProcessInstanceCmd},
		{path: "ops execute smoke-test", cmd: opsExecuteSmokeTestCmd},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			require.Equal(t, tt.path, commandPath(tt.cmd))
			require.Equal(t, AllTenantsSupportRejectedConcreteDestination, allTenantsSupportForCommand(tt.cmd))
		})
	}
}

// TestCommandCapabilityForCommand_IncludesAllTenantsSupport keeps capability
// metadata aligned with the same resolver used by root validation.
func TestCommandCapabilityForCommand_IncludesAllTenantsSupport(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	tests := []struct {
		path string
		cmd  *cobra.Command
		want AllTenantsSupport
	}{
		{path: "get process-instance", cmd: getProcessInstanceCmd, want: AllTenantsSupportAccepted},
		{path: "cancel process-instance", cmd: cancelProcessInstanceCmd, want: AllTenantsSupportAccepted},
		{path: "get resource", cmd: getResourceCmd, want: AllTenantsSupportAccepted},
		{path: "deploy process-definition", cmd: deployProcessDefinitionCmd, want: AllTenantsSupportRejectedConcreteDestination},
		{path: "embed deploy", cmd: embedDeployCmd, want: AllTenantsSupportRejectedConcreteDestination},
		{path: "run process-instance", cmd: runProcessInstanceCmd, want: AllTenantsSupportRejectedConcreteDestination},
		{path: "ops execute smoke-test", cmd: opsExecuteSmokeTestCmd, want: AllTenantsSupportRejectedConcreteDestination},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			capability := commandCapabilityForCommand(tt.cmd)

			require.Equal(t, tt.path, capability.Path)
			require.Equal(t, tt.want, capability.AllTenantsSupport)
			require.Equal(t, allTenantsSupportForCommand(tt.cmd), capability.AllTenantsSupport)
		})
	}
}

// TestCapabilityDocumentForRoot_KeepsV1WithAllTenantsSupport proves the new
// support metadata is additive and does not require a document version bump.
func TestCapabilityDocumentForRoot_KeepsV1WithAllTenantsSupport(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	require.Equal(t, "v1", doc.Version)
	searchCapability, ok := findCommandCapability(doc.Commands, "get process-instance")
	require.True(t, ok)
	require.Equal(t, AllTenantsSupportAccepted, searchCapability.AllTenantsSupport)
	destinationCapability, ok := findCommandCapability(doc.Commands, "run process-instance")
	require.True(t, ok)
	require.Equal(t, AllTenantsSupportRejectedConcreteDestination, destinationCapability.AllTenantsSupport)
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

// TestCapabilityDocumentForRoot_CoversCLIDebtAssessment preserves the historical baseline
// and accounts explicitly for commands added after that assessment.
func TestCapabilityDocumentForRoot_CoversCLIDebtAssessment(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)
	paths := commandCapabilityPaths(doc.Commands)
	require.Len(t, paths, 62)

	assessmentPaths := readCLIDebtAssessmentCommandPaths(t)
	require.Len(t, assessmentPaths, 57)
	require.ElementsMatch(t, paths, append(assessmentPaths, "update element", "suspend", "suspend process-instance", "resume", "resume process-instance"))
}

// TestCommandCapabilityForCommand_UserTaskReadContract verifies every alias
// resolves to the canonical read-only command with documented local-variable search support.
func TestCommandCapabilityForCommand_UserTaskReadContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getUserTaskCmd)
	require.Equal(t, "get user-task", capability.Path)
	require.ElementsMatch(t, []string{"user-tasks", "ut", "uts"}, capability.Aliases)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "unattended reads")
	require.Equal(t, []OutputModeContract{
		{Name: RenderModeOneLine.String(), Supported: true},
		{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		{Name: RenderModeKeysOnly.String(), Supported: true},
	}, capability.OutputModes)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-vars",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include effective variables for selected user tasks",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var-value-limit",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum characters to show for variable values when --with-vars is set; 0 disables truncation",
	})
	for _, expected := range []FlagContract{
		{
			Name:        "var-exists",
			Type:        "stringArray",
			Required:    false,
			Repeated:    true,
			Description: "require local variable name(s) to exist; repeat or separate names with commas",
		},
		{
			Name:        "var",
			Type:        "stringArray",
			Required:    false,
			Repeated:    true,
			Description: "require local variable equality or advanced clause(s); repeat or separate clauses with commas",
		},
		{
			Name:        "var-like",
			Type:        "stringArray",
			Required:    false,
			Repeated:    true,
			Description: "require local variable value pattern clause(s); repeat or separate clauses with commas",
		},
	} {
		require.Contains(t, capability.Flags, expected)
	}
	for _, excluded := range []string{"forms", "audit-log", "date", "sort", "watch"} {
		require.False(t, hasFlagContractNamed(capability.Flags, excluded), "out-of-scope --%s flag was advertised", excluded)
	}

	for _, alias := range capability.Aliases {
		resolved, remaining, err := root.Find([]string{"get", alias})
		require.NoError(t, err)
		require.Empty(t, remaining)
		require.Same(t, getUserTaskCmd, resolved)
	}
}

// TestCommandCapabilityForCommand_UpdateUserTaskContract verifies aliases,
// mutation metadata, automation, and the four separate user-task operations.
func TestCommandCapabilityForCommand_UpdateUserTaskContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(updateUserTaskCmd)
	require.Equal(t, "update user-task", capability.Path)
	require.ElementsMatch(t, []string{"user-tasks", "ut", "uts"}, capability.Aliases)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, []OutputModeContract{
		{Name: RenderModeOneLine.String(), Supported: true},
		{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		{Name: RenderModeKeysOnly.String(), Supported: true},
	}, capability.OutputModes)
	for _, name := range []string{"key", "complete", "set-assignee", "unassign", "force", "set-vars", "set-vars-file", "dry-run", "no-wait", "workers", "no-worker-limit", "fail-fast"} {
		require.True(t, hasFlagContractNamed(capability.Flags, name), "missing --%s", name)
	}
	require.False(t, flagUpdateUserTaskComplete)
	require.NoError(t, updateUserTaskCmd.Flags().Set("complete", "true"))
	require.NoError(t, updateUserTaskCmd.Flags().Set("key", ""))
	require.True(t, updateUserTaskCmd.Flags().Lookup("key").Value.(*userTaskKeyValue).emptyOccurrence)
	resetCommandTreeFlags(root)
	require.False(t, flagUpdateUserTaskComplete)
	require.False(t, updateUserTaskCmd.Flags().Lookup("key").Value.(*userTaskKeyValue).emptyOccurrence)

	require.Contains(t, updateUserTaskCmd.Flags().Lookup("complete").Usage, "await backend confirmation")
	require.Contains(t, updateUserTaskCmd.Flags().Lookup("no-wait").Usage, "skip variable-update verification")
	require.Contains(t, updateUserTaskCmd.Flags().Lookup("no-wait").Usage, "incompatible with --complete")

	for _, alias := range capability.Aliases {
		resolved, remaining, err := root.Find([]string{"update", alias})
		require.NoError(t, err)
		require.Empty(t, remaining)
		require.Same(t, updateUserTaskCmd, resolved)
	}
}

// TestCommandCapabilityForCommand_ProcessInstanceExpectIncidentFlag preserves state and incident flag metadata for automation.
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
		Description: "state expectation; valid values are: [active, suspended, completed, canceled, terminated, absent]; suspended: only c8.10+",
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
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var-exists",
		Type:        "stringArray",
		Required:    false,
		Repeated:    true,
		Description: "require variable name(s) to exist; repeat or separate names with commas",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var",
		Type:        "stringArray",
		Required:    false,
		Repeated:    true,
		Description: "require variable equality or advanced clause(s); repeat or separate clauses with commas",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var-like",
		Type:        "stringArray",
		Required:    false,
		Repeated:    true,
		Description: "require variable value pattern clause(s); repeat or separate clauses with commas",
	})
}

// TestCommandCapabilityForCommand_ProcessInstanceElementFlagAndContract verifies element enrichment is discoverable as read-only command metadata.
func TestCommandCapabilityForCommand_ProcessInstanceElementFlagAndContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getProcessInstanceCmd)

	require.Equal(t, "get process-instance", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Aliases, "pis")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-elements",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime element instances for keyed or list/search process-instance output",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-listeners",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime listener jobs; requires --with-elements",
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "one-line", Supported: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
}

// TestCommandContractGetProcessInstanceWithListeners verifies listener
// enrichment is exposed in the process-instance command's automation metadata.
func TestCommandContractGetProcessInstanceWithListeners(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getProcessInstanceCmd)

	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-listeners",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime listener jobs; requires --with-elements",
	})
}

// TestCommandCapabilityForCommand_UpdateProcessInstanceContract keeps both mutation modes discoverable.
func TestCommandCapabilityForCommand_UpdateProcessInstanceContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(updateProcessInstanceCmd)

	require.Contains(t, capability.Flags, FlagContract{
		Name: "set-bid", Type: "string",
		Description: "assign an exact native business ID to explicit process keys (only c8.10+)",
	})

	require.Equal(t, "update process-instance", capability.Path)
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
		Description: "process instance key(s) to update; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "set-vars",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "JSON object with variables to set on each process instance",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "set-vars-file",
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
		Description: "preview variable updates or business-ID assignment without submitting mutation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after the update request is accepted without confirmation",
	})
}

// TestCommandCapabilityForCommand_GetAndUpdateJobContract keeps collection, selector and legacy metadata aligned with execution.
func TestCommandCapabilityForCommand_GetAndUpdateJobContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	getCapability := commandCapabilityForCommand(getJobCmd)
	require.Equal(t, "get job", getCapability.Path)
	require.Equal(t, CommandMutationReadOnly, getCapability.Mutation)
	require.Equal(t, ContractSupportFull, getCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, getCapability.AutomationSupport)
	require.Contains(t, getCapability.AutomationNotes, "unattended job reads")
	require.Contains(t, getCapability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.Contains(t, getCapability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})

	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "job key for exact lookup; omit to list or search jobs",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "state",
		Shorthand:   "s",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "Camunda job state to filter in search mode; case-insensitive",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "type",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "job type to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "pi-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process instance key to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "element-instance-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "worker",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "worker name to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "retries",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "exact retry count to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "kind",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "Camunda job kind to filter in search mode; case-insensitive",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "listener-event-type",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "listener event type to filter in search mode; case-insensitive",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "limit",
		Shorthand:   "l",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "maximum number of matching jobs to return across all pages; omit to continue through all matches",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "batch-size",
		Shorthand:   "n",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "number of jobs to request per page; does not cap total results (max limit 1000 enforced by server)",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "total",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return only the numeric total of matching jobs",
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
	require.Contains(t, updateCapability.AutomationNotes, "non-mutating dry-run previews")
	require.Contains(t, updateCapability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.NotContains(t, updateCapability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})

	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringArray",
		Required:    false,
		Repeated:    true,
		Description: "job key to update; for --complete, repeat or use comma-separated keys and stdin '-'",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "set-retries",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "retry count to set or remaining retries for --fail (including 0)",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "set-timeout",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "timeout duration to submit for the job, for example 60s, 5m, or 1h",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "fail",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "report a technical job failure",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "set-retry-backoff",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "duration before a failed job becomes retryable, for example 60s, 5m, or 1h",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "set-error-message",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "operator message for worker outcome modes",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "throw-bpmn-error",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "throw a BPMN error; requires --set-error-code",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "complete",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "complete the job through the worker outcome API",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "set-vars",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "JSON object with variables for BPMN error or completion outcomes",
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
		Description: "skip retry confirmation; bulk completion permits one reconciliation read without polling",
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

// TestCommandCapabilityForCommand_GetElementContract verifies discovery
// metadata for the runtime element read command added to the get family.
func TestCommandCapabilityForCommand_GetElementContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getElementCmd)

	require.Equal(t, "get element", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "unattended element reads")
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
	require.Equal(t, []string{"ei"}, capability.Aliases)
	require.Contains(t, capability.Flags, FlagContract{Name: "with-vars", Type: "bool", Description: "include complete element-local variables; skipped for keys-only and total output"})

	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key for exact lookup; omit to list or search runtime elements",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pi-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process instance key to filter in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "state",
		Shorthand:   "s",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "runtime element state to filter in search mode; case-insensitive",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "type",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "runtime element type to filter in search mode; case-insensitive",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process definition key to filter in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to filter in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "batch-size",
		Shorthand:   "n",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "number of elements to request per page; does not cap total results (max limit 1000 enforced by server)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "limit",
		Shorthand:   "l",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "maximum number of matching elements to return across all pages; omit to continue through all matches",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "total",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return only the numeric total of matching elements",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-listeners",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime listener jobs",
	})
}

// TestCommandCapabilityForCommand_GetIncidentContract verifies canonical CLI behavior with isolated fixtures.
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
		Name:        "no-error-message",
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
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to validate and filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-instance-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key to filter incidents",
	})
	require.False(t, hasFlagContractNamed(capability.Flags, "flow-node-id"))
	require.False(t, hasFlagContractNamed(capability.Flags, "fni-key"))
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

func TestCommandCapabilityForCommand_BpmnSelectorAlignedCommandContracts(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	t.Cleanup(resetGetIncidentFlagState)

	tests := []struct {
		name              string
		cmd               *cobra.Command
		path              string
		mutation          CommandMutation
		automation        AutomationSupport
		wantAutomation    string
		wantOutputMode    OutputModeContract
		wantBpmnFlag      string
		wantDryRun        bool
		wantAutoConfirm   bool
		wantProcessKeysIn bool
	}{
		{
			name:              "get process-instance",
			cmd:               getProcessInstanceCmd,
			path:              "get process-instance",
			mutation:          CommandMutationReadOnly,
			automation:        AutomationSupportFull,
			wantOutputMode:    OutputModeContract{Name: "keys-only", Supported: true},
			wantBpmnFlag:      "BPMN process ID to filter process instances",
			wantProcessKeysIn: true,
		},
		{
			name:            "cancel process-instance",
			cmd:             cancelProcessInstanceCmd,
			path:            "cancel process-instance",
			mutation:        CommandMutationStateChanging,
			automation:      AutomationSupportFull,
			wantAutomation:  "unattended destructive confirmation",
			wantOutputMode:  OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
			wantBpmnFlag:    "BPMN process ID to filter process instances",
			wantDryRun:      true,
			wantAutoConfirm: true,
		},
		{
			name:            "delete process-instance",
			cmd:             deleteProcessInstanceCmd,
			path:            "delete process-instance",
			mutation:        CommandMutationStateChanging,
			automation:      AutomationSupportFull,
			wantAutomation:  "unattended destructive confirmation",
			wantOutputMode:  OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
			wantBpmnFlag:    "BPMN process ID to filter process instances",
			wantDryRun:      true,
			wantAutoConfirm: true,
		},
		{
			name:           "get incident",
			cmd:            getIncidentCmd,
			path:           "get incident",
			mutation:       CommandMutationReadOnly,
			automation:     AutomationSupportFull,
			wantOutputMode: OutputModeContract{Name: "keys-only", Supported: true},
			wantBpmnFlag:   "BPMN process ID to validate and filter incidents",
		},
		{
			name:           "get process-definition",
			cmd:            getProcessDefinitionCmd,
			path:           "get process-definition",
			mutation:       CommandMutationReadOnly,
			automation:     AutomationSupportUnsupported,
			wantOutputMode: OutputModeContract{Name: "json", Supported: true, MachinePreferred: true, Notes: "preferred for automation when not using --xml or --watch"},
			wantBpmnFlag:   "BPMN process ID to filter process instances",
		},
		{
			name:            "delete process-definition",
			cmd:             deleteProcessDefinitionCmd,
			path:            "delete process-definition",
			mutation:        CommandMutationStateChanging,
			automation:      AutomationSupportFull,
			wantAutomation:  "unattended destructive confirmation",
			wantOutputMode:  OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
			wantBpmnFlag:    "BPMN process ID of the process definition (all versions) to delete",
			wantDryRun:      true,
			wantAutoConfirm: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capability := commandCapabilityForCommand(tt.cmd)
			require.Equal(t, tt.path, capability.Path)
			require.Equal(t, tt.mutation, capability.Mutation)
			require.Equal(t, ContractSupportFull, capability.ContractSupport)
			require.Equal(t, tt.automation, capability.AutomationSupport)
			if tt.wantAutomation != "" {
				require.Contains(t, capability.AutomationNotes, tt.wantAutomation)
			}
			require.Contains(t, capability.OutputModes, tt.wantOutputMode)
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "bpmn-process-id",
				Shorthand:   "b",
				Type:        "string",
				Required:    false,
				Repeated:    false,
				Description: tt.wantBpmnFlag,
			})
			if tt.wantDryRun {
				require.True(t, hasFlagContractNamed(capability.Flags, "dry-run"), "missing dry-run flag")
			}
			if tt.wantAutoConfirm {
				require.Contains(t, capability.Flags, FlagContract{
					Name:        "auto-confirm",
					Shorthand:   "y",
					Type:        "bool",
					Required:    false,
					Repeated:    false,
					Description: "auto-confirm prompts for non-interactive use",
				})
			}
			if tt.wantProcessKeysIn {
				require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
			}
		})
	}
}

// Verifies run process-instance advertises keys-only output for command composition.
func TestCommandCapabilityForCommand_RunProcessInstanceKeysOnlyContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(runProcessInstanceCmd)

	require.Equal(t, "run process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "BPMN process IDs to start; latest version unless --pd-version is set",
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
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed purges")
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

// TestCommandCapabilityForCommand_OpsPurgeAllProcessDefinitionsContract verifies discovery metadata for the all-process-definitions purge command.
func TestCommandCapabilityForCommand_OpsPurgeAllProcessDefinitionsContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	capability := commandCapabilityForCommand(opsPurgeAllProcessDefinitionsCmd)

	require.Equal(t, "ops purge all-process-definitions", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed all-process-definitions purges")
	require.Contains(t, capability.Aliases, "all-pds")
	require.NotContains(t, capability.Aliases, "purge-definitions")
	require.NotContains(t, capability.Aliases, "delete-all")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "one-line",
		Supported: true,
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process definition key to select for candidate discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to filter candidate process definitions",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "latest",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "only include the latest matching process-definition version(s)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-version",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "process definition version to filter candidate discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-version-tag",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process definition version tag to filter candidate discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "discover and validate process-definition cleanup without submitting deletion requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "write an audit report to the given path",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-format",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "audit report format: markdown, json (default inferred from report-file extension)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
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
		Name:        "force",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "force cancellation of affected unfinished process instances before deleting process definitions",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after deletion requests are accepted without deletion confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling validation or deletion work after the first error",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-worker-limit",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "use all queued jobs as workers when --workers is unset",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers; omit for an automatic limit",
	})
	require.NotContains(t, capability.Flags, FlagContract{Name: "xml"})
	require.NotContains(t, capability.Flags, FlagContract{Name: "with-stats"})

	doc := capabilityDocumentForRoot(root)
	paths := commandCapabilityPaths(doc.Commands)
	require.Contains(t, paths, "ops purge all-process-definitions")
	require.NotContains(t, paths, "ops purge purge-definitions")
	require.NotContains(t, paths, "ops purge delete-all")
}

// TestCommandCapabilityForCommand_OpsPurgeProcessInstancesWithIncidentsContract verifies discovery metadata for the incident purge command.
func TestCommandCapabilityForCommand_OpsPurgeProcessInstancesWithIncidentsContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	capability := commandCapabilityForCommand(opsPurgeProcessInstancesWithIncidentsCmd)

	require.Equal(t, "ops purge process-instances-with-incidents", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed incident-based purges")
	require.Contains(t, capability.Aliases, "pi-with-incidents")
	require.Contains(t, capability.Aliases, "piwi")
	require.NotContains(t, capability.Aliases, "incident-pis")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "one-line",
		Supported: true,
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "inc-key",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to select for candidate discovery",
	})
	require.False(t, hasFlagContractNamed(capability.Flags, "key"))
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-instance-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key to filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "discover and validate incident-based process-instance cleanup without submitting deletion requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-format",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "audit report format: markdown, json (default inferred from report-file extension)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
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
		Name:        "force",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "allow cancellation when deletion encounters nonterminal process instances",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after deletion requests are accepted without deletion confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers; omit for an automatic limit",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling validation or deletion work after the first error",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-worker-limit",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "use all queued jobs as workers when --workers is unset",
	})
	require.NotContains(t, capability.Flags, FlagContract{Name: "pi-keys-only"})
	require.NotContains(t, capability.Flags, FlagContract{Name: "total"})
	require.False(t, hasFlagContractNamed(capability.Flags, "flow-node-id"))
	require.False(t, hasFlagContractNamed(capability.Flags, "fni-key"))
}

func TestCommandCapabilityForCommand_OpsExecuteRetentionPolicyContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsExecuteRetentionPolicyCmd)

	require.Equal(t, "ops execute retention-policy", capability.Path)
	require.Contains(t, capability.Aliases, "ret-pol")
	require.Contains(t, capability.Aliases, "rp")
	require.NotContains(t, capability.Aliases, "rt")
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed retention cleanup")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "retention-days",
		Type:        "int",
		Required:    true,
		Repeated:    false,
		Description: "required non-negative age in days for process-instance retention eligibility",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "discover and validate retention cleanup without submitting deletion requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "write an audit report to the given path",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-format",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "audit report format: markdown, json (default inferred from report-file extension)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to filter process instances",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers; omit for an automatic limit",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after deletion requests are accepted without deletion confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "force",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "allow cancellation when deletion encounters nonterminal process instances",
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

// TestCommandCapabilityForCommand_OpsExecuteSmokeTestContract verifies discoverable smoke-test metadata stays automation-safe and deployment-wide.
func TestCommandCapabilityForCommand_OpsExecuteSmokeTestContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	capability := commandCapabilityForCommand(opsExecuteSmokeTestCmd)

	require.Equal(t, "ops execute smoke-test", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed smoke-test cleanup")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.NotContains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "count",
		Shorthand:   "n",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "number of process instances to create from the deployed smoke-test definition",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers; omit for an automatic limit",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-worker-limit",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "use all queued smoke-test jobs as workers when --workers is unset",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-cleanup",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "retain created process instances and all deployed fixture process definitions",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling smoke-test work after the first error",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "validate the smoke-test plan without submitting mutation requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after cleanup requests are accepted without deletion confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "write an audit report to the given path",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-format",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "audit report format: markdown, json (default inferred from report-file extension)",
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

func TestCapabilityDocumentForRoot_OpsExecuteSmokeTestDiscovery(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	doc := capabilityDocumentForRoot(root)

	opsCapability, ok := findCommandCapability(doc.Commands, "ops")
	require.True(t, ok)
	executeCapability, ok := findCommandCapability(opsCapability.Children, "ops execute")
	require.True(t, ok)
	smokeTestCapability, ok := findCommandCapability(executeCapability.Children, "ops execute smoke-test")
	require.True(t, ok)
	require.Equal(t, "Execute a cluster smoke test workflow", smokeTestCapability.Summary)
	require.Equal(t, CommandMutationStateChanging, smokeTestCapability.Mutation)
	require.Equal(t, ContractSupportFull, smokeTestCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, smokeTestCapability.AutomationSupport)
	require.Contains(t, smokeTestCapability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, smokeTestCapability.Flags, FlagContract{
		Name:        "report-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "write an audit report to the given path",
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
		Description: "maximum concurrent workers; omit for an automatic limit",
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

// TestCommandCapabilityForCommand_ResolveProcessInstanceContract preserves keyed metadata and advertises scoped BID controls.
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
		Description: "maximum concurrent workers; omit for an automatic limit",
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
	for _, name := range []string{"bid", "limit", "batch-size"} {
		require.NotNil(t, resolveProcessInstanceCmd.Flags().Lookup(name))
	}
	require.Equal(t, "string", resolveProcessInstanceCmd.Flags().Lookup("bid").Value.Type())
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

// TestGetJobAndUpdateJobHelp_DocumentsDiscoveryAndMutationGuards keeps job guidance
// discoverable alongside the parent group's user-task completion mode.
func TestGetJobAndUpdateJobHelp_DocumentsDiscoveryAndMutationGuards(t *testing.T) {
	assertCommandHelpOutput(t, []string{"get"}, []string{"Inspect cluster health", "process-instance", "incident"}, nil)
	assertCommandHelpOutput(t, []string{"get", "job"}, []string{"--retries", "--total", "c8volt get job --key"}, nil)
	assertCommandHelpOutput(t, []string{"update"}, []string{"--set-*", "job retries", "user-task"}, nil)
	assertCommandHelpOutput(t, []string{"update", "job"}, []string{"--set-retries", "--set-timeout", "--set-error-code", "--set-error-message", "--complete"}, nil)
}

// TestGetElementHelp_DocumentsSearchAndOutputModes keeps user-facing help in
// sync with the element CLI contract and generated documentation source.
func TestGetElementHelp_DocumentsSearchAndOutputModes(t *testing.T) {
	assertCommandHelpOutput(t, []string{"get", "element"}, []string{"--pi-key", "--with-vars", "--with-listeners", "worker waiting time"}, nil)
}

func TestGetIncidentHelp_DocumentsAliasesPipelinesAndInheritedOutputModes(t *testing.T) {
	assertCommandHelpOutput(t, []string{"get", "incident"}, []string{"Search defaults to active incidents", "--pi-keys-only", "--error-type"}, nil)
}

func TestIncidentCommandHelpOmitsLegacyElementTerminology(t *testing.T) {
	tests := [][]string{
		{"get", "incident"},
		{"ops", "repair", "incident"},
		{"ops", "purge", "process-instances-with-incidents"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			output := assertCommandHelpOutput(t, args, []string{
				"--element-id string",
				"--element-instance-key string",
			}, []string{
				"--flow-node-id",
				"--fni-key",
			})
			require.NotContains(t, output, "flow node")
		})
	}
}

// TestUpdateProcessInstanceHelp_DocumentsVariableUpdateDiscovery preserves root-variable
// guidance when the parent also advertises element-local updates and all task mutations.
func TestUpdateProcessInstanceHelp_DocumentsVariableUpdateDiscovery(t *testing.T) {
	assertCommandHelpOutput(t, []string{"update", "process-instance"}, []string{"root-scope variables", "--set-vars-file", "--set-bid", "--dry-run"}, nil)
}

func TestProcessInstanceSelectorValidationHelpContract(t *testing.T) {
	assertCommandHelpOutput(t, []string{"get", "process-instance"}, []string{"--with-incidents", "--with-elements", "--has-user-tasks", "--var stringArray", "--var-exists stringArray", "--var-like stringArray", "--total", "--limit"}, nil)
	assertCommandHelpOutput(t, []string{"cancel", "process-instance"}, []string{"affected instance families", "Empty searches finish without confirmation or mutation", "--force", "--dry-run"}, nil)
	assertCommandHelpOutput(t, []string{"delete", "process-instance"}, []string{"children before parents", "--force", "--dry-run"}, nil)
	assertCommandHelpOutput(t, []string{"run", "process-instance"}, []string{"--set-vars", "--set-bid", "--pd-key", "--count"}, nil)
}

// TestProcessDefinitionSelectorValidationHelpContract keeps process-definition
// help text aligned with selector validation and watch execution limits.
func TestProcessDefinitionSelectorValidationHelpContract(t *testing.T) {
	assertCommandHelpOutput(t, []string{"get", "process-definition"}, []string{"--latest", "--with-stats", "--watch", "--xml"}, nil)
	assertCommandHelpOutput(t, []string{"delete", "process-definition"}, []string{"Unfinished instances block deletion", "--force", "--dry-run", "--tenant"}, nil)
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

// flattenCommandCapabilities gives contract tests a simple view of nested command metadata.
func flattenCommandCapabilities(commands []CommandCapability) []CommandCapability {
	var out []CommandCapability
	for _, command := range commands {
		out = append(out, command)
		out = append(out, flattenCommandCapabilities(command.Children)...)
	}
	return out
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

func hasFlagContractNamed(flags []FlagContract, name string) bool {
	for _, flag := range flags {
		if flag.Name == name {
			return true
		}
	}
	return false
}

func readCLIDebtAssessmentCommandPaths(t *testing.T) []string {
	t.Helper()

	path := filepath.Join("..", "specs", "254-cli-debt-refactor", "assessment.md")
	body, err := os.ReadFile(path)
	require.NoError(t, err)

	const header = "## Command Node Assessment"
	inSection := false
	var paths []string
	for _, line := range strings.Split(string(body), "\n") {
		switch {
		case line == header:
			inSection = true
			continue
		case inSection && strings.HasPrefix(line, "## "):
			return paths
		case !inSection || !strings.HasPrefix(line, "| `"):
			continue
		}

		cells := strings.Split(line, "|")
		require.Lenf(t, cells, 17, "assessment row should keep 15 required cells: %s", line)
		for _, cell := range cells[1 : len(cells)-1] {
			value := strings.TrimSpace(cell)
			require.NotEmptyf(t, value, "assessment cell must be populated: %s", line)
			require.NotContainsf(t, strings.ToLower(value), "todo", "assessment cell must not be placeholder: %s", line)
			require.NotContainsf(t, strings.ToLower(value), "tbd", "assessment cell must not be placeholder: %s", line)
		}

		commandPath := strings.Trim(strings.TrimSpace(cells[1]), "`")
		require.NotContainsf(t, paths, commandPath, "duplicate assessment command path %q", commandPath)
		paths = append(paths, commandPath)
	}

	return paths
}

// readGoTopLevelDeclarations returns names declared at package scope in a Go source file.
func readGoTopLevelDeclarations(t *testing.T, path string) map[string]struct{} {
	t.Helper()

	declarations, err := readGoTopLevelDeclarationsFromFile(path)
	require.NoError(t, err)
	return declarations
}

// readGoTopLevelDeclarationsIfExists lets source-structure tests distinguish an
// unsplit baseline from a focused file that now has to own the listed declarations.
func readGoTopLevelDeclarationsIfExists(t *testing.T, path string) (map[string]struct{}, bool) {
	t.Helper()

	declarations, err := readGoTopLevelDeclarationsFromFile(path)
	if os.IsNotExist(err) {
		return nil, false
	}
	require.NoError(t, err)
	return declarations, true
}

// readGoTopLevelDeclarationsFromFile parses package-scope names without loading
// or type-checking the command package.
func readGoTopLevelDeclarationsFromFile(path string) (map[string]struct{}, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	declarations := make(map[string]struct{})
	for _, declaration := range file.Decls {
		switch decl := declaration.(type) {
		case *ast.FuncDecl:
			declarations[decl.Name.Name] = struct{}{}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch typed := spec.(type) {
				case *ast.TypeSpec:
					declarations[typed.Name.Name] = struct{}{}
				case *ast.ValueSpec:
					for _, name := range typed.Names {
						declarations[name.Name] = struct{}{}
					}
				}
			}
		}
	}
	return declarations, nil
}

// TestCommandContractBulkCompletionSelectors advertises alternative selection without adding keys-only output.
func TestCommandContractBulkCompletionSelectors(t *testing.T) {
	capability := commandCapabilityForCommand(updateJobCmd)
	for _, name := range []string{"state", "type", "pi-key", "element-instance-key", "element-id", "worker", "retries", "kind", "listener-event-type", "batch-size", "limit", "workers", "fail-fast", "no-worker-limit"} {
		found := false
		for _, flag := range capability.Flags {
			if flag.Name == name {
				found = true
				require.False(t, flag.Required)
			}
		}
		require.True(t, found, name)
	}
}

// TestSuspendProcessInstanceMetadata keeps selection flags and explicit
// mutation/output/automation capabilities aligned with generated documentation.
func TestSuspendProcessInstanceMetadata(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	cap := commandCapabilityForCommand(suspendProcessInstanceCmd)
	require.Equal(t, "suspend process-instance", cap.Path)
	require.Equal(t, CommandMutationStateChanging, cap.Mutation)
	require.Equal(t, ContractSupportFull, cap.ContractSupport)
	require.Equal(t, AutomationSupportFull, cap.AutomationSupport)
	require.Contains(t, cap.Aliases, "pi")
	require.Equal(t, []OutputModeContract{
		{Name: RenderModeOneLine.String(), Supported: true},
		{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		{Name: RenderModeKeysOnly.String(), Supported: true},
	}, cap.OutputModes)
	for _, name := range []string{"key", "bid", "batch-size", "limit", "tree", "dry-run", "no-wait", "workers", "no-worker-limit", "fail-fast"} {
		require.NotNil(t, suspendProcessInstanceCmd.Flags().Lookup(name))
	}
	treeFlag := suspendProcessInstanceCmd.Flags().Lookup("tree")
	require.Equal(t, "false", treeFlag.DefValue)
	require.Equal(t, "bool", treeFlag.Value.Type())
	require.Empty(t, treeFlag.Shorthand)

	for _, name := range []string{"force", "no-state-check", "state", "bpmn-process-id"} {
		require.Nil(t, suspendProcessInstanceCmd.Flags().Lookup(name))
	}
	for _, name := range []string{"backoff-timeout", "backoff-max-retries"} {
		flag := suspendCmd.PersistentFlags().Lookup(name)
		require.NotNil(t, flag)
		require.True(t, flag.Hidden)
	}
	require.Equal(t, "2m0s", suspendCmd.PersistentFlags().Lookup("backoff-timeout").DefValue)

	for _, path := range []string{"suspend", "suspend process-instance"} {
		_, ok := findCommandCapability(capabilityDocumentForRoot(root).Commands, path)
		require.True(t, ok, path)
	}
}

// TestResumeProcessInstanceMetadata protects key/BID/tree selection and operator outcomes.
func TestResumeProcessInstanceMetadata(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	cap := commandCapabilityForCommand(resumeProcessInstanceCmd)
	require.Equal(t, "resume process-instance", cap.Path)
	require.Equal(t, CommandMutationStateChanging, cap.Mutation)
	require.Equal(t, ContractSupportFull, cap.ContractSupport)
	require.Equal(t, AutomationSupportFull, cap.AutomationSupport)
	require.Contains(t, cap.Aliases, "pi")
	require.Equal(t, []OutputModeContract{
		{Name: RenderModeOneLine.String(), Supported: true},
		{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		{Name: RenderModeKeysOnly.String(), Supported: true},
	}, cap.OutputModes)
	for _, name := range []string{"key", "bid", "batch-size", "limit", "tree", "dry-run", "no-wait", "workers", "no-worker-limit", "fail-fast"} {
		require.NotNil(t, resumeProcessInstanceCmd.Flags().Lookup(name))
	}
	for _, name := range []string{"force", "no-state-check", "state", "bpmn-process-id"} {
		require.Nil(t, resumeProcessInstanceCmd.Flags().Lookup(name))
	}
	for _, name := range []string{"backoff-timeout", "backoff-max-retries"} {
		flag := resumeCmd.PersistentFlags().Lookup(name)
		require.NotNil(t, flag)
		require.True(t, flag.Hidden)
	}
	require.Equal(t, "2m0s", resumeCmd.PersistentFlags().Lookup("backoff-timeout").DefValue)

	treeFlag := resumeProcessInstanceCmd.Flags().Lookup("tree")
	require.Equal(t, "false", treeFlag.DefValue)
	require.Equal(t, "bool", treeFlag.Value.Type())
	require.Empty(t, treeFlag.Shorthand)

	for _, path := range []string{"resume", "resume process-instance"} {
		_, ok := findCommandCapability(capabilityDocumentForRoot(root).Commands, path)
		require.True(t, ok, path)
	}
}

// TestCapabilityDocumentVersionCheck exposes the persistent invocation-only opt-out.
func TestCapabilityDocumentVersionCheck(t *testing.T) {
	resetCommandTreeFlags(Root())
	for _, command := range []*cobra.Command{Root(), getProcessDefinitionCmd, configTestConnectionCmd} {
		capability := commandCapabilityForCommand(command)
		require.Contains(t, capability.Flags, FlagContract{Name: "no-version-check", Type: "bool", Description: "skip the automatic advisory gateway version check"})
	}
	require.Equal(t, "false", Root().PersistentFlags().Lookup("no-version-check").DefValue)
}

// TestCommandContractFlagIntentSurface covers all rename rows while retaining
// job selector retries, HTTP timeout, action arity and unchanged enrichment controls.
func TestCommandContractFlagIntentSurface(t *testing.T) {
	for _, tc := range flagIntentSurface {
		for old, canonical := range tc.names {
			require.NotNil(t, tc.cmd.Flags().Lookup(canonical), tc.cmd.CommandPath())
			require.Contains(t, tc.cmd.Flags().FlagUsages(), "--"+canonical+" ")
			if tc.cmd == updateJobCmd && (old == "retries" || old == "timeout") {
				continue
			}
			require.Nil(t, tc.cmd.Flags().Lookup(old), tc.cmd.CommandPath())
			require.NotContains(t, tc.cmd.Flags().FlagUsages(), "--"+old+" ")
			require.Equal(t, "--"+canonical, flagMigrationsForCommand(tc.cmd)[old].Replacement)
		}
	}
	require.Equal(t, "bool", updateJobCmd.Flags().Lookup("throw-bpmn-error").Value.Type())
	require.Equal(t, "string", updateJobCmd.Flags().Lookup("set-error-code").Value.Type())
	require.NotNil(t, updateJobCmd.Flags().Lookup("retries"))
	require.Nil(t, updateJobCmd.LocalNonPersistentFlags().Lookup("timeout"))
	require.NotNil(t, rootCmd.PersistentFlags().Lookup("timeout"))
	require.NotNil(t, getJobCmd.Flags().Lookup("retries"))
	require.NotNil(t, getProcessInstanceCmd.Flags().Lookup("bid"))
	for _, name := range []string{"with-incidents", "with-vars", "with-elements", "with-listeners"} {
		require.NotNil(t, getProcessInstanceCmd.Flags().Lookup(name))
	}
	require.NotNil(t, getUserTaskCmd.Flags().Lookup("with-vars"))
	require.NotNil(t, getClusterVersionCmd.Flags().Lookup("with-brokers"))
	require.NotNil(t, opsAnalyseSlowProcessInstancesCmd.Flags().Lookup("with-full-timeline"))
	require.True(t, rootCmd.PersistentFlags().Lookup("log-with-source").Hidden)
}
