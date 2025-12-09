package cmd

import options "github.com/grafvonb/c8volt/c8volt/foptions"

func collectOptions() []options.FacadeOption {
	var opts []options.FacadeOption
	if flagNoWait {
		opts = append(opts, options.WithNoWait())
	}
	if flagNoStateCheck {
		opts = append(opts, options.WithNoStateCheck())
	}
	if flagForce {
		opts = append(opts, options.WithForce())
	}
	if flagDeployPDWithRun {
		opts = append(opts, options.WithRun())
	}
	if flagGetPDWithStat {
		opts = append(opts, options.WithStat())
	}
	if flagDryRun {
		opts = append(opts, options.WithDryRun())
	}
	if flagVerbose {
		opts = append(opts, options.WithVerbose())
	}
	if flagFailFast {
		opts = append(opts, options.WithFailFast())
	}
	if flagNoWorkerLimit {
		opts = append(opts, options.WithNoWorkerLimit())
	}
	return opts
}
