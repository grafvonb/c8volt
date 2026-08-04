// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

const opsDurableMilestoneMinimumElapsed = 30 * time.Second

// opsProgressModeInput captures the command state that decides where progress may be written.
type opsProgressModeInput struct {
	RenderMode RenderMode
	Verbose    bool
	Quiet      bool
	Automation bool
	Debug      bool
}

// opsProgressModeForCommand keeps progress gating tied to existing root flags and render mode.
func opsProgressModeForCommand(cmd *cobra.Command, mode RenderMode) opsProgressModeInput {
	return opsProgressModeInput{
		RenderMode: mode,
		Verbose:    flagVerbose,
		Quiet:      flagQuiet,
		Automation: automationModeEnabled(cmd),
		Debug:      flagDebug,
	}
}

// opsProgressChannelForMode applies the shared stdout-safe progress channel contract.
func opsProgressChannelForMode(input opsProgressModeInput) ops.ProgressChannel {
	switch {
	case input.Quiet:
		return ops.ProgressChannel{Mode: ops.ProgressModeQuiet}
	case input.Automation:
		return ops.ProgressChannel{Mode: ops.ProgressModeAutomation, StructuredReportAllowed: true}
	case input.RenderMode == RenderModeJSON:
		return ops.ProgressChannel{Mode: ops.ProgressModeJSON}
	case input.RenderMode == RenderModeKeysOnly:
		return ops.ProgressChannel{Mode: ops.ProgressModeKeysOnly}
	case input.Debug:
		return ops.ProgressChannel{Mode: ops.ProgressModeDebug, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}
	case input.Verbose:
		return ops.ProgressChannel{Mode: ops.ProgressModeVerbose, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}
	default:
		return ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}
	}
}

type opsProgressMilestonePacer struct {
	minimumElapsed        time.Duration
	now                   func() time.Time
	lastMilestoneAt       time.Time
	lastProgressSignature opsProgressMilestoneSignature
	hasMilestoneSignature bool
}

type opsProgressMilestoneSignature struct {
	Kind             ops.ProgressEventKind
	Phase            string
	CurrentPage      int
	Seen             int
	Selected         int
	Done             int
	Total            int
	CompletedSamples int
}

func newOpsProgressMilestonePacer(now func() time.Time) *opsProgressMilestonePacer {
	if now == nil {
		now = time.Now
	}
	startedAt := now()
	return &opsProgressMilestonePacer{
		minimumElapsed:  opsDurableMilestoneMinimumElapsed,
		now:             now,
		lastMilestoneAt: startedAt,
	}
}

func (p *opsProgressMilestonePacer) AllowDurableMilestone(event ops.ProgressEvent, channel ops.ProgressChannel) bool {
	if p == nil || !opsProgressDurableMilestoneChannelAllowed(channel) {
		return false
	}
	signature, ok := opsProgressMilestoneSignatureForEvent(event)
	if !ok {
		return false
	}
	if p.hasMilestoneSignature && !opsProgressMilestoneSignatureAdvanced(signature, p.lastProgressSignature) {
		return false
	}
	now := p.now()
	if now.Sub(p.lastMilestoneAt) < p.minimumElapsed {
		return false
	}
	p.lastMilestoneAt = now
	p.lastProgressSignature = signature
	p.hasMilestoneSignature = true
	return true
}

func opsProgressMilestoneSignatureAdvanced(current opsProgressMilestoneSignature, previous opsProgressMilestoneSignature) bool {
	if current.Kind != previous.Kind || current.Phase != previous.Phase {
		return true
	}
	switch current.Kind {
	case ops.ProgressEventKindPage:
		return current.CurrentPage > previous.CurrentPage || current.Seen > previous.Seen || current.Selected > previous.Selected
	case ops.ProgressEventKindFrozenScope:
		return current.Done > previous.Done
	case ops.ProgressEventKindETA:
		return current.CompletedSamples > previous.CompletedSamples
	default:
		return current != previous
	}
}

func opsProgressDurableMilestoneChannelAllowed(channel ops.ProgressChannel) bool {
	return channel.Mode == ops.ProgressModeHuman && channel.DurableAllowed && channel.StderrAllowed && !channel.StdoutAllowed
}

func opsProgressMilestoneSignatureForEvent(event ops.ProgressEvent) (opsProgressMilestoneSignature, bool) {
	switch {
	case event.Kind == ops.ProgressEventKindPage && event.Page != nil:
		progress := event.Page
		if progress.CurrentPage <= 0 && progress.Seen <= 0 && progress.Selected <= 0 {
			return opsProgressMilestoneSignature{}, false
		}
		return opsProgressMilestoneSignature{
			Kind:        ops.ProgressEventKindPage,
			Phase:       strings.TrimSpace(progress.Phase),
			CurrentPage: progress.CurrentPage,
			Seen:        progress.Seen,
			Selected:    progress.Selected,
		}, true
	case event.Kind == ops.ProgressEventKindFrozenScope && event.FrozenScope != nil:
		progress := event.FrozenScope
		if progress.Done <= 0 {
			return opsProgressMilestoneSignature{}, false
		}
		return opsProgressMilestoneSignature{
			Kind:  ops.ProgressEventKindFrozenScope,
			Phase: strings.TrimSpace(progress.Phase),
			Done:  progress.Done,
			Total: progress.Total,
		}, true
	case event.Kind == ops.ProgressEventKindETA && event.ETA != nil:
		progress := event.ETA
		if progress.CompletedSamples <= 0 {
			return opsProgressMilestoneSignature{}, false
		}
		return opsProgressMilestoneSignature{
			Kind:             ops.ProgressEventKindETA,
			Phase:            strings.TrimSpace(progress.Phase),
			CompletedSamples: progress.CompletedSamples,
			Total:            progress.Total,
		}, true
	default:
		return opsProgressMilestoneSignature{}, false
	}
}

// formatOpsTotalCertainty renders a resource count without implying unavailable or approximate totals are exact.
func formatOpsTotalCertainty(total *int64, kind ops.TotalCertainty, resource string) string {
	label := strings.TrimSpace(resource)
	if label == "" {
		label = "resource(s)"
	}
	switch kind {
	case ops.TotalCertaintyExact:
		if total != nil {
			return fmt.Sprintf("%d %s", *total, label)
		}
	case ops.TotalCertaintyLowerBound:
		if total != nil {
			return fmt.Sprintf("%d+ %s", *total, label)
		}
	case ops.TotalCertaintyEstimated:
		if total != nil {
			return fmt.Sprintf("about %d %s", *total, label)
		}
	}
	return fmt.Sprintf("unknown %s", label)
}

// formatOpsPageCount renders page context only when the workflow has usable page metadata.
func formatOpsPageCount(current int, pageCount *int64, kind ops.PageCountKind) string {
	switch {
	case current <= 0:
		return ""
	case pageCount == nil || kind == ops.PageCountKindUnknown:
		return fmt.Sprintf("page %d", current)
	case kind == ops.PageCountKindEstimated:
		return fmt.Sprintf("page %d/~%d", current, *pageCount)
	default:
		return fmt.Sprintf("page %d/%d", current, *pageCount)
	}
}

// formatOpsPageProgress renders compact discovery progress for human activity updates.
func formatOpsPageProgress(progress ops.PageProgress, resource string) string {
	parts := []string{strings.TrimSpace(progress.Phase)}
	pageText := formatOpsPageCount(progress.CurrentPage, progress.PageCount, progress.PageCountKind)
	if pageText != "" {
		parts = append(parts, pageText)
	}
	seenTotal := progress.PageCount
	if progress.Seen > 0 {
		parts = append(parts, fmt.Sprintf("%d seen", progress.Seen))
	}
	if progress.Selected > 0 && progress.Selected != progress.Seen {
		parts = append(parts, fmt.Sprintf("%d selected", progress.Selected))
	}
	if progress.LimitReached {
		parts = append(parts, "user-limited")
	}
	if len(parts) == 1 && seenTotal == nil && resource != "" {
		parts = append(parts, strings.TrimSpace(resource))
	}
	return strings.Join(nonEmptyOpsProgressParts(parts), ", ")
}

// formatOpsPreflightScope renders compact preflight lines before expensive ops-scale work begins.
func formatOpsPreflightScope(scope ops.PreflightScope) []string {
	rendered := formatOpsPreflightScopeLines(scope)
	lines := make([]string, 0, len(rendered))
	for _, line := range rendered {
		lines = append(lines, line.Text)
	}
	return lines
}

type opsPreflightRenderedLine struct {
	Text string
	Warn bool
}

func formatOpsPreflightScopeLines(scope ops.PreflightScope) []opsPreflightRenderedLine {
	resource := opsPreflightResourceLabelsFor(scope.CoreResource)
	count := formatOpsPreflightTotal(scope.Total, scope.TotalKind, resource)
	selector := strings.TrimSpace(scope.SelectorSummary)
	if selector == "" {
		selector = "selected scope"
	}
	label := opsPreflightLabel(scope)
	scopeText := fmt.Sprintf("%s scope: %s matched %s", label, selector, count)
	if opsPreflightSelectorRedundant(label, selector) {
		scopeText = fmt.Sprintf("%s scope: matched %s", label, count)
	}
	parts := []string{scopeText}
	if scope.PageSize > 0 {
		parts = append(parts, fmt.Sprintf("page size: %d", scope.PageSize))
	}
	if pageText := formatOpsPreflightPageContext(scope.PageCount, scope.PageCountKind); pageText != "" {
		parts = append(parts, pageText)
	}
	lines := []opsPreflightRenderedLine{{Text: strings.Join(nonEmptyOpsProgressParts(parts), "; ")}}
	consequence := formatOpsConsequenceSummary(scope.ConsequenceSummary)
	if consequence != "" {
		warn := opsPreflightConsequenceShouldWarn(scope.ConsequenceSummary)
		lines = append(lines, opsPreflightRenderedLine{
			Text: formatOpsConsequenceLine(label, consequence, scope.ConsequenceSummary, warn),
			Warn: warn,
		})
	}
	return lines
}

func opsPreflightLabel(scope ops.PreflightScope) string {
	command := strings.ToLower(strings.TrimSpace(scope.Command))
	selector := strings.ToLower(strings.TrimSpace(scope.SelectorSummary))
	switch {
	case strings.Contains(command, "slow-process-instances"):
		return "slow analysis"
	case strings.Contains(command, "process-instances-with-incidents") || strings.Contains(selector, "process-instances-with-incidents"):
		return "incident purge"
	case strings.Contains(command, "orphan-process-instances") || strings.Contains(selector, "orphan-process-instances"):
		return "orphan purge"
	case strings.Contains(command, "retention-policy") || strings.Contains(selector, "retention-policy"):
		return "retention cleanup"
	case strings.Contains(command, "all-process-definitions") || strings.Contains(selector, "all-process-definitions"):
		return "process-definition purge"
	case strings.Contains(command, "repair incident") || selector == "incident repair":
		return "incident repair"
	case strings.Contains(command, "repair process-instance") || selector == "process-instance repair":
		return "process-instance repair"
	case strings.Contains(command, "get process-instance"):
		return "process-instance search"
	case strings.Contains(command, "get incident"):
		return "incident search"
	case strings.Contains(command, "get job"):
		return "job search"
	case strings.Contains(command, "get element"):
		return "element search"
	case strings.Contains(command, "get process-definition"):
		return "process-definition search"
	case strings.Contains(selector, "cancel"):
		return "process-instance cancel"
	case strings.Contains(selector, "delete"):
		return "process-instance delete"
	case strings.Contains(selector, "mutation"):
		return "process-instance mutation"
	default:
		label := strings.TrimSpace(scope.Command)
		if label == "" {
			label = strings.TrimSpace(scope.SelectorSummary)
		}
		if label == "" {
			label = "scope check"
		}
		return strings.TrimSpace(label)
	}
}

func opsPreflightSelectorRedundant(label string, selector string) bool {
	label = opsPreflightComparableLabel(label)
	selector = opsPreflightComparableLabel(selector)
	if label == "" || selector == "" {
		return false
	}
	return label == selector ||
		strings.Contains(selector, label) ||
		strings.Contains(label, selector)
}

func opsPreflightComparableLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("-", "", "_", "", " ", "")
	return replacer.Replace(value)
}

func formatOpsConsequenceLine(label string, consequence string, summary ops.ConsequenceSummary, warn bool) string {
	if warn {
		risk := strings.ToLower(strings.TrimSpace(summary.RiskSummary))
		work := strings.TrimSpace(summary.WorkSummary)
		if work == "" {
			work = consequence
		}
		switch {
		case strings.Contains(risk, "expensive"):
			return fmt.Sprintf("%s is expensive: %s", label, work)
		case strings.Contains(risk, "irreversible"):
			return fmt.Sprintf("%s is irreversible: %s", label, work)
		case strings.Contains(risk, "repair"):
			return fmt.Sprintf("%s changes state: %s", label, work)
		case strings.Contains(risk, "destructive") || strings.Contains(risk, "delete") || strings.Contains(risk, "purge") || strings.Contains(risk, "mutation"):
			return fmt.Sprintf("%s is destructive: %s", label, work)
		}
	}
	return fmt.Sprintf("%s: %s", label, consequence)
}

func opsPreflightConsequenceShouldWarn(summary ops.ConsequenceSummary) bool {
	risk := strings.ToLower(strings.TrimSpace(summary.RiskSummary))
	return strings.Contains(risk, "expensive") ||
		strings.Contains(risk, "destructive") ||
		strings.Contains(risk, "delete") ||
		strings.Contains(risk, "purge") ||
		strings.Contains(risk, "repair") ||
		strings.Contains(risk, "mutation") ||
		strings.Contains(risk, "irreversible")
}

func printOpsPreflightLines(cmd *cobra.Command, scope ops.PreflightScope) {
	if cmd == nil {
		return
	}
	for _, line := range formatOpsPreflightScopeLines(scope) {
		printOpsDurableLine(cmd, line.Text, line.Warn)
	}
}

func printOpsDurableLine(cmd *cobra.Command, line string, warn bool) {
	if cmd == nil || strings.TrimSpace(line) == "" {
		return
	}
	if log, err := logging.FromContext(cmd.Context()); err == nil {
		if warn {
			log.Warn(line)
		} else {
			log.Info(line)
		}
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), line)
}

type opsPreflightResourceLabels struct {
	Singular string
	Plural   string
}

func opsPreflightResourceLabelsFor(coreResource string) opsPreflightResourceLabels {
	switch strings.TrimSpace(coreResource) {
	case "", "process_instance":
		return opsPreflightResourceLabels{Singular: "process instance", Plural: "process instances"}
	case "process_definition":
		return opsPreflightResourceLabels{Singular: "process definition", Plural: "process definitions"}
	case "incident":
		return opsPreflightResourceLabels{Singular: "incident", Plural: "incidents"}
	case "job":
		return opsPreflightResourceLabels{Singular: "job", Plural: "jobs"}
	case "element":
		return opsPreflightResourceLabels{Singular: "element", Plural: "elements"}
	default:
		label := strings.ReplaceAll(strings.TrimSpace(coreResource), "_", " ")
		if label == "" {
			label = "resource"
		}
		return opsPreflightResourceLabels{Singular: label, Plural: label + "s"}
	}
}

func formatOpsPreflightTotal(total *int64, kind ops.TotalCertainty, resource opsPreflightResourceLabels) string {
	plural := strings.TrimSpace(resource.Plural)
	if plural == "" {
		plural = "resources"
	}
	singular := strings.TrimSpace(resource.Singular)
	if singular == "" {
		singular = strings.TrimSuffix(plural, "s")
	}
	switch kind {
	case ops.TotalCertaintyExact:
		if total != nil {
			switch *total {
			case 0:
				return "no " + plural
			case 1:
				return "1 " + singular
			default:
				return fmt.Sprintf("%d %s", *total, plural)
			}
		}
	case ops.TotalCertaintyLowerBound:
		if total != nil {
			return fmt.Sprintf("at least %d %s", *total, plural)
		}
	case ops.TotalCertaintyEstimated:
		if total != nil {
			return fmt.Sprintf("about %d %s", *total, plural)
		}
	}
	return "an unknown number of " + plural
}

// formatOpsPreflightPageContext labels page count certainty without promoting lower-bound totals to exact.
func formatOpsPreflightPageContext(pageCount *int64, kind ops.PageCountKind) string {
	if pageCount == nil || kind == ops.PageCountKindUnknown {
		return ""
	}
	switch kind {
	case ops.PageCountKindEstimated:
		return fmt.Sprintf("discovery pages: at least %d", *pageCount)
	default:
		return fmt.Sprintf("discovery pages: %d", *pageCount)
	}
}

// formatOpsConsequenceSummary joins structured consequence parts into one operator-facing sentence.
func formatOpsConsequenceSummary(summary ops.ConsequenceSummary) string {
	return strings.Join(nonEmptyOpsProgressParts([]string{summary.WorkSummary, summary.RiskSummary}), "; ")
}

// formatOpsFrozenScopeProgress renders exact done/total counters and gates ETA until a remaining duration exists.
func formatOpsFrozenScopeProgress(progress ops.FrozenScopeProgress) string {
	resource := strings.TrimSpace(progress.CoreResource)
	if resource == "" {
		resource = "resource(s)"
	}
	parts := []string{
		strings.TrimSpace(progress.Phase),
		fmt.Sprintf("%d/%d %s", progress.Done, progress.Total, resource),
	}
	if progress.Elapsed > 0 {
		if progress.Total > 0 {
			parts = append(parts, fmt.Sprintf("%.1f%%", float64(progress.Done)*100/float64(progress.Total)))
		}
		parts = append(parts, fmt.Sprintf("%s elapsed", progress.Elapsed.Round(time.Second)))
	}
	if progress.Rate != nil && *progress.Rate > 0 {
		parts = append(parts, fmt.Sprintf("~%.1f/s", *progress.Rate))
	}
	if progress.ETA != nil && *progress.ETA > 0 {
		parts = append(parts, fmt.Sprintf("~%s remaining", progress.ETA.Round(time.Second)))
	}
	if progress.Errors > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", progress.Errors))
	}
	return strings.Join(nonEmptyOpsProgressParts(parts), ", ")
}

// opsETAAllowed returns true only when ETA has enough exact-scope timing data to be useful.
func opsETAAllowed(window ops.ETASampleWindow) bool {
	return window.MinimumSamplesMet && window.Total > 0 && window.CompletedSamples > 0 && window.Elapsed > 0 && window.Remaining != nil && *window.Remaining > 0
}

// formatOpsETASampleWindow renders standalone ETA samples for workflows that emit timing facts separately from counters.
func formatOpsETASampleWindow(window ops.ETASampleWindow) string {
	if !opsETAAllowed(window) {
		return ""
	}
	parts := []string{
		strings.TrimSpace(window.Phase),
		fmt.Sprintf("%d/%d sample(s)", window.CompletedSamples, window.Total),
	}
	if window.Elapsed > 0 {
		parts = append(parts, fmt.Sprintf("%s elapsed", window.Elapsed.Round(time.Second)))
	}
	if window.Rate != nil && *window.Rate > 0 {
		parts = append(parts, fmt.Sprintf("~%.1f/s", *window.Rate))
	}
	if window.Remaining != nil && *window.Remaining > 0 {
		parts = append(parts, fmt.Sprintf("~%s remaining", window.Remaining.Round(time.Second)))
	}
	return strings.Join(nonEmptyOpsProgressParts(parts), ", ")
}

// nonEmptyOpsProgressParts trims empty formatter fragments before joining human progress text.
func nonEmptyOpsProgressParts(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
