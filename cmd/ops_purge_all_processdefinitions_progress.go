// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

const (
	opsPurgeAllProcessDefinitionsCancelPhase             = "cancel"
	opsPurgeAllProcessDefinitionsDrainPhase              = "drain process instances"
	opsPurgeAllProcessDefinitionsHistoryDeletePhase      = "delete"
	opsPurgeAllProcessDefinitionsPlannedAffectedResource = "process instance(s)"
	opsPurgeAllProcessDefinitionsGenericActivity         = "running process-definition purge workflow"
)

// opsPurgeAllProcessDefinitionsProgressNow is overridden by command tests to
// exercise milestone timing without sleeping.
var opsPurgeAllProcessDefinitionsProgressNow = time.Now

// opsPurgeAllProcessDefinitionsProgressConfig groups the command-owned
// dependencies needed for one all-process-definitions purge progress lifetime.
type opsPurgeAllProcessDefinitionsProgressConfig struct {
	Policy opsSemanticProgressOutputPolicy
	Now    func() time.Time
}

// opsPurgeAllProcessDefinitionsProgress coordinates nested purge stage facts
// into one workflow activity without changing backend execution.
type opsPurgeAllProcessDefinitionsProgress struct {
	mu                   sync.Mutex
	cmd                  *cobra.Command
	policy               opsSemanticProgressOutputPolicy
	now                  func() time.Time
	startedAt            time.Time
	lastInformationalAt  time.Time
	activityStarted      bool
	mutationClockStarted bool
	stop                 func()
	currentPhase         string
	stages               []opsPurgeAllProcessDefinitionsStageRecord
	stageIndexes         map[string]int
	durableActivated     bool
	closed               bool
}

// opsPurgeAllProcessDefinitionsStageRecord stores one mutation phase aggregate
// and the separate planned affected scope supplied by the service.
type opsPurgeAllProcessDefinitionsStageRecord struct {
	Scope                opsSemanticProgressScope
	Aggregate            opsSemanticProgressAggregate
	PlannedAffectedCount *int
	Dirty                bool
}

// newOpsPurgeAllProcessDefinitionsProgress constructs a dormant coordinator;
// real execution or the first stage fact starts its activity lifetime.
func newOpsPurgeAllProcessDefinitionsProgress(cmd *cobra.Command, cfg opsPurgeAllProcessDefinitionsProgressConfig) *opsPurgeAllProcessDefinitionsProgress {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &opsPurgeAllProcessDefinitionsProgress{
		cmd:          cmd,
		policy:       cfg.Policy,
		now:          now,
		stop:         func() {},
		stageIndexes: make(map[string]int),
	}
}

// newOpsPurgeAllProcessDefinitionsProgressForCommand derives the current
// output policy for the APD command's real-execution progress coordinator.
func newOpsPurgeAllProcessDefinitionsProgressForCommand(cmd *cobra.Command) *opsPurgeAllProcessDefinitionsProgress {
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	return newOpsPurgeAllProcessDefinitionsProgress(cmd, opsPurgeAllProcessDefinitionsProgressConfig{
		Policy: opsSemanticProgressOutputPolicyForChannel(channel),
		Now:    opsPurgeAllProcessDefinitionsProgressNow,
	})
}

// configureOpsPurgeAllProcessDefinitionsProgress keeps discovery progress on
// the APD renderer while forwarding entered mutation stages to the coordinator.
func configureOpsPurgeAllProcessDefinitionsProgress(cmd *cobra.Command, request *ops.AllProcessDefinitionsPurgeRequest, progress *opsPurgeAllProcessDefinitionsProgress) {
	if request == nil {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	request.Progress = func(event ops.ProgressEvent) {
		handleOpsTenantScopeProgressEvent(cmd, event, channel)
		switch event.Kind {
		case ops.ProgressEventKindPreflight:
			if event.Preflight != nil {
				printOpsPreflightScope(cmd, *event.Preflight, channel)
			}
		case ops.ProgressEventKindPage:
			if event.Page != nil {
				printOpsSlowProcessAnalysisProgress(cmd, formatOpsPageProgress(*event.Page, "process definition(s)"), channel)
			}
		case ops.ProgressEventKindStage, ops.ProgressEventKindCompletion:
			if progress != nil {
				progress.Report(event)
			}
		}
	}
}

// runOpsPurgeAllProcessDefinitionsWithCommandProgress preserves the legacy
// preview activity while making real execution use one coordinator-owned scope.
func runOpsPurgeAllProcessDefinitionsWithCommandProgress(cmd *cobra.Command, request ops.AllProcessDefinitionsPurgeRequest, progress *opsPurgeAllProcessDefinitionsProgress, run func() (ops.AllProcessDefinitionsPurgeResult, error)) (ops.AllProcessDefinitionsPurgeResult, error) {
	if request.DryRun {
		return purgeAllProcessDefinitionsWithCommandActivity(cmd, request, run)
	}
	if progress != nil {
		progress.Start()
	}
	return run()
}

// Start opens the generic real-execution activity once so nested lower-level
// activity cannot become the workflow owner.
func (p *opsPurgeAllProcessDefinitionsProgress) Start() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startLocked()
}

// Report routes only recognized stage and completion facts into the active
// command-owned workflow state.
func (p *opsPurgeAllProcessDefinitionsProgress) Report(event ops.ProgressEvent) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	switch event.Kind {
	case ops.ProgressEventKindStage:
		if event.Stage != nil {
			p.enterStageLocked(*event.Stage)
		}
	case ops.ProgressEventKindCompletion:
		if event.Completion != nil {
			p.ingestCompletionLocked(*event.Completion)
		}
	}
}

// Aggregate returns a stage aggregate copy for focused coordinator tests.
func (p *opsPurgeAllProcessDefinitionsProgress) Aggregate(phase string) (opsSemanticProgressAggregate, bool) {
	if p == nil {
		return opsSemanticProgressAggregate{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	idx, ok := p.stageIndexes[strings.TrimSpace(phase)]
	if !ok {
		return opsSemanticProgressAggregate{}, false
	}
	return p.stages[idx].Aggregate, true
}

// Close ends the real-execution activity once and preserves the existing
// single-stage final flush behavior while joining cross-stage dirty history.
func (p *opsPurgeAllProcessDefinitionsProgress) Close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	stop := p.stop
	var finalLine string
	if p.policy.PacedAggregate && p.durableActivated {
		finalLine = formatOpsPurgeAllProcessDefinitionsFinalAggregate(p.stages)
		for i := range p.stages {
			p.stages[i].Dirty = false
		}
	}
	p.mu.Unlock()
	if finalLine != "" {
		printOpsDurableLine(p.cmd, finalLine, false)
	}
	if stop != nil {
		stop()
	}
}

// startLocked initializes the workflow activity clock under the coordinator
// mutex and intentionally starts no activity for suppressed modes.
func (p *opsPurgeAllProcessDefinitionsProgress) startLocked() {
	if p == nil || p.closed || p.activityStarted {
		return
	}
	p.activityStarted = true
	p.stop = func() {}
	if p.cmd != nil && p.policy.TransientActivity {
		p.stop = logging.StartActivityWithImportance(opsPurgeAllProcessDefinitionsProgressCommandContext(p.cmd), opsPurgeAllProcessDefinitionsGenericActivity, logging.ActivityImportanceWorkflow)
	}
}

// startMutationClockLocked anchors durable pacing to the first actual stage
// entry, not to discovery or generic workflow activity setup.
func (p *opsPurgeAllProcessDefinitionsProgress) startMutationClockLocked() {
	if p == nil || p.mutationClockStarted {
		return
	}
	p.mutationClockStarted = true
	p.startedAt = p.now()
	p.lastInformationalAt = p.startedAt
}

// enterStageLocked records a service-owned stage entry before the first
// matching completion and updates only the current workflow activity.
func (p *opsPurgeAllProcessDefinitionsProgress) enterStageLocked(stage ops.StageProgress) {
	phase := strings.TrimSpace(stage.Phase)
	scope, waitingLabel, ok := opsPurgeAllProcessDefinitionsStageScope(stage)
	if !ok {
		return
	}
	p.startLocked()
	p.startMutationClockLocked()
	p.currentPhase = phase
	if waitingLabel != "" {
		p.updateActivityLocked(waitingLabel)
		return
	}
	idx, ok := p.stageIndexes[phase]
	if !ok {
		if len(p.stages) >= 3 {
			return
		}
		aggregate := opsSemanticProgressAggregate{
			AffectedValid: strings.TrimSpace(scope.AffectedResource) != "",
		}
		if stage.Total != nil && *stage.Total > 0 {
			aggregate.Total = *stage.Total
		}
		p.stages = append(p.stages, opsPurgeAllProcessDefinitionsStageRecord{
			Scope:                scope,
			Aggregate:            aggregate,
			PlannedAffectedCount: toolx.CopyPtr(stage.PlannedAffectedCount),
		})
		idx = len(p.stages) - 1
		p.stageIndexes[phase] = idx
	} else if p.stages[idx].PlannedAffectedCount == nil && stage.PlannedAffectedCount != nil {
		p.stages[idx].PlannedAffectedCount = toolx.CopyPtr(stage.PlannedAffectedCount)
	}
	p.updateActivityLocked(formatOpsPurgeAllProcessDefinitionsStageAggregate(p.stages[idx]))
}

// ingestCompletionLocked applies only completions for phases that have been
// explicitly entered, keeping historical facts from repainting current activity.
func (p *opsPurgeAllProcessDefinitionsProgress) ingestCompletionLocked(completion ops.CompletionProgress) {
	phase := strings.TrimSpace(completion.Phase)
	if phase == "" {
		return
	}
	idx, ok := p.stageIndexes[phase]
	if !ok {
		return
	}
	record := &p.stages[idx]
	aggregate, dirty := applyOpsSemanticCompletionToAggregate(record.Aggregate, completion)
	record.Aggregate = aggregate
	if dirty {
		record.Dirty = true
	}
	line := formatOpsPurgeAllProcessDefinitionsStageAggregate(*record)
	if p.currentPhase == phase {
		p.updateActivityLocked(line)
	}
	if p.policy.VerboseItems {
		p.printDurableLineLocked(formatOpsPurgeAllProcessDefinitionsStageCompletion(*record, completion), p.policy.FailureWarnings && completion.Disposition == ops.CompletionDispositionFailed)
		p.durableActivated = true
		record.Dirty = false
		return
	}
	if p.policy.FailureWarnings && completion.Disposition == ops.CompletionDispositionFailed {
		p.printDurableLineLocked(formatOpsPurgeAllProcessDefinitionsStageCompletion(*record, completion), true)
		p.durableActivated = true
		record.Dirty = false
		return
	}
	if p.policy.PacedAggregate && record.Dirty {
		p.startMutationClockLocked()
		now := p.now()
		if now.Before(p.lastInformationalAt.Add(opsDurableMilestoneMinimumElapsed)) {
			return
		}
		p.printDurableLineLocked(line, false)
		p.lastInformationalAt = now
		p.durableActivated = true
		record.Dirty = false
	}
}

// updateActivityLocked writes one workflow-priority activity update only when
// the active output mode permits transient activity.
func (p *opsPurgeAllProcessDefinitionsProgress) updateActivityLocked(line string) {
	if p == nil || p.cmd == nil || !p.policy.TransientActivity || strings.TrimSpace(line) == "" {
		return
	}
	logging.UpdateActivityWithImportance(opsPurgeAllProcessDefinitionsProgressCommandContext(p.cmd), line, logging.ActivityImportanceWorkflow)
}

// printDurableLineLocked mirrors the shared semantic reporter's quiet failure
// behavior while keeping stage rendering APD-specific.
func (p *opsPurgeAllProcessDefinitionsProgress) printDurableLineLocked(line string, warn bool) {
	if p.policy.Channel.Mode == ops.ProgressModeQuiet && warn {
		printOpsDurableLineDirect(p.cmd, line)
		return
	}
	printOpsDurableLine(p.cmd, line, warn)
}

// opsPurgeAllProcessDefinitionsStageScope maps stable service phases into the
// command vocabulary without parsing service log text.
func opsPurgeAllProcessDefinitionsStageScope(stage ops.StageProgress) (opsSemanticProgressScope, string, bool) {
	switch strings.TrimSpace(stage.Phase) {
	case opsPurgeAllProcessDefinitionsCancelPhase:
		return opsSemanticProgressScope{
			Phase:                     opsPurgeAllProcessDefinitionsCancelPhase,
			ActivityLabel:             "cancelling process-instance root trees",
			CoreResource:              "process-instance tree(s)",
			AffectedResource:          "affected process instances",
			AffectedCoverageAvailable: true,
			SubmittedVerb:             "submitted",
			ConfirmedVerb:             "cancelled",
			FailedVerb:                "failed",
		}, "", true
	case opsPurgeAllProcessDefinitionsDrainPhase:
		return opsSemanticProgressScope{}, "waiting for active process instances to drain", true
	case opsPurgeAllProcessDefinitionsHistoryDeletePhase:
		return opsSemanticProgressScope{
			Phase:                     opsPurgeAllProcessDefinitionsHistoryDeletePhase,
			ActivityLabel:             "deleting process-instance histories",
			CoreResource:              "process-instance tree(s)",
			AffectedResource:          "affected process instances",
			AffectedCoverageAvailable: true,
			SubmittedVerb:             "submitted",
			ConfirmedVerb:             "deleted",
			FailedVerb:                "failed",
		}, "", true
	case processDefinitionDeleteCompletionPhase:
		return opsSemanticProgressScope{
			Phase:         processDefinitionDeleteCompletionPhase,
			ActivityLabel: "deleting process definitions",
			CoreResource:  "process definition(s)",
			SubmittedVerb: "submitted",
			ConfirmedVerb: "deleted",
			FailedVerb:    "failed",
		}, "", true
	default:
		return opsSemanticProgressScope{}, "", false
	}
}

// formatOpsPurgeAllProcessDefinitionsStageAggregate renders one APD stage
// aggregate while keeping planned affected scope separate from completed work.
func formatOpsPurgeAllProcessDefinitionsStageAggregate(record opsPurgeAllProcessDefinitionsStageRecord) string {
	scope := record.Scope
	aggregate := record.Aggregate
	resource := strings.TrimSpace(scope.CoreResource)
	if resource == "" {
		resource = "resource(s)"
	}
	label := strings.TrimSpace(scope.ActivityLabel)
	if label == "" {
		label = strings.TrimSpace(scope.Phase)
	}
	if label == "" {
		label = "progress"
	}
	parts := []string{label}
	if aggregate.Total > 0 {
		parts = append(parts, fmt.Sprintf("%d/%d %s", aggregate.Completed, aggregate.Total, resource))
	} else {
		parts = append(parts, fmt.Sprintf("%d %s completed", aggregate.Completed, resource))
	}
	if aggregate.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", aggregate.Failed))
	}
	if aggregate.Completed > 0 && aggregate.AffectedValid && strings.TrimSpace(scope.AffectedResource) != "" {
		parts = append(parts, fmt.Sprintf("%s: %d", strings.TrimSpace(scope.AffectedResource), aggregate.Affected))
	}
	if record.PlannedAffectedCount != nil {
		parts = append(parts, fmt.Sprintf("affected scope: %d %s", *record.PlannedAffectedCount, opsPurgeAllProcessDefinitionsPlannedAffectedResource))
	}
	return strings.Join(nonEmptyOpsProgressParts(parts), ", ")
}

// formatOpsPurgeAllProcessDefinitionsFinalAggregate renders one close-time
// historical record for every dirty mutation stage in execution order.
func formatOpsPurgeAllProcessDefinitionsFinalAggregate(records []opsPurgeAllProcessDefinitionsStageRecord) string {
	lines := make([]string, 0, len(records))
	for _, record := range records {
		if !record.Dirty {
			continue
		}
		lines = append(lines, formatOpsPurgeAllProcessDefinitionsStageAggregate(record))
	}
	switch len(lines) {
	case 0:
		return ""
	case 1:
		return lines[0]
	default:
		return "stage progress: " + strings.Join(lines, "; ")
	}
}

// formatOpsPurgeAllProcessDefinitionsStageCompletion renders the APD-specific
// aggregate for one verbose or warning completion line.
func formatOpsPurgeAllProcessDefinitionsStageCompletion(record opsPurgeAllProcessDefinitionsStageRecord, completion ops.CompletionProgress) string {
	identity := strings.TrimSpace(completion.Identity)
	if identity == "" {
		identity = "item"
	}
	verb := opsSemanticProgressLifecycleVerb(record.Scope, completion.Disposition)
	line := fmt.Sprintf("%s %s", identity, strings.TrimSpace(verb))
	if detail := strings.TrimSpace(completion.FailureDetail); detail != "" && completion.Disposition == ops.CompletionDispositionFailed {
		line += ": " + detail
	}
	return line + " (" + formatOpsPurgeAllProcessDefinitionsStageAggregate(record) + ")"
}

// opsPurgeAllProcessDefinitionsProgressCommandContext keeps nil command
// contexts from disabling focused coordinator tests.
func opsPurgeAllProcessDefinitionsProgressCommandContext(cmd *cobra.Command) context.Context {
	if cmd == nil || cmd.Context() == nil {
		return context.Background()
	}
	return cmd.Context()
}
