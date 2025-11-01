package cmd

import "github.com/grafvonb/c8volt/c8volt/options"

func collectOptions() []options.FacadeOption {
	var opts []options.FacadeOption
	if flagCancelWait {
		opts = append(opts, options.WithWait())
	}
	if flagCancelPINoStateCheck {
		opts = append(opts, options.WithNoStateCheck())
	}
	if flagDeletePIWithCancel {
		opts = append(opts, options.WithCancel())
	}
	return opts
}
