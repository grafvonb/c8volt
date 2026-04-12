package testx

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func ResetCommandTreeFlags(cmd *cobra.Command) {
	resetFlagSet := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(flag *pflag.Flag) {
			_ = flag.Value.Set(flag.DefValue)
			flag.Changed = false
		})
	}

	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())
	resetFlagSet(cmd.InheritedFlags())

	for _, child := range cmd.Commands() {
		ResetCommandTreeFlags(child)
	}
}
