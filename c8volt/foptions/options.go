// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package foptions

import (
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

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

func WithIgnoreTenant() FacadeOption { return func(c *FacadeCfg) { c.IgnoreTenant = true } }

// WithIncidentState selects the incident state scope for process-instance incident enrichment.
func WithIncidentState(state string) FacadeOption {
	return func(c *FacadeCfg) { c.IncidentState = state }
}

// WithIncidentErrorType filters process-instance incident enrichment by error type.
func WithIncidentErrorType(errorType string) FacadeOption {
	return func(c *FacadeCfg) { c.IncidentErrorType = errorType }
}

// WithIncidentErrorMessage filters process-instance incident enrichment by message substring.
func WithIncidentErrorMessage(message string) FacadeOption {
	return func(c *FacadeCfg) { c.IncidentErrorMessage = message }
}

// WithAffectedProcessInstanceCount carries impact-check expansion metadata for facade-level summaries.
func WithAffectedProcessInstanceCount(count int) FacadeOption {
	return func(c *FacadeCfg) { c.AffectedProcessInstanceCount = count }
}

// WithProgress installs an optional structured progress callback for facade
// calls that support long-running discovery or frozen-scope enrichment.
func WithProgress(progress func(ProgressEvent)) FacadeOption {
	return func(c *FacadeCfg) { c.Progress = progress }
}

type FacadeOption func(*FacadeCfg)

type FacadeCfg struct {
	NoStateCheck                 bool
	Force                        bool
	NoWait                       bool
	Run                          bool
	FailFast                     bool
	Verbose                      bool
	Stat                         bool
	DryRun                       bool
	NoWorkerLimit                bool
	IgnoreTenant                 bool
	IncidentState                string
	IncidentErrorType            string
	IncidentErrorMessage         string
	AffectedProcessInstanceCount int
	Progress                     func(ProgressEvent)
}

// ProgressEventKind identifies the kind of structured progress fact emitted by a facade call.
type ProgressEventKind string

const (
	// ProgressEventKindFrozenScope carries exact counters for a frozen work set.
	ProgressEventKindFrozenScope ProgressEventKind = "frozen_scope"
)

// FrozenScopeProgress reports exact progress across an immutable work set.
type FrozenScopeProgress struct {
	Phase        string         `json:"phase,omitempty"`
	CoreResource string         `json:"coreResource,omitempty"`
	Done         int            `json:"done,omitempty"`
	Total        int            `json:"total,omitempty"`
	Elapsed      time.Duration  `json:"elapsed,omitempty"`
	Rate         *float64       `json:"rate,omitempty"`
	ETA          *time.Duration `json:"eta,omitempty"`
	Errors       int            `json:"errors,omitempty"`
}

// ProgressEvent is a typed envelope for facade progress callbacks.
type ProgressEvent struct {
	Kind        ProgressEventKind    `json:"kind,omitempty"`
	FrozenScope *FrozenScopeProgress `json:"frozenScope,omitempty"`
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
	if c.IgnoreTenant {
		out = append(out, services.WithIgnoreTenant())
	}
	if c.IncidentState != "" {
		out = append(out, services.WithIncidentState(c.IncidentState))
	}
	if c.IncidentErrorType != "" {
		out = append(out, services.WithIncidentErrorType(c.IncidentErrorType))
	}
	if c.IncidentErrorMessage != "" {
		out = append(out, services.WithIncidentErrorMessage(c.IncidentErrorMessage))
	}
	if c.AffectedProcessInstanceCount > 0 {
		out = append(out, services.WithAffectedProcessInstanceCount(c.AffectedProcessInstanceCount))
	}
	if c.Progress != nil {
		out = append(out, services.WithProgress(toDomainProgressFunc(c.Progress)))
	}
	return out
}

func toDomainProgressFunc(fn func(ProgressEvent)) func(d.OpsProgressEvent) {
	if fn == nil {
		return nil
	}
	return func(event d.OpsProgressEvent) {
		fn(fromDomainProgressEvent(event))
	}
}

func fromDomainProgressEvent(event d.OpsProgressEvent) ProgressEvent {
	return ProgressEvent{
		Kind:        ProgressEventKind(event.Kind),
		FrozenScope: fromDomainFrozenScopeProgressPtr(event.FrozenScope),
	}
}

func fromDomainFrozenScopeProgressPtr(progress *d.OpsFrozenScopeProgress) *FrozenScopeProgress {
	if progress == nil {
		return nil
	}
	out := FrozenScopeProgress{
		Phase:        progress.Phase,
		CoreResource: progress.CoreResource,
		Done:         progress.Done,
		Total:        progress.Total,
		Elapsed:      progress.Elapsed,
		Rate:         progress.Rate,
		ETA:          progress.ETA,
		Errors:       progress.Errors,
	}
	return &out
}
