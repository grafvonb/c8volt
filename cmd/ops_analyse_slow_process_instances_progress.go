// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"sync"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// opsSlowProcessAnalysisSemanticProgressNow is overridden by command tests to
// exercise completion-driven milestone pacing without real sleeps.
var opsSlowProcessAnalysisSemanticProgressNow = time.Now

// configureOpsSlowProcessAnalysisPreflight wires command-owned rendering and prompting into service-owned discovery.
func configureOpsSlowProcessAnalysisPreflight(cmd *cobra.Command, request *ops.SlowProcessAnalysisRequest) *opsSlowProcessAnalysisSemanticProgress {
	return configureOpsSlowProcessAnalysisPreflightWithPacer(cmd, request, newOpsProgressMilestonePacer(nil))
}

// configureOpsSlowProcessAnalysisPreflightWithPacer allows progress tests to inject milestone pacing deterministically.
func configureOpsSlowProcessAnalysisPreflightWithPacer(cmd *cobra.Command, request *ops.SlowProcessAnalysisRequest, pacer *opsProgressMilestonePacer) *opsSlowProcessAnalysisSemanticProgress {
	if request == nil {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	progress := &opsSlowProcessAnalysisSemanticProgress{cmd: cmd}
	request.Progress = func(event ops.ProgressEvent) {
		switch event.Kind {
		case ops.ProgressEventKindPreflight:
			if event.Preflight != nil {
				printOpsPreflightScope(cmd, *event.Preflight, channel)
			}
		case ops.ProgressEventKindPage:
			if event.Page != nil {
				printOpsSlowProcessAnalysisProgressMilestone(cmd, formatOpsPageProgress(*event.Page, "process instance(s)"), event, channel, pacer)
			}
		case ops.ProgressEventKindFrozenScope:
			if event.FrozenScope != nil {
				printOpsSlowProcessAnalysisProgressMilestone(cmd, formatOpsFrozenScopeProgress(*event.FrozenScope), event, channel, pacer)
			}
		case ops.ProgressEventKindETA:
			if event.ETA != nil && opsETAAllowed(*event.ETA) {
				printOpsSlowProcessAnalysisProgressMilestone(cmd, formatOpsETASampleWindow(*event.ETA), event, channel, pacer)
			}
		case ops.ProgressEventKindCompletion:
			progress.Report(event)
		}
	}
	if request.SelectionMode == ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch {
		request.ConfirmPreflight = func(scope ops.PreflightScope) error {
			if !opsSlowProcessAnalysisPreflightConfirmationAllowed(channel) || !scope.RequiresConfirmation {
				return nil
			}
			prompt := strings.TrimSpace(scope.ConsequenceSummary.ConfirmationText)
			if prompt == "" {
				prompt = "Continue slow analysis?"
			}
			return confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt)
		}
	}
	return progress
}

// opsSlowProcessAnalysisSemanticProgress owns separate semantic reporters for
// finite enrichment phases discovered during slow-process analysis.
type opsSlowProcessAnalysisSemanticProgress struct {
	mu        sync.Mutex
	cmd       *cobra.Command
	reporters map[string]*opsSemanticProgressReporter
	closed    bool
}

// Report routes matching enrichment completion facts to their phase-specific
// semantic reporter.
func (p *opsSlowProcessAnalysisSemanticProgress) Report(event ops.ProgressEvent) {
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

// Close flushes and stops every enrichment reporter opened by this analysis.
func (p *opsSlowProcessAnalysisSemanticProgress) Close() {
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

// reporterLocked creates one enrichment reporter per completion phase while
// leaving discovery-page progress on the existing slow-analysis renderer.
func (p *opsSlowProcessAnalysisSemanticProgress) reporterLocked(completion ops.CompletionProgress) *opsSemanticProgressReporter {
	if p == nil || p.closed {
		return nil
	}
	scope, ok := opsSlowProcessAnalysisSemanticProgressScope(completion)
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
			Now:    opsSlowProcessAnalysisSemanticProgressNow,
		})
	}
	return p.reporters[scope.Phase]
}

// opsSlowProcessAnalysisSemanticProgressScope maps completion phases with exact
// enrichment totals and excludes discovery-only slow-analysis work.
func opsSlowProcessAnalysisSemanticProgressScope(completion ops.CompletionProgress) (opsSemanticProgressScope, bool) {
	phase := strings.TrimSpace(completion.Phase)
	switch phase {
	case "loading runtime elements", "loading listener jobs":
		return opsSemanticProgressScope{
			Phase:         phase,
			ActivityLabel: phase,
			CoreResource:  completion.CoreResource,
			Total:         completion.Total,
			ConfirmedVerb: "loaded",
			FailedVerb:    "failed",
		}, true
	default:
		return opsSemanticProgressScope{}, false
	}
}

// printOpsSlowProcessAnalysisProgress routes transient and verbose progress without touching command stdout.
func printOpsSlowProcessAnalysisProgress(cmd *cobra.Command, line string, channel ops.ProgressChannel) {
	printOpsSlowProcessAnalysisProgressMilestone(cmd, line, ops.ProgressEvent{}, channel, nil)
}

// printOpsSlowProcessAnalysisProgressMilestone keeps default progress transient while allowing paced durable milestones.
func printOpsSlowProcessAnalysisProgressMilestone(cmd *cobra.Command, line string, event ops.ProgressEvent, channel ops.ProgressChannel, pacer *opsProgressMilestonePacer) {
	if cmd == nil || strings.TrimSpace(line) == "" {
		return
	}
	if channel.TransientAllowed {
		logging.UpdateActivityWithImportance(cmd.Context(), line, logging.ActivityImportanceWorkflow)
	}
	if !opsSlowProcessAnalysisDurableProgressAllowed(channel) {
		if pacer != nil && pacer.AllowDurableMilestone(event, channel) {
			printOpsDurableLine(cmd, line, false)
		}
		return
	}
	printOpsDurableLine(cmd, line, false)
}

// printOpsPreflightScope writes durable preflight lines only to the command's stderr/activity channel.
func printOpsPreflightScope(cmd *cobra.Command, scope ops.PreflightScope, channel ops.ProgressChannel) {
	if cmd == nil || !channel.DurableAllowed || !channel.StderrAllowed {
		return
	}
	if scope.TenantContext != nil {
		attachTenantContext(cmd, *scope.TenantContext)
		printOpsTenantContext(cmd, *scope.TenantContext, channel)
	}
	printOpsPreflightLines(cmd, scope)
}

// opsSlowProcessAnalysisDurableProgressAllowed keeps page/counter detail behind verbose or debug while activity remains compact.
func opsSlowProcessAnalysisDurableProgressAllowed(channel ops.ProgressChannel) bool {
	return channel.DurableAllowed && channel.StderrAllowed && (channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug)
}

// opsSlowProcessAnalysisPreflightConfirmationAllowed limits prompts to human progress modes.
func opsSlowProcessAnalysisPreflightConfirmationAllowed(channel ops.ProgressChannel) bool {
	return channel.Mode == ops.ProgressModeHuman || channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug
}
