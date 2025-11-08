package cmd

import "github.com/grafvonb/c8volt/c8volt/foptions"

func collectOptions() []foptions.FacadeOption {
	var opts []foptions.FacadeOption
	if flagCancelNoWait || flagRunNoWait {
		opts = append(opts, foptions.WithNoWait())
	}
	if flagCancelNoStateCheck || flagDeleteNoStateCheck {
		opts = append(opts, foptions.WithNoStateCheck())
	}
	if flagDeletePIWithForce || flagCancelPIWithForce {
		opts = append(opts, foptions.WithForce())
	}
	if flagDeployPDWithRun {
		opts = append(opts, foptions.WithRun())
	}
	return opts
}
