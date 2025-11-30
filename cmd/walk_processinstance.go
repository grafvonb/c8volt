package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
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
	Use:     "process-instance",
	Short:   "Traverse (walk) the parent/child graph of process instances",
	Aliases: []string{"pi", "pis"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}

		if flagViewAsTree && (!flagWalkPIModeFamily && flagWalkPIMode != walkPIModeFamily) {
			flagWalkPIModeFamily = true
			flagWalkPIModeChildren = false
			flagWalkPIModeParent = false
		}

		type walker struct {
			fetch func() (KeysPath, Chain, error)
			view  func(*cobra.Command, KeysPath, Chain) error
		}
		var familyEdges map[string][]string

		walkers := map[string]walker{
			walkPIModeParent: {
				fetch: func() (KeysPath, Chain, error) {
					_, path, chain, err := cli.Ancestry(cmd.Context(), flagWalkPIKey, collectOptions()...)
					return path, chain, err
				},
				view: ancestorsView,
			},
			walkPIModeChildren: {
				fetch: func() (KeysPath, Chain, error) {
					path, _, chain, err := cli.Descendants(cmd.Context(), flagWalkPIKey, collectOptions()...)
					return path, chain, err
				},
				view: descendantsView,
			},
			walkPIModeFamily: {
				fetch: func() (KeysPath, Chain, error) {
					path, edges, chain, err := cli.Family(cmd.Context(), flagWalkPIKey, collectOptions()...)
					if err == nil {
						familyEdges = edges
					}
					return path, chain, err
				},
				view: func(cmd *cobra.Command, path KeysPath, chain Chain) error {
					mode := pickMode()
					if mode == RenderModeTree {
						if len(path) == 0 {
							return nil
						}
						rootKey := path[0]
						return renderFamilyTree(cmd, rootKey, familyEdges, chain, flagWalkPIKey)
					}
					return familyView(cmd, path, chain)
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
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("invalid --mode %q (must be %s, %s, or %s)", flagWalkPIMode, walkPIModeParent, walkPIModeChildren, walkPIModeFamily))
		}
		path, chain, err := w.fetch()
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		if err := w.view(cmd, path, chain); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
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
}
