// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"sync"
	"time"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

const processDefinitionDeployCompletionPhase = "deploy process definitions"

// processDefinitionDeploySemanticProgressNow is overridden by command tests to
// exercise durable milestone pacing without real sleeps.
var processDefinitionDeploySemanticProgressNow = time.Now

// processDefinitionDeploySemanticProgress owns the command-local completion
// reporter for deployment facts whose total is known only after upload.
type processDefinitionDeploySemanticProgress struct {
	mu       sync.Mutex
	cmd      *cobra.Command
	reporter *opsSemanticProgressReporter
	closed   bool
}

// newProcessDefinitionDeploySemanticProgress returns a lazy reporter wrapper
// so v8.7 deployments stay silent when services cannot prove definition keys.
func newProcessDefinitionDeploySemanticProgress(cmd *cobra.Command) *processDefinitionDeploySemanticProgress {
	return &processDefinitionDeploySemanticProgress{cmd: cmd}
}

// Close ends the deployment workflow activity when any completion fact opened it.
func (p *processDefinitionDeploySemanticProgress) Close() {
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

// FacadeProgress maps public facade deployment completion facts into the
// command reporter without adding service-layer wording.
func (p *processDefinitionDeploySemanticProgress) FacadeProgress(event foptions.ProgressEvent) {
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

// Report sends one deployment completion event to the lazy semantic reporter.
func (p *processDefinitionDeploySemanticProgress) Report(event ops.ProgressEvent) {
	if p == nil || event.Kind != ops.ProgressEventKindCompletion || event.Completion == nil {
		return
	}
	if strings.TrimSpace(event.Completion.Phase) != processDefinitionDeployCompletionPhase {
		return
	}
	p.mu.Lock()
	reporter := p.reporterLocked(event.Completion.Total)
	p.mu.Unlock()
	if reporter != nil {
		reporter.Report(event)
	}
}

// reporterLocked constructs the reporter after the first completion reveals a
// trustworthy process-definition total.
func (p *processDefinitionDeploySemanticProgress) reporterLocked(total int) *opsSemanticProgressReporter {
	if p == nil || p.closed {
		return nil
	}
	if p.reporter == nil {
		channel := opsProgressChannelForMode(opsProgressModeForCommand(p.cmd, pickMode()))
		p.reporter = newOpsSemanticProgressReporter(p.cmd, opsSemanticProgressConfig{
			Scope: opsSemanticProgressScope{
				Phase:         processDefinitionDeployCompletionPhase,
				ActivityLabel: "deploying process definitions",
				CoreResource:  "process definition(s)",
				Total:         total,
				SubmittedVerb: "submitted",
				ConfirmedVerb: "deployed",
				FailedVerb:    "failed",
			},
			Policy: opsSemanticProgressOutputPolicyForChannel(channel),
			Now:    processDefinitionDeploySemanticProgressNow,
		})
	}
	return p.reporter
}

// appendProcessDefinitionDeployProgressOptions routes deployment completion
// facts into command progress and suppresses duplicate service detail logs.
func appendProcessDefinitionDeployProgressOptions(cmd *cobra.Command, opts []foptions.FacadeOption, progress *processDefinitionDeploySemanticProgress) []foptions.FacadeOption {
	out := append([]foptions.FacadeOption{}, opts...)
	out = append(out, foptions.WithSuppressWorkflowDetailLogs())
	if progress != nil {
		out = append(out, foptions.WithProgress(progress.FacadeProgress))
	}
	return out
}
