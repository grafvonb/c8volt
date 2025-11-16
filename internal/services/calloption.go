package services

func WithNoStateCheck() CallOption { return func(c *CallCfg) { c.NoStateCheck = true } }
func WithForce() CallOption        { return func(c *CallCfg) { c.Force = true } }
func WithNoWait() CallOption       { return func(c *CallCfg) { c.NoWait = true } }
func WithRun() CallOption          { return func(c *CallCfg) { c.Run = true } }
func WithFailFast() CallOption     { return func(c *CallCfg) { c.FailFast = true } }
func WithStat() CallOption         { return func(c *CallCfg) { c.WithStat = true } }

type CallOption func(*CallCfg)

type CallCfg struct {
	NoStateCheck bool
	Force        bool
	NoWait       bool
	Run          bool
	FailFast     bool
	WithStat     bool
}

func ApplyCallOptions(opts []CallOption) *CallCfg {
	c := &CallCfg{}
	for _, o := range opts {
		o(c)
	}
	return c
}
