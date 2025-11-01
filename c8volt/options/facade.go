package options

import "github.com/grafvonb/c8volt/internal/services"

func WithNoStateCheck() FacadeOption { return func(c *FacadeCfg) { c.NoStateCheck = true } }
func WithCancel() FacadeOption       { return func(c *FacadeCfg) { c.Cancel = true } }
func WithWait() FacadeOption         { return func(c *FacadeCfg) { c.Wait = true } }

type FacadeOption func(*FacadeCfg)

type FacadeCfg struct {
	NoStateCheck bool
	Cancel       bool
	Wait         bool
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
	if c.Cancel {
		out = append(out, services.WithCancel())
	}
	if c.Wait {
		out = append(out, services.WithWait())
	}
	return out
}
