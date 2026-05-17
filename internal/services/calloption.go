// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package services

func WithNoStateCheck() CallOption  { return func(c *CallCfg) { c.NoStateCheck = true } }
func WithForce() CallOption         { return func(c *CallCfg) { c.Force = true } }
func WithNoWait() CallOption        { return func(c *CallCfg) { c.NoWait = true } }
func WithRun() CallOption           { return func(c *CallCfg) { c.Run = true } }
func WithFailFast() CallOption      { return func(c *CallCfg) { c.FailFast = true } }
func WithStat() CallOption          { return func(c *CallCfg) { c.WithStat = true } }
func WithDryRun() CallOption        { return func(c *CallCfg) { c.DryRun = true } }
func WithVerbose() CallOption       { return func(c *CallCfg) { c.Verbose = true } }
func WithNoWorkerLimit() CallOption { return func(c *CallCfg) { c.NoWorkerLimit = true } }
func WithIgnoreTenant() CallOption  { return func(c *CallCfg) { c.IgnoreTenant = true } }
func WithSuppressWorkflowDetailLogs() CallOption {
	return func(c *CallCfg) { c.SuppressWorkflowDetailLogs = true }
}
func WithSuppressProcessInstanceDetailLogs() CallOption {
	return func(c *CallCfg) { c.SuppressProcessInstanceDetailLogs = true }
}
func WithIncidentState(state string) CallOption {
	return func(c *CallCfg) { c.IncidentState = state }
}
func WithIncidentErrorType(errorType string) CallOption {
	return func(c *CallCfg) { c.IncidentErrorType = errorType }
}
func WithIncidentErrorMessage(message string) CallOption {
	return func(c *CallCfg) { c.IncidentErrorMessage = message }
}
func WithAffectedProcessInstanceCount(count int) CallOption {
	return func(c *CallCfg) { c.AffectedProcessInstanceCount = count }
}

type CallOption func(*CallCfg)

type CallCfg struct {
	NoStateCheck                      bool
	Force                             bool
	NoWait                            bool
	Run                               bool
	FailFast                          bool
	WithStat                          bool
	DryRun                            bool
	Verbose                           bool
	NoWorkerLimit                     bool
	IgnoreTenant                      bool
	SuppressWorkflowDetailLogs        bool
	SuppressProcessInstanceDetailLogs bool
	IncidentState                     string
	IncidentErrorType                 string
	IncidentErrorMessage              string
	AffectedProcessInstanceCount      int
}

func ApplyCallOptions(opts []CallOption) *CallCfg {
	c := &CallCfg{}
	for _, o := range opts {
		o(c)
	}
	return c
}
