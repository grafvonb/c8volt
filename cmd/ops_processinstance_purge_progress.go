// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"sync"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

type opsProcessInstancePurgeSemanticProgress struct {
	mu       sync.Mutex
	cmd      *cobra.Command
	reporter *opsSemanticProgressReporter
	closed   bool
}

func configureOpsExecuteRetentionPolicyProgress(cmd *cobra.Command, request *ops.RetentionPolicyRequest) *opsProcessInstancePurgeSemanticProgress {
	if request == nil {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	progress := &opsProcessInstancePurgeSemanticProgress{cmd: cmd}
	request.Progress = func(event ops.ProgressEvent) {
		printOpsProcessInstancePurgeProgressEvent(cmd, event, channel, progress)
	}
	return progress
}

func configureOpsPurgeOrphanProcessInstancesProgress(cmd *cobra.Command, request *ops.OrphanPurgeRequest) *opsProcessInstancePurgeSemanticProgress {
	if request == nil {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	progress := &opsProcessInstancePurgeSemanticProgress{cmd: cmd}
	request.Progress = func(event ops.ProgressEvent) {
		printOpsProcessInstancePurgeProgressEvent(cmd, event, channel, progress)
	}
	return progress
}

func configureOpsPurgeProcessInstancesWithIncidentsProgress(cmd *cobra.Command, request *ops.IncidentPurgeRequest) *opsProcessInstancePurgeSemanticProgress {
	if request == nil {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	progress := &opsProcessInstancePurgeSemanticProgress{cmd: cmd}
	request.Progress = func(event ops.ProgressEvent) {
		printOpsProcessInstancePurgeProgressEvent(cmd, event, channel, progress)
	}
	return progress
}

func printOpsProcessInstancePurgeProgressEvent(cmd *cobra.Command, event ops.ProgressEvent, channel ops.ProgressChannel, progress *opsProcessInstancePurgeSemanticProgress) {
	switch event.Kind {
	case ops.ProgressEventKindPreflight:
		if event.Preflight != nil {
			printOpsPreflightScope(cmd, *event.Preflight, channel)
		}
	case ops.ProgressEventKindPage:
		if event.Page != nil {
			printOpsSlowProcessAnalysisProgress(cmd, formatOpsPageProgress(*event.Page, "process instance(s)"), channel)
		}
	case ops.ProgressEventKindFrozenScope:
		if event.FrozenScope != nil && !opsProcessInstancePurgeCompletionFrozenScope(*event.FrozenScope) {
			printOpsSlowProcessAnalysisProgress(cmd, formatProcessInstanceMutationFrozenProgress(*event.FrozenScope), channel)
		}
	case ops.ProgressEventKindCompletion:
		if progress != nil {
			progress.Report(event)
		}
	}
}

func (p *opsProcessInstancePurgeSemanticProgress) Report(event ops.ProgressEvent) {
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

func (p *opsProcessInstancePurgeSemanticProgress) Close() {
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

func (p *opsProcessInstancePurgeSemanticProgress) reporterLocked(completion ops.CompletionProgress) *opsSemanticProgressReporter {
	if p == nil || p.closed || !opsProcessInstancePurgeCompletionMatches(completion) {
		return nil
	}
	if p.reporter == nil {
		channel := opsProgressChannelForMode(opsProgressModeForCommand(p.cmd, pickMode()))
		p.reporter = newOpsSemanticProgressReporter(p.cmd, opsSemanticProgressConfig{
			Scope:  processInstanceMutationSemanticProgressScope("delete", completion.Total, completion.AffectedCount != nil),
			Policy: opsSemanticProgressOutputPolicyForChannel(channel),
		})
	}
	return p.reporter
}

func opsProcessInstancePurgeCompletionMatches(completion ops.CompletionProgress) bool {
	return strings.TrimSpace(completion.Phase) == "delete"
}

func opsProcessInstancePurgeCompletionFrozenScope(progress ops.FrozenScopeProgress) bool {
	return strings.TrimSpace(progress.Phase) == "deleting process instances"
}
