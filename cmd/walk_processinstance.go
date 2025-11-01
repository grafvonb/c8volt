package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var (
	flagWalkKey  string
	flagWalkMode string
)

const (
	modeParent   = "parent"
	modeChildren = "children"
	modeFamily   = "family"
)

var walkProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Short:   "Traverse (walk) the parent/child graph of process instances",
	Aliases: []string{"pi", "pis"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, _, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}

		type walker struct {
			fetch func() (KeysPath, Chain, error)
			view  func(*cobra.Command, KeysPath, Chain) error
		}

		walkers := map[string]walker{
			modeParent: {
				fetch: func() (KeysPath, Chain, error) {
					_, path, chain, err := cli.Ancestry(cmd.Context(), flagWalkKey, collectOptions()...)
					return path, chain, err
				},
				view: ancestorsView,
			},
			modeChildren: {
				fetch: func() (KeysPath, Chain, error) {
					path, _, chain, err := cli.Descendants(cmd.Context(), flagWalkKey, collectOptions()...)
					return path, chain, err
				},
				view: descendantsView,
			},
			modeFamily: {
				fetch: func() (KeysPath, Chain, error) {
					path, _, chain, err := cli.Family(cmd.Context(), flagWalkKey, collectOptions()...)
					return path, chain, err
				},
				view: familyView,
			},
		}

		w, ok := walkers[flagWalkMode]
		if !ok {
			ferrors.HandleAndExit(log, fmt.Errorf("invalid --mode %q (must be %s, %s, or %s)", flagWalkMode, modeParent, modeChildren, modeFamily))
		}

		path, chain, err := w.fetch()
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}
		if err := w.view(cmd, path, chain); err != nil {
			ferrors.HandleAndExit(log, err)
		}
	},
}

func init() {
	walkCmd.AddCommand(walkProcessInstanceCmd)

	fs := walkProcessInstanceCmd.Flags()
	fs.StringVarP(&flagWalkKey, "key", "k", "", "start walking from this process instance key")
	_ = walkProcessInstanceCmd.MarkFlagRequired("key")

	fs.StringVar(&flagWalkMode, "mode", modeParent, "walk mode: parent, children, family")

	// shell completion for --mode
	_ = walkProcessInstanceCmd.RegisterFlagCompletionFunc("mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{modeParent, modeChildren, modeFamily}, cobra.ShellCompDirectiveNoFileComp
	})
}
