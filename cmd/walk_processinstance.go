// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagWalkPIKey           string
	flagWalkPIModeParent    bool
	flagWalkPIModeChildren  bool
	flagWalkPIFlat          bool
	flagWalkPIWithIncidents bool
	flagWalkPIWithVars      bool
	flagWalkPIWithElements  bool
	flagWalkPIWithListeners bool
)

const (
	walkPIModeParent   = "parent"
	walkPIModeChildren = "children"
	walkPIModeFamily   = "family"
)

var walkProcessInstanceCmd = &cobra.Command{
	ValidArgsFunction: completeWalkPISelector,
	Use:               "process-instance",
	Short:             "Inspect the parent/child tree of process instances",
	Long: `Inspect the process-instance family around a key.

Use --parent for ancestry or --children for descendants; the default includes
both. Add --with-incidents, --with-vars, or --with-elements for details.
Add --with-listeners to --with-elements for listener jobs.

Alternatively, select by exact --bid (only c8.10+). It must identify one instance;
multiple matches are ambiguous.`,
	Example: `  c8volt walk process-instance --key <process-instance-key>
  c8volt walk process-instance --key <process-instance-key> --with-incidents
  c8volt walk process-instance --key <process-instance-key> --with-elements --with-listeners`,
	Aliases: []string{"pi", "pis"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateWalkPISelection(cmd); err != nil {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateWalkPIWithIncidentsUsage(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateWalkPIWithVarsUsage(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateWalkPIWithListenersUsage(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateWalkPIWithElementsUsage(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		walkKey, selection, err := resolveWalkPIStart(cmd, cli)
		if err != nil {
			handleBusinessIDWalkError(cmd, log, cfg.App.NoErrCodes, err, selection, len(selection.Keys))
		}
		if walkKey == "" {
			if err := emptyWalkSelectionView(cmd); err != nil {
				handleWalkRenderError(cmd, log, cfg.App.NoErrCodes, err)
			}
			return
		}
		type walker struct {
			fetch        func() (process.TraversalResult, error)
			view         func(*cobra.Command, process.TraversalResult) error
			enrichedView func(*cobra.Command, process.IncidentEnrichedTraversalResult) error
		}

		walkers := map[string]walker{
			walkPIModeParent: {
				fetch: func() (process.TraversalResult, error) {
					return cli.AncestryResult(cmd.Context(), walkKey, appendExplicitLargeWorkProgressOption(cmd, collectExplicitPIAdminInputOptions())...)
				},
				view: func(cmd *cobra.Command, result process.TraversalResult) error {
					if pickMode() == RenderModeJSON {
						return renderJSONPayload(cmd, RenderModeJSON, traversalPayload(result))
					}
					if err := ancestorsView(cmd, result.Keys, result.Chain); err != nil {
						return err
					}
					printTraversalWarning(cmd, result)
					return nil
				},
				enrichedView: func(cmd *cobra.Command, result process.IncidentEnrichedTraversalResult) error {
					if err := incidentEnrichedAncestorsView(cmd, result); err != nil {
						return err
					}
					printIncidentEnrichedTraversalWarning(cmd, result)
					return nil
				},
			},
			walkPIModeChildren: {
				fetch: func() (process.TraversalResult, error) {
					return cli.DescendantsResult(cmd.Context(), walkKey, appendExplicitLargeWorkProgressOption(cmd, collectExplicitPIAdminInputOptions())...)
				},
				view: func(cmd *cobra.Command, result process.TraversalResult) error {
					if pickMode() == RenderModeJSON {
						return renderJSONPayload(cmd, RenderModeJSON, traversalPayload(result))
					}
					return descendantsView(cmd, result.Keys, result.Chain)
				},
				enrichedView: func(cmd *cobra.Command, result process.IncidentEnrichedTraversalResult) error {
					return incidentEnrichedDescendantsView(cmd, result)
				},
			},
			walkPIModeFamily: {
				fetch: func() (process.TraversalResult, error) {
					return cli.FamilyResult(cmd.Context(), walkKey, appendExplicitLargeWorkProgressOption(cmd, collectExplicitPIAdminInputOptions())...)
				},
				view: func(cmd *cobra.Command, result process.TraversalResult) error {
					if pickMode() == RenderModeJSON {
						return renderJSONPayload(cmd, RenderModeJSON, traversalPayload(result))
					}
					if !flagWalkPIFlat && pickMode() != RenderModeKeysOnly {
						if len(result.Keys) == 0 {
							return nil
						}
						if err := renderFamilyTree(cmd, result.RootKey, result.Edges, result.Chain, walkKey); err != nil {
							return err
						}
						printTraversalWarning(cmd, result)
						return nil
					}
					if err := familyView(cmd, result.Keys, result.Chain); err != nil {
						return err
					}
					printTraversalWarning(cmd, result)
					return nil
				},
				enrichedView: func(cmd *cobra.Command, result process.IncidentEnrichedTraversalResult) error {
					if err := incidentEnrichedFamilyView(cmd, result); err != nil {
						return err
					}
					printIncidentEnrichedTraversalWarning(cmd, result)
					return nil
				},
			},
		}
		selectedMode := walkPIModeFamily
		switch {
		case flagWalkPIModeParent:
			selectedMode = walkPIModeParent
		case flagWalkPIModeChildren:
			selectedMode = walkPIModeChildren
		}
		w := walkers[selectedMode]
		result, err := w.fetch()
		if err != nil {
			handleBusinessIDWalkError(cmd, log, cfg.App.NoErrCodes, err, traversalPayload(result), len(selection.Keys))
		}
		if flagWalkPIWithIncidents || flagWalkPIWithVars || flagWalkPIWithElements {
			adminInputOpts := appendExplicitLargeWorkProgressOption(cmd, collectExplicitPIAdminInputOptions())
			adminInputIncidentOpts := appendExplicitLargeWorkProgressOption(cmd, append(collectIncidentEnrichmentOptions(), options.WithIgnoreTenant()))
			walkedInstances := processInstancesFromTraversal(result)
			var incidentEnriched process.IncidentEnrichedTraversalResult
			if flagWalkPIWithIncidents {
				incidentEnriched, err = cli.EnrichTraversalWithIncidents(cmd.Context(), result, adminInputIncidentOpts...)
				if err != nil {
					handleBusinessIDWalkError(cmd, log, cfg.App.NoErrCodes, err, traversalPayload(result), len(selection.Keys))
				}
			}
			var variableEnriched process.VariableEnrichedProcessInstances
			if flagWalkPIWithVars {
				variableEnriched, err = cli.EnrichProcessInstancesWithVariables(cmd.Context(), walkedInstances, adminInputOpts...)
				if err != nil {
					handleBusinessIDWalkError(cmd, log, cfg.App.NoErrCodes, err, traversalPayload(result), len(selection.Keys))
				}
			}
			var elementEnriched process.ElementEnrichedProcessInstances
			if flagWalkPIWithElements {
				if flagWalkPIWithListeners {
					elementEnriched, err = enrichProcessInstancesWithElementListenerActivityOptions(cmd, cli, walkedInstances, adminInputOpts)
				} else {
					elementEnriched, err = enrichProcessInstancesWithElementActivityOptions(cmd, cli, walkedInstances, adminInputOpts)
				}
				if err != nil {
					handleBusinessIDWalkError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("walk process instance elements: %w", err), traversalPayload(result), len(selection.Keys))
				}
			}
			activityItems := activityItemsFromTraversal(result, incidentEnriched, variableEnriched, elementEnriched, flagWalkPIWithIncidents)
			if err != nil {
				handleBusinessIDWalkError(cmd, log, cfg.App.NoErrCodes, err, traversalPayload(result), len(selection.Keys))
			}
			switch pickMode() {
			case RenderModeJSON:
				if flagWalkPIWithIncidents && !flagWalkPIWithVars && !flagWalkPIWithElements {
					if err := renderJSONPayload(cmd, RenderModeJSON, incidentEnrichedTraversalPayload(incidentEnriched)); err != nil {
						handleWalkRenderError(cmd, log, cfg.App.NoErrCodes, err)
					}
					return
				}
				if err := renderJSONPayload(cmd, RenderModeJSON, activityTraversalPayload(result, activityItems)); err != nil {
					handleWalkRenderError(cmd, log, cfg.App.NoErrCodes, err)
				}
				return
			case RenderModeOneLine:
				if flagWalkPIWithIncidents && !flagWalkPIWithVars && !flagWalkPIWithElements {
					if err := w.enrichedView(cmd, incidentEnriched); err != nil {
						handleWalkRenderError(cmd, log, cfg.App.NoErrCodes, err)
					}
					return
				}
				if err := walkActivityView(cmd, selectedMode, result, activityItems); err != nil {
					handleWalkRenderError(cmd, log, cfg.App.NoErrCodes, err)
				}
				return
			}
		}
		if err := w.view(cmd, result); err != nil {
			handleWalkRenderError(cmd, log, cfg.App.NoErrCodes, err)
		}
	},
}

// init registers one starting selector and the existing traversal/enrichment controls.
func init() {
	walkCmd.AddCommand(walkProcessInstanceCmd)

	fs := walkProcessInstanceCmd.Flags()
	fs.StringVarP(&flagWalkPIKey, "key", "k", "", "start walking from this process instance key")
	fs.String("bid", "", "exact business ID for one unambiguous starting instance (only c8.10+)")

	fs.BoolVar(&flagWalkPIModeParent, "parent", false, "show ancestry from the selected process instance toward the root")
	fs.BoolVar(&flagWalkPIModeChildren, "children", false, "show descendants from the selected process instance")
	fs.BoolVar(&flagWalkPIFlat, "flat", false, "render family output as a flat path instead of an ASCII tree")
	fs.BoolVar(&flagWalkPIWithIncidents, "with-incidents", false, "show incident keys, states, and messages for selected process-instance walks")
	fs.StringVar(&flagGetPIIncidentState, "incident-state", "active", "incident state scope for --with-incidents: active, pending, resolved, migrated, unknown, all")
	fs.IntVar(&flagGetPIIncidentMessageLimit, "incident-message-limit", 0, "maximum characters to show for incident messages when --with-incidents is set; 0 disables truncation")
	fs.BoolVar(&flagWalkPIWithVars, "with-vars", false, "show process-instance-scope variables for selected process-instance walks")
	fs.IntVar(&flagGetPIVarValueLimit, "var-value-limit", 0, "maximum characters to show for variable values when --with-vars is set; 0 disables truncation")
	fs.BoolVar(&flagWalkPIWithElements, "with-elements", false, "show runtime element instances for selected process-instance walks")
	fs.BoolVar(&flagWalkPIWithListeners, "with-listeners", false, "include runtime listener jobs; requires --with-elements")

	setCommandMutation(walkProcessInstanceCmd, CommandMutationReadOnly)
	setContractSupport(walkProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(walkProcessInstanceCmd, AutomationSupportUnsupported, "automation mode is not supported for traversal commands")
}

// validateWalkPIWithVarsUsage keeps walk variable enrichment in render modes that can show variable detail rows.
func validateWalkPIWithVarsUsage(cmd *cobra.Command) error {
	if flagGetPIVarValueLimit < 0 {
		return invalidFlagValuef("invalid value for --var-value-limit: %d, expected non-negative integer", flagGetPIVarValueLimit)
	}
	if isPIVarValueLimitFlagChanged(cmd) && !flagWalkPIWithVars {
		return missingDependentFlagsf("--var-value-limit requires --with-vars")
	}
	if !flagWalkPIWithVars {
		return nil
	}
	if strings.TrimSpace(flagWalkPIKey) == "" && searchBusinessIDFlag(cmd) == nil {
		return invalidFlagValuef("--with-vars requires --key or --bid")
	}
	if flagViewKeysOnly {
		return mutuallyExclusiveFlagsf("--with-vars cannot be combined with --keys-only")
	}
	return nil
}

// validateWalkPIWithIncidentsUsage keeps walk incident enrichment in render modes that can show incident detail rows.
func validateWalkPIWithIncidentsUsage(cmd *cobra.Command) error {
	if flagGetPIIncidentMessageLimit < 0 {
		return invalidFlagValuef("invalid value for --incident-message-limit: %d, expected non-negative integer", flagGetPIIncidentMessageLimit)
	}
	if err := validatePIIncidentStateFlag(flagGetPIIncidentState); err != nil {
		return err
	}
	if isPIIncidentMessageLimitFlagChanged(cmd) && !flagWalkPIWithIncidents {
		return missingDependentFlagsf("--incident-message-limit requires --with-incidents")
	}
	if isPIIncidentStateFlagChanged(cmd) && !flagWalkPIWithIncidents {
		return missingDependentFlagsf("--incident-state requires --with-incidents")
	}
	if !flagWalkPIWithIncidents {
		return nil
	}
	if strings.TrimSpace(flagWalkPIKey) == "" && searchBusinessIDFlag(cmd) == nil {
		return invalidFlagValuef("--with-incidents requires --key or --bid")
	}
	if flagViewKeysOnly {
		return mutuallyExclusiveFlagsf("--with-incidents cannot be combined with --keys-only")
	}
	return nil
}

// validateWalkPIWithListenersUsage rejects listener enrichment when walk output
// lacks the element section needed to represent nested listener detail rows.
func validateWalkPIWithListenersUsage(cmd *cobra.Command) error {
	if !flagWalkPIWithListeners {
		return nil
	}
	if !flagWalkPIWithElements {
		return missingDependentFlagsf("--with-listeners requires --with-elements")
	}
	if flagViewKeysOnly {
		return mutuallyExclusiveFlagsf("--with-listeners cannot be combined with --keys-only")
	}
	return nil
}

// validateWalkPIWithElementsUsage keeps runtime element enrichment in keyed
// render modes that can carry element detail rows or JSON arrays.
func validateWalkPIWithElementsUsage(cmd *cobra.Command) error {
	if !flagWalkPIWithElements {
		return nil
	}
	if strings.TrimSpace(flagWalkPIKey) == "" && searchBusinessIDFlag(cmd) == nil {
		return invalidFlagValuef("--with-elements requires --key or --bid")
	}
	if flagViewKeysOnly {
		return mutuallyExclusiveFlagsf("--with-elements cannot be combined with --keys-only")
	}
	return nil
}
