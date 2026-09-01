// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"sync"
	"time"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// processDefinitionDeleteCompletionPhase selects completion facts emitted by
// process-definition deletion services.
const processDefinitionDeleteCompletionPhase = "delete process definitions"

// processDefinitionDeleteSemanticProgressNow is overridden by command tests to
// exercise durable milestone pacing without real sleeps.
var processDefinitionDeleteSemanticProgressNow = time.Now

// processDefinitionDeleteSemanticProgress owns the command-local completion
// reporter for process-definition deletion facts.
type processDefinitionDeleteSemanticProgress struct {
	mu       sync.Mutex
	cmd      *cobra.Command
	total    int
	reporter *opsSemanticProgressReporter
	closed   bool
}

// newProcessDefinitionDeleteSemanticProgress returns a lazy deletion reporter
// wrapper whose total can be supplied before or during the first completion.
func newProcessDefinitionDeleteSemanticProgress(cmd *cobra.Command, total int) *processDefinitionDeleteSemanticProgress {
	return &processDefinitionDeleteSemanticProgress{cmd: cmd, total: total}
}

// Start eagerly opens the deletion reporter once a confirmed frozen scope is
// known, while still allowing lazy construction for auto-confirmed paths.
func (p *processDefinitionDeleteSemanticProgress) Start(total int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	_ = p.reporterLocked(total)
}

// Close flushes and stops the process-definition deletion reporter once.
func (p *processDefinitionDeleteSemanticProgress) Close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	reporter := p.reporter
	p.mu.Unlock()
	if reporter != nil {
		reporter.Close()
	}
}

// FacadeProgress maps public facade deletion completion facts into the command
// reporter without adding service-layer wording.
func (p *processDefinitionDeleteSemanticProgress) FacadeProgress(event foptions.ProgressEvent) {
	if event.Kind != foptions.ProgressEventKindCompletion || event.Completion == nil {
		return
	}
	p.Report(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:            event.Completion.Phase,
			CoreResource:     event.Completion.CoreResource,
			Total:            event.Completion.Total,
			Identity:         event.Completion.Identity,
			Disposition:      ops.CompletionDisposition(event.Completion.Disposition),
			FailureDetail:    event.Completion.FailureDetail,
			AffectedResource: event.Completion.AffectedResource,
			AffectedCount:    event.Completion.AffectedCount,
		},
	})
}

// Report forwards one process-definition completion fact into the lazy semantic
// reporter after filtering out non-completion progress.
func (p *processDefinitionDeleteSemanticProgress) Report(event ops.ProgressEvent) {
	if p == nil || event.Kind != ops.ProgressEventKindCompletion || event.Completion == nil {
		return
	}
	p.mu.Lock()
	reporter := p.reporterLocked(event.Completion.Total)
	p.mu.Unlock()
	if reporter != nil {
		reporter.Report(event)
	}
}

// reporterLocked constructs the reporter under the wrapper mutex and keeps the
// first trustworthy total for the command scope.
func (p *processDefinitionDeleteSemanticProgress) reporterLocked(total int) *opsSemanticProgressReporter {
	if p == nil || p.closed {
		return nil
	}
	if total > 0 && p.total <= 0 {
		p.total = total
	}
	if p.reporter == nil {
		channel := opsProgressChannelForMode(opsProgressModeForCommand(p.cmd, pickMode()))
		p.reporter = newOpsSemanticProgressReporter(p.cmd, opsSemanticProgressConfig{
			Scope: opsSemanticProgressScope{
				Phase:         processDefinitionDeleteCompletionPhase,
				ActivityLabel: "deleting process definitions",
				CoreResource:  "process definition(s)",
				Total:         p.total,
				SubmittedVerb: "submitted",
				ConfirmedVerb: "deleted",
				FailedVerb:    "failed",
			},
			Policy: opsSemanticProgressOutputPolicyForChannel(channel),
			Now:    processDefinitionDeleteSemanticProgressNow,
		})
	}
	return p.reporter
}
