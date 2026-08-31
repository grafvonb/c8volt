// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"sync"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

const opsRepairCompletionPhase = "repairing incidents"

// opsRepairSemanticProgress owns the command-local completion reporter for
// incident and process-instance-selected repair work.
type opsRepairSemanticProgress struct {
	mu       sync.Mutex
	cmd      *cobra.Command
	reporter *opsSemanticProgressReporter
	closed   bool
}

// configureOpsRepairProgress installs command-owned repair progress rendering on the facade request.
func configureOpsRepairProgress(cmd *cobra.Command, request *ops.RepairRequest) *opsRepairSemanticProgress {
	if request == nil {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	progress := &opsRepairSemanticProgress{cmd: cmd}
	request.Progress = func(event ops.ProgressEvent) {
		printOpsRepairProgressEvent(cmd, event, channel, progress)
	}
	return progress
}

// printOpsRepairProgressEvent routes repair preflight and counters without writing to result stdout.
func printOpsRepairProgressEvent(cmd *cobra.Command, event ops.ProgressEvent, channel ops.ProgressChannel, progress *opsRepairSemanticProgress) {
	switch event.Kind {
	case ops.ProgressEventKindPreflight:
		if event.Preflight != nil {
			printOpsPreflightScope(cmd, *event.Preflight, channel)
		}
	case ops.ProgressEventKindPage:
		if event.Page != nil {
			printOpsSlowProcessAnalysisProgress(cmd, formatOpsPageProgress(*event.Page, ""), channel)
		}
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

// Report forwards repair completion facts into the lazy semantic reporter.
func (p *opsRepairSemanticProgress) Report(event ops.ProgressEvent) {
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

// Close ends the repair workflow activity if any completion fact opened it.
func (p *opsRepairSemanticProgress) Close() {
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

// reporterLocked constructs the repair reporter after a real repair completion
// fact arrives, leaving dry-run planning progress on the existing renderer.
func (p *opsRepairSemanticProgress) reporterLocked(completion ops.CompletionProgress) *opsSemanticProgressReporter {
	if p == nil || p.closed || strings.TrimSpace(completion.Phase) != opsRepairCompletionPhase {
		return nil
	}
	if p.reporter == nil {
		channel := opsProgressChannelForMode(opsProgressModeForCommand(p.cmd, pickMode()))
		p.reporter = newOpsSemanticProgressReporter(p.cmd, opsSemanticProgressConfig{
			Scope: opsSemanticProgressScope{
				Phase:         opsRepairCompletionPhase,
				ActivityLabel: "repairing incidents",
				CoreResource:  "incident(s)",
				Total:         completion.Total,
				SubmittedVerb: "submitted",
				ConfirmedVerb: "repaired",
				FailedVerb:    "failed",
			},
			Policy: opsSemanticProgressOutputPolicyForChannel(channel),
		})
	}
	return p.reporter
}
