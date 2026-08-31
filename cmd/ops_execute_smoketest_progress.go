// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"sync"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// opsSmokeTestSemanticProgress owns command-local reporters for the finite
// smoke-test stages that expose real completion facts.
type opsSmokeTestSemanticProgress struct {
	mu        sync.Mutex
	cmd       *cobra.Command
	reporters map[string]*opsSemanticProgressReporter
	closed    bool
}

// configureOpsExecuteSmokeTestProgress installs command-owned smoke-test progress rendering on the facade request.
func configureOpsExecuteSmokeTestProgress(cmd *cobra.Command, request *ops.SmokeTestRequest) *opsSmokeTestSemanticProgress {
	if request == nil {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	progress := &opsSmokeTestSemanticProgress{cmd: cmd}
	request.Progress = func(event ops.ProgressEvent) {
		printOpsExecuteSmokeTestProgressEvent(cmd, event, channel, progress)
	}
	return progress
}

// printOpsExecuteSmokeTestProgressEvent preserves existing stage counters and
// forwards completion facts to semantic stage reporters.
func printOpsExecuteSmokeTestProgressEvent(cmd *cobra.Command, event ops.ProgressEvent, channel ops.ProgressChannel, progress *opsSmokeTestSemanticProgress) {
	switch event.Kind {
	case ops.ProgressEventKindFrozenScope:
		if event.FrozenScope != nil {
			printOpsSlowProcessAnalysisProgress(cmd, formatProcessInstanceMutationFrozenProgress(*event.FrozenScope), channel)
		}
	case ops.ProgressEventKindCompletion:
		if progress != nil {
			progress.Report(event)
		}
	}
}

// Report forwards a smoke-test stage completion to the reporter for that
// exact stage and ignores nested lower-level completion phases.
func (p *opsSmokeTestSemanticProgress) Report(event ops.ProgressEvent) {
	if p == nil || event.Kind != ops.ProgressEventKindCompletion || event.Completion == nil {
		return
	}
	p.mu.Lock()
	reporter := p.reporterLocked(*event.Completion)
	p.mu.Unlock()
	if reporter != nil {
		reporter.Report(event)
	}
}

// Close ends every smoke-test stage reporter that was opened by completion
// facts during the command run.
func (p *opsSmokeTestSemanticProgress) Close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	reporters := make([]*opsSemanticProgressReporter, 0, len(p.reporters))
	for _, reporter := range p.reporters {
		reporters = append(reporters, reporter)
	}
	p.mu.Unlock()
	for _, reporter := range reporters {
		reporter.Close()
	}
}

// reporterLocked constructs one semantic reporter per smoke-test stage so each
// stage keeps its own exact total and lifecycle vocabulary.
func (p *opsSmokeTestSemanticProgress) reporterLocked(completion ops.CompletionProgress) *opsSemanticProgressReporter {
	if p == nil || p.closed {
		return nil
	}
	scope, ok := opsSmokeTestSemanticProgressScope(completion)
	if !ok {
		return nil
	}
	if p.reporters == nil {
		p.reporters = make(map[string]*opsSemanticProgressReporter)
	}
	if p.reporters[scope.Phase] == nil {
		channel := opsProgressChannelForMode(opsProgressModeForCommand(p.cmd, pickMode()))
		p.reporters[scope.Phase] = newOpsSemanticProgressReporter(p.cmd, opsSemanticProgressConfig{
			Scope:  scope,
			Policy: opsSemanticProgressOutputPolicyForChannel(channel),
		})
	}
	return p.reporters[scope.Phase]
}

// opsSmokeTestSemanticProgressScope maps high-level smoke-test completion
// phases to command-owned wording and rejects nested service phases.
func opsSmokeTestSemanticProgressScope(completion ops.CompletionProgress) (opsSemanticProgressScope, bool) {
	phase := strings.TrimSpace(completion.Phase)
	scope := opsSemanticProgressScope{
		Phase:         phase,
		ActivityLabel: phase,
		CoreResource:  completion.CoreResource,
		Total:         completion.Total,
		SubmittedVerb: "submitted",
		FailedVerb:    "failed",
	}
	switch phase {
	case "deploying smoke-test fixture":
		scope.ConfirmedVerb = "deployed"
	case "starting process instances":
		scope.ConfirmedVerb = "started"
	case "walking process-instance families":
		scope.ConfirmedVerb = "walked"
	case "cleaning up smoke-test process instances", "cleaning up smoke-test process definition":
		scope.ConfirmedVerb = "deleted"
	default:
		return opsSemanticProgressScope{}, false
	}
	return scope, true
}
