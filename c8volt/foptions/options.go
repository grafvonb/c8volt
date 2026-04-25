package foptions

import "github.com/grafvonb/c8volt/internal/services"

// WithNoStateCheck disables facade-level state validation before a state-changing operation.
func WithNoStateCheck() FacadeOption { return func(c *FacadeCfg) { c.NoStateCheck = true } }

// WithForce allows facade operations to perform prerequisite actions such as cancelling active instances before deletion.
func WithForce() FacadeOption { return func(c *FacadeCfg) { c.Force = true } }

// WithNoWait asks operations to return after submission instead of waiting for confirmation.
func WithNoWait() FacadeOption { return func(c *FacadeCfg) { c.NoWait = true } }

// WithRun asks deploy-style flows to start process instances after successful deployment.
func WithRun() FacadeOption { return func(c *FacadeCfg) { c.Run = true } }

// WithFailFast cancels remaining bulk work after the first scheduled item fails.
func WithFailFast() FacadeOption { return func(c *FacadeCfg) { c.FailFast = true } }

// WithVerbose enables progress-oriented logging and diagnostics in facade calls.
func WithVerbose() FacadeOption { return func(c *FacadeCfg) { c.Verbose = true } }

// WithStat requests optional statistics where the selected Camunda version supports them.
func WithStat() FacadeOption { return func(c *FacadeCfg) { c.Stat = true } }

// WithDryRun requests dependency expansion or validation without applying the state-changing action.
func WithDryRun() FacadeOption { return func(c *FacadeCfg) { c.DryRun = true } }

// WithNoWorkerLimit disables the default cap that keeps requested workers within the runtime worker policy.
func WithNoWorkerLimit() FacadeOption { return func(c *FacadeCfg) { c.NoWorkerLimit = true } }

// WithAllowInconsistent opts into eventually consistent or destructive API operations that are otherwise guarded.
func WithAllowInconsistent() FacadeOption { return func(c *FacadeCfg) { c.AllowInconsistent = true } }

type FacadeOption func(*FacadeCfg)

type FacadeCfg struct {
	NoStateCheck      bool
	Force             bool
	NoWait            bool
	Run               bool
	FailFast          bool
	Verbose           bool
	Stat              bool
	DryRun            bool
	NoWorkerLimit     bool
	AllowInconsistent bool
}

// ApplyFacadeOptions folds facade options into a new configuration value.
// opts are applied in order, so later options can extend the same mutable config during construction.
func ApplyFacadeOptions(opts []FacadeOption) *FacadeCfg {
	c := &FacadeCfg{}
	for _, o := range opts {
		o(c)
	}
	return c
}

// MapFacadeOptionsToCallOptions translates public facade options to internal service call options.
// This is the boundary that keeps the exported facade independent from internal service configuration types.
func MapFacadeOptionsToCallOptions(opts []FacadeOption) []services.CallOption {
	c := ApplyFacadeOptions(opts)
	var out []services.CallOption
	if c.NoStateCheck {
		out = append(out, services.WithNoStateCheck())
	}
	if c.Force {
		out = append(out, services.WithForce())
	}
	if c.NoWait {
		out = append(out, services.WithNoWait())
	}
	if c.Run {
		out = append(out, services.WithRun())
	}
	if c.FailFast {
		out = append(out, services.WithFailFast())
	}
	if c.Stat {
		out = append(out, services.WithStat())
	}
	if c.DryRun {
		out = append(out, services.WithDryRun())
	}
	if c.Verbose {
		out = append(out, services.WithVerbose())
	}
	if c.NoWorkerLimit {
		out = append(out, services.WithNoWorkerLimit())
	}
	if c.AllowInconsistent {
		out = append(out, services.WithAllowInconsistent())
	}
	return out
}
