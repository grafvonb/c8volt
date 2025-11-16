package foptions

import "github.com/grafvonb/c8volt/internal/services"

func WithNoStateCheck() FacadeOption { return func(c *FacadeCfg) { c.NoStateCheck = true } }
func WithForce() FacadeOption        { return func(c *FacadeCfg) { c.Force = true } }
func WithNoWait() FacadeOption       { return func(c *FacadeCfg) { c.NoWait = true } }
func WithRun() FacadeOption          { return func(c *FacadeCfg) { c.Run = true } }
func WithFailFast() FacadeOption     { return func(c *FacadeCfg) { c.FailFast = true } }
func WithVerbose() FacadeOption      { return func(c *FacadeCfg) { c.Verbose = true } }
func WithStat() FacadeOption         { return func(c *FacadeCfg) { c.Stat = true } }

type FacadeOption func(*FacadeCfg)

type FacadeCfg struct {
	NoStateCheck bool
	Force        bool
	NoWait       bool
	Run          bool
	FailFast     bool
	Verbose      bool
	Stat         bool
}

func ApplyFacadeOptions(opts []FacadeOption) *FacadeCfg {
	c := &FacadeCfg{}
	for _, o := range opts {
		o(c)
	}
	return c
}

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
	return out
}
