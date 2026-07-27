// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

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
	resource := "process instance(s)"
	if strings.TrimSpace(scope.CoreResource) != "" && scope.CoreResource != "process_instance" {
		resource = strings.ReplaceAll(scope.CoreResource, "_", " ") + "(s)"
	}
	count := formatOpsTotalCertainty(scope.Total, scope.TotalKind, resource)
	selector := strings.TrimSpace(scope.SelectorSummary)
	if selector == "" {
		selector = "selected scope"
	}
	parts := []string{fmt.Sprintf("preflight: %s matches %s", selector, count)}
	if scope.PageSize > 0 {
		parts = append(parts, fmt.Sprintf("page size %d", scope.PageSize))
	}
	if pageText := formatOpsPreflightPageContext(scope.PageCount, scope.PageCountKind); pageText != "" {
		parts = append(parts, pageText)
	}
	lines := []string{strings.Join(nonEmptyOpsProgressParts(parts), "; ")}
	consequence := formatOpsConsequenceSummary(scope.ConsequenceSummary)
	if consequence != "" {
		lines = append(lines, "preflight: "+consequence)
	}
	return lines
}

// formatOpsPreflightPageContext labels page count certainty without promoting lower-bound totals to exact.
func formatOpsPreflightPageContext(pageCount *int64, kind ops.PageCountKind) string {
	if pageCount == nil || kind == ops.PageCountKindUnknown {
		return ""
	}
	switch kind {
	case ops.PageCountKindEstimated:
		return fmt.Sprintf("discovery will require at least %d page(s)", *pageCount)
	default:
		return fmt.Sprintf("discovery will require %d page(s)", *pageCount)
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
