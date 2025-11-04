package cmd

import "github.com/grafvonb/c8volt/c8volt/options"

func collectOptions() []options.FacadeOption {
	var opts []options.FacadeOption
	if flagCancelNoWait || flagRunNoWait {
		opts = append(opts, options.WithNoWait())
	}
	if flagCancelNoStateCheck {
		opts = append(opts, options.WithNoStateCheck())
	}
	if flagDeletePIWithForce || flagCancelPIWithForce {
		opts = append(opts, options.WithForce())
	}
	if flagDeployPDWithRun {
		opts = append(opts, options.WithRun())
	}
	return opts
}
