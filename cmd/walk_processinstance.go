package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagWalkPIKey          string
	flagWalkPIMode         string
	flagWalkPIModeFamily   bool
	flagWalkPIModeParent   bool
	flagWalkPIModeChildren bool
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
		"Use this command when you need to understand ancestry or descendants before cancelling, deleting, or checking downstream effects. Choose --parent for ancestry, --children for descendants, and --family for the combined view. Add --tree with --family for an ASCII tree.\n\n" +
		"When an ancestor is missing but reachable family data still exists, walk returns the partial tree plus a warning. Direct single-resource lookups stay strict.",
	Example: `  ./c8volt walk pi --key 2251799813711967 --family
  ./c8volt walk pi --key 2251799813711967 --family --tree
  ./c8volt walk pi --key 2251799813711977 --parent
  ./c8volt --json walk pi --key 2251799813711967 --children`,
	Aliases: []string{"pi", "pis"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		if flagViewAsTree && (!flagWalkPIModeFamily && flagWalkPIMode != walkPIModeFamily) {
			flagWalkPIModeFamily = true
			flagWalkPIModeChildren = false
			flagWalkPIModeParent = false
		}

		type walker struct {
			fetch func() (process.TraversalResult, error)
			view  func(*cobra.Command, process.TraversalResult) error
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
			},
			walkPIModeFamily: {
				fetch: func() (process.TraversalResult, error) {
					return cli.FamilyResult(cmd.Context(), flagWalkPIKey, collectOptions()...)
				},
				view: func(cmd *cobra.Command, result process.TraversalResult) error {
					if pickMode() == RenderModeJSON {
						return renderJSONPayload(cmd, RenderModeJSON, traversalPayload(result))
					}
					mode := pickMode()
					if mode == RenderModeTree {
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
			},
		}
		switch {
		case flagWalkPIModeParent:
			flagWalkPIMode = walkPIModeParent
		case flagWalkPIModeChildren:
			flagWalkPIMode = walkPIModeChildren
		case flagWalkPIModeFamily:
			flagWalkPIMode = walkPIModeFamily
		}
		w, ok := walkers[flagWalkPIMode]
		if !ok {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, invalidFlagValuef("invalid --mode %q (must be %s, %s, or %s)", flagWalkPIMode, walkPIModeParent, walkPIModeChildren, walkPIModeFamily))
		}
		result, err := w.fetch()
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
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

	fs.StringVar(&flagWalkPIMode, "mode", walkPIModeChildren, "walk mode: parent, children, family")
	fs.BoolVar(&flagWalkPIModeParent, "parent", false, "shorthand for --mode=parent")
	fs.BoolVar(&flagWalkPIModeChildren, "children", false, "shorthand for --mode=children")
	fs.BoolVar(&flagWalkPIModeFamily, "family", false, "shorthand for --mode=family")
	fs.BoolVar(&flagViewAsTree, "tree", false, "render family mode as an ASCII tree (only valid with --family)")

	// shell completion for --mode
	_ = walkProcessInstanceCmd.RegisterFlagCompletionFunc("mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{walkPIModeParent, walkPIModeChildren, walkPIModeFamily}, cobra.ShellCompDirectiveNoFileComp
	})

	setCommandMutation(walkProcessInstanceCmd, CommandMutationReadOnly)
	setContractSupport(walkProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(walkProcessInstanceCmd, AutomationSupportUnsupported, "automation mode is not supported for traversal commands")
}
