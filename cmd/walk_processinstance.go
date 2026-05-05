// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagWalkPIKey           string
	flagWalkPIModeParent    bool
	flagWalkPIModeChildren  bool
	flagWalkPIFlat          bool
	flagWalkPIWithIncidents bool
)

const (
	walkPIModeParent   = "parent"
	walkPIModeChildren = "children"
	walkPIModeFamily   = "family"
)

var walkProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Inspect the parent/child tree of process instances",
	Long: "Inspect the parent/child tree of process instances.\n\n" +
		"By default, walk shows the full process-instance family as an ASCII tree. Use --parent for ancestry, --children for descendants, or --flat for a path-style family view.\n\n" +
		"Add --with-incidents to keyed walks to show incident keys and messages below matching process-instance rows.\n\n" +
		"When an ancestor is missing but reachable family data still exists, walk returns the partial tree plus a warning. Direct single-resource lookups stay strict.",
	Example: `  ./c8volt walk pi --key 2251799813711967
  ./c8volt walk pi --key 2251799813711967 --with-incidents
  ./c8volt walk pi --key 2251799813711967 --with-incidents --incident-message-limit 80
  ./c8volt walk pi --key 2251799813711967 --flat
  ./c8volt walk pi --key 2251799813711977 --parent
  ./c8volt --json walk pi --key 2251799813711967 --children --with-incidents`,
	Aliases: []string{"pi", "pis"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateWalkPIWithIncidentsUsage(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		type walker struct {
			fetch        func() (process.TraversalResult, error)
			view         func(*cobra.Command, process.TraversalResult) error
			enrichedView func(*cobra.Command, process.IncidentEnrichedTraversalResult) error
		}

		walkers := map[string]walker{
			walkPIModeParent: {
				fetch: func() (process.TraversalResult, error) {
					return cli.AncestryResult(cmd.Context(), flagWalkPIKey, collectOptions()...)
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
					return cli.DescendantsResult(cmd.Context(), flagWalkPIKey, collectOptions()...)
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
					return cli.FamilyResult(cmd.Context(), flagWalkPIKey, collectOptions()...)
				},
				view: func(cmd *cobra.Command, result process.TraversalResult) error {
					if pickMode() == RenderModeJSON {
						return renderJSONPayload(cmd, RenderModeJSON, traversalPayload(result))
					}
					if !flagWalkPIFlat {
						if len(result.Keys) == 0 {
							return nil
						}
						if err := renderFamilyTree(cmd, result.RootKey, result.Edges, result.Chain, flagWalkPIKey); err != nil {
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
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if flagWalkPIWithIncidents {
			enriched, err := cli.EnrichTraversalWithIncidents(cmd.Context(), result, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
			switch pickMode() {
			case RenderModeJSON:
				if err := renderJSONPayload(cmd, RenderModeJSON, incidentEnrichedTraversalPayload(enriched)); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
				}
				return
			case RenderModeOneLine:
				if err := w.enrichedView(cmd, enriched); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
				}
				return
			}
		}
		if err := w.view(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
	},
}

func init() {
	walkCmd.AddCommand(walkProcessInstanceCmd)

	fs := walkProcessInstanceCmd.Flags()
	fs.StringVarP(&flagWalkPIKey, "key", "k", "", "start walking from this process instance key")
	_ = walkProcessInstanceCmd.MarkFlagRequired("key")

	fs.BoolVar(&flagWalkPIModeParent, "parent", false, "show ancestry from the selected process instance toward the root")
	fs.BoolVar(&flagWalkPIModeChildren, "children", false, "show descendants from the selected process instance")
	fs.BoolVar(&flagWalkPIFlat, "flat", false, "render family output as a flat path instead of an ASCII tree")
	fs.BoolVar(&flagWalkPIWithIncidents, "with-incidents", false, "show incident keys and messages for keyed process-instance walks")
	fs.IntVar(&flagGetPIIncidentMessageLimit, "incident-message-limit", 0, "maximum characters to show for human incident messages when --with-incidents is set; 0 disables truncation")

	setCommandMutation(walkProcessInstanceCmd, CommandMutationReadOnly)
	setContractSupport(walkProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(walkProcessInstanceCmd, AutomationSupportUnsupported, "automation mode is not supported for traversal commands")
}

// validateWalkPIWithIncidentsUsage keeps walk incident enrichment in render modes that can show incident detail rows.
func validateWalkPIWithIncidentsUsage(cmd *cobra.Command) error {
	if flagGetPIIncidentMessageLimit < 0 {
		return invalidFlagValuef("invalid value for --incident-message-limit: %d, expected non-negative integer", flagGetPIIncidentMessageLimit)
	}
	if isPIIncidentMessageLimitFlagChanged(cmd) && !flagWalkPIWithIncidents {
		return missingDependentFlagsf("--incident-message-limit requires --with-incidents")
	}
	if !flagWalkPIWithIncidents {
		return nil
	}
	if strings.TrimSpace(flagWalkPIKey) == "" {
		return invalidFlagValuef("--with-incidents requires --key")
	}
	if flagViewKeysOnly {
		return mutuallyExclusiveFlagsf("--with-incidents cannot be combined with --keys-only")
	}
	return nil
}
