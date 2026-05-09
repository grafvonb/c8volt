// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// processInstanceContinuationState is the command-level decision made after a
// process-instance search page has been fetched and, for mutating commands,
// processed. The state is deliberately separate from the service-level overflow
// metadata: overflow says what Operate can prove about more matches, while the
// continuation state says what the CLI should do next for this invocation.
type processInstanceContinuationState string

const (
	// processInstanceContinuationPrompt means another page is known to exist,
	// but interactive mode must ask before more instances are shown or changed.
	processInstanceContinuationPrompt processInstanceContinuationState = "prompt"
	// processInstanceContinuationAutoContinue means the command may continue
	// paging without a prompt because the caller chose JSON, dry-run, or
	// another non-interactive/confirmed path.
	processInstanceContinuationAutoContinue processInstanceContinuationState = "auto_continue"
	// processInstanceContinuationCompleted means the current page exhausted the
	// result set, so the command can finish with a complete result.
	processInstanceContinuationCompleted processInstanceContinuationState = "completed"
	// processInstanceContinuationPartialComplete records a deliberate user stop
	// after at least one page was processed. This matters for cancel/delete:
	// previous pages may already have changed process instances.
	processInstanceContinuationPartialComplete processInstanceContinuationState = "partial_complete"
	// processInstanceContinuationWarningStop is used when the backend cannot
	// prove whether more matches exist. The CLI stops instead of pretending the
	// result is complete.
	processInstanceContinuationWarningStop processInstanceContinuationState = "warning_stop"
	// processInstanceContinuationLimitReached means the local --limit boundary,
	// not the backend result set, stopped collection or mutation.
	processInstanceContinuationLimitReached processInstanceContinuationState = "limit_reached"
)

// processInstanceProgressSummary describes the current pagination state for user-facing progress output.
//
// It is used to render one-line progress diagnostics (in verbose mode) and to drive continuation behavior
// such as prompting, auto-continue, warning stop, and partial-complete reporting.
type processInstanceProgressSummary struct {
	// PageSize is the configured request size used for the current page fetch.
	PageSize int32
	// CurrentPageCount is the number of process instances returned on this page.
	CurrentPageCount int
	// CumulativeCount is the total number of process instances processed/collected so far.
	CumulativeCount int
	// OverflowState indicates whether additional matching items are known, unknown, or exhausted.
	OverflowState process.ProcessInstanceOverflowState
	// ContinuationState determines the next paging action (prompt/auto-continue/complete/etc.).
	ContinuationState processInstanceContinuationState
}

// processInstancePageImpact captures per-page impact counts used by cancel/delete paging prompts.
//
// These values are accumulated across pages to present users with a continuation prompt that reflects
// both the visible page size and the real operational impact when dependencies are included.
type processInstancePageImpact struct {
	// Requested is the raw number of keys selected from the current search page.
	Requested int
	// Affected is the expanded number of instances impacted after dependency resolution.
	Affected int
	// Roots is the number of root instances in the expanded impact set.
	Roots int
}

// processInstancePageActionResult is the per-page result produced by mutating
// process-instance commands. It keeps operational impact, reporters, and dry-run
// previews together so the paging loop can aggregate them without knowing the
// command-specific cancel/delete implementation details.
type processInstancePageActionResult struct {
	Impact        processInstancePageImpact
	Reports       []process.Reporter
	DryRunPreview *processInstanceDryRunPreview
}

// processInstancePageActionResults is the accumulated result returned from a
// paged cancel/delete operation after all selected pages are processed.
type processInstancePageActionResults struct {
	Reports        []process.Reporter
	DryRunPreviews []processInstanceDryRunPreview
}

// pickPISearchSize normalizes the legacy global batch-size flag to the maximum
// supported Operate page size whenever the flag is unset or out of range. The
// flag validation path reports invalid user input separately; this guard keeps
// internal callers from creating unusable page requests.
func pickPISearchSize() int32 {
	if flagGetPISize <= 0 || flagGetPISize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return flagGetPISize
}

// resolvePISearchSize applies the operator-facing precedence for paging:
// command flag first, config default second, and the product maximum as a safe
// fallback. This keeps ad-hoc commands overrideable without ignoring fleet-wide
// defaults configured for repeated operations.
func resolvePISearchSize(cmd *cobra.Command, cfg *config.Config) int32 {
	if cmd != nil && cmd.Flags().Changed("batch-size") {
		return pickPISearchSize()
	}
	if cfg != nil && cfg.App.ProcessInstancePageSize > 0 && cfg.App.ProcessInstancePageSize <= consts.MaxPISearchSize {
		return cfg.App.ProcessInstancePageSize
	}
	return consts.MaxPISearchSize
}

// newPISearchPageRequest builds the process-instance page request for the current command and config.
func newPISearchPageRequest(cmd *cobra.Command, cfg *config.Config, from int32) process.ProcessInstancePageRequest {
	return process.ProcessInstancePageRequest{
		From: from,
		Size: resolvePISearchSize(cmd, cfg),
	}
}

// pickPIContinuationState translates backend overflow metadata into a CLI
// continuation decision. A known next page either prompts or auto-continues
// depending on command mode; an indeterminate backend response is treated as a
// warning stop because continuing could imply a complete result that Operate did
// not actually guarantee.
func pickPIContinuationState(overflow process.ProcessInstanceOverflowState, autoConfirm bool) processInstanceContinuationState {
	switch overflow {
	case process.ProcessInstanceOverflowStateHasMore:
		if autoConfirm {
			return processInstanceContinuationAutoContinue
		}
		return processInstanceContinuationPrompt
	case process.ProcessInstanceOverflowStateIndeterminate:
		return processInstanceContinuationWarningStop
	default:
		return processInstanceContinuationCompleted
	}
}

// newPIProgressSummary converts page metadata into user-facing continuation
// progress and gives the local --limit switch priority over backend overflow.
// The same summary drives verbose output and the continuation switch in the
// paging loops, so operators see the reason for the behavior the command takes.
func newPIProgressSummary(page process.ProcessInstancePage, cumulative int, autoConfirm bool) processInstanceProgressSummary {
	continuationState := pickPIContinuationState(page.OverflowState, autoConfirm)
	if isPILimitReached(cumulative) {
		continuationState = processInstanceContinuationLimitReached
	}
	return processInstanceProgressSummary{
		PageSize:          page.Request.Size,
		CurrentPageCount:  len(page.Items),
		CumulativeCount:   cumulative,
		OverflowState:     page.OverflowState,
		ContinuationState: continuationState,
	}
}

// isPILimitReached reports whether the locally requested cap has been reached.
// It intentionally checks the cumulative count after page limiting so verbose
// output can distinguish an operator limit from a naturally exhausted result.
func isPILimitReached(cumulative int) bool {
	return flagGetPILimit > 0 && cumulative >= int(flagGetPILimit)
}

// limitPIItems trims a page to the remaining --limit budget before rendering or
// mutation. This prevents later pages, and the tail of the current page, from
// being touched once the operator requested a bounded run.
func limitPIItems(items []process.ProcessInstance, cumulative int) []process.ProcessInstance {
	if flagGetPILimit <= 0 {
		return items
	}
	remaining := int(flagGetPILimit) - cumulative
	if remaining <= 0 {
		return nil
	}
	if len(items) > remaining {
		return items[:remaining]
	}
	return items
}

// limitPIPageItems preserves page metadata while applying the local item limit
// to the page payload. The overflow and cursor metadata still describe the
// backend result, which is needed for accurate partial-complete messaging.
func limitPIPageItems(page process.ProcessInstancePage, cumulative int) process.ProcessInstancePage {
	page.Items = limitPIItems(page.Items, cumulative)
	return page
}

// shouldRenderPISearchPageIncrementally keeps interactive one-line and key-only
// output responsive by printing each page before asking whether to continue. JSON
// waits for a complete collection so the command can emit a single valid JSON
// document, and auto-confirmed runs avoid incremental rendering because they do
// not need a prompt boundary.
func shouldRenderPISearchPageIncrementally(cmd *cobra.Command) bool {
	if flagCmdAutoConfirm {
		return false
	}
	mode := pickMode()
	if automationModeEnabled(cmd) {
		return mode == RenderModeOneLine || mode == RenderModeKeysOnly
	}
	return mode == RenderModeOneLine || mode == RenderModeKeysOnly
}

// shouldAutoContinuePISearchPages reports whether paged process-instance search should
// consume additional pages without interactive confirmation.
func shouldAutoContinuePISearchPages(cmd *cobra.Command) bool {
	return shouldImplicitlyConfirm(cmd) || pickMode() == RenderModeJSON
}

// isCmdAborted recognizes both the direct abort sentinel and the wrapped local
// precondition form used by prompt handling. Paged mutating commands need this
// distinction to report partial completion instead of treating an intentional
// user stop like an operational failure.
func isCmdAborted(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrCmdAborted) {
		return true
	}
	if errors.Is(err, ferrors.ErrLocalPrecondition) && strings.Contains(err.Error(), ErrCmdAborted.Error()) {
		return true
	}
	return false
}

// canUsePIReportedTotal reports whether server total metadata still matches the
// result the command will present. Client-side relationship and incident filters
// run after the page is fetched, so totals from Operate would overstate the final
// output for those modes.
func canUsePIReportedTotal() bool {
	return !(flagGetPIChildrenOnly || flagGetPIRootsOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPIDirectIncidentsOnly || flagGetPINoIncidentsOnly || hasPIIncidentDetailFilters())
}

// canUsePIExactReportedTotal guards the fast total path used by count-style
// commands. Lower-bound totals are useful for progress, but only an exact total
// can replace page iteration when the operator asks for a precise count.
func canUsePIExactReportedTotal(page process.ProcessInstancePage) bool {
	return canUsePIReportedTotal() &&
		page.ReportedTotal != nil &&
		page.ReportedTotal.Kind == process.ProcessInstanceReportedTotalKindExact
}

// nextPISearchPageRequest advances using cursor paging when Operate provides an
// end cursor, and falls back to offset paging for older or compatibility paths.
// The next request is rebuilt instead of mutating the current one so any changed
// command/config page-size rules are applied consistently.
func nextPISearchPageRequest(cmd *cobra.Command, cfg *config.Config, current process.ProcessInstancePageRequest, page process.ProcessInstancePage) process.ProcessInstancePageRequest {
	if page.EndCursor != "" {
		req := newPISearchPageRequest(cmd, cfg, 0)
		req.After = page.EndCursor
		return req
	}
	return newPISearchPageRequest(cmd, cfg, current.From+int32(len(page.Items)))
}

// startCommandActivity wraps long page fetches with the shared activity
// indicator while keeping nil-command tests cheap and deterministic.
func startCommandActivity(cmd *cobra.Command, msg string) func() {
	if cmd == nil {
		return func() {}
	}
	return logging.StartActivity(cmd.Context(), msg)
}

// printPISearchProgress writes verbose one-line progress to stderr for normal
// command execution. It mirrors logPISearchProgress, but keeps human-visible
// diagnostics out of stdout so JSON/key output remains parseable.
func printPISearchProgress(cmd *cobra.Command, summary processInstanceProgressSummary) {
	if cmd == nil || !flagVerbose || pickMode() != RenderModeOneLine {
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), formatPISearchProgress(summary))
}

// logPISearchProgress sends the same verbose progress text to the command
// logger. This gives automated operators a stable diagnostic line even when
// stderr is not collected.
func logPISearchProgress(cmd *cobra.Command, log *slog.Logger, summary processInstanceProgressSummary) {
	if cmd == nil || log == nil || !flagVerbose || pickMode() != RenderModeOneLine {
		return
	}
	log.InfoContext(cmd.Context(), formatPISearchProgress(summary))
}

// formatPISearchProgress produces the compact progress sentence shared by
// stderr and structured logging. The wording is intentionally operational:
// page size, current page count, cumulative count, whether more matches may
// exist, and the next action the CLI will take.
func formatPISearchProgress(summary processInstanceProgressSummary) string {
	line := fmt.Sprintf("page size: %d, current page: %d, total so far: %d, more matches: %s, next step: %s",
		summary.PageSize,
		summary.CurrentPageCount,
		summary.CumulativeCount,
		describePIOverflowState(summary.OverflowState),
		describePIContinuationState(summary.ContinuationState),
	)
	if detail := describePIProgressDetail(summary); detail != "" {
		line += ", " + detail
	}
	return line
}

// describePIOverflowState maps service overflow metadata to wording that fits
// the "more matches" field in verbose progress output.
func describePIOverflowState(state process.ProcessInstanceOverflowState) string {
	switch state {
	case process.ProcessInstanceOverflowStateHasMore:
		return "yes"
	case process.ProcessInstanceOverflowStateIndeterminate:
		return "unknown"
	default:
		return "no"
	}
}

// describePIContinuationState maps the internal continuation decision to the
// short "next step" token shown in verbose output and logs.
func describePIContinuationState(state processInstanceContinuationState) string {
	switch state {
	case processInstanceContinuationPrompt:
		return "prompt"
	case processInstanceContinuationAutoContinue:
		return "auto-continue"
	case processInstanceContinuationPartialComplete:
		return "partial-complete"
	case processInstanceContinuationWarningStop:
		return "warning-stop"
	case processInstanceContinuationLimitReached:
		return "limit-reached"
	default:
		return "complete"
	}
}

// describePIProgressDetail adds the operator-facing reason behind the next
// action. The message is deliberately explicit for partial, warning, and limit
// stops because those are the cases where operators need to know whether
// remaining matching instances may have been left untouched.
func describePIProgressDetail(summary processInstanceProgressSummary) string {
	switch summary.ContinuationState {
	case processInstanceContinuationPrompt:
		return "detail: prompt before processing the next page"
	case processInstanceContinuationAutoContinue:
		return "detail: auto-confirm will continue with the next page"
	case processInstanceContinuationPartialComplete:
		return fmt.Sprintf("detail: stopped after %d processed process instance(s); remaining matches were left untouched", summary.CumulativeCount)
	case processInstanceContinuationWarningStop:
		return fmt.Sprintf("warning: stopped after %d processed process instance(s) because more matching process instances may remain", summary.CumulativeCount)
	case processInstanceContinuationLimitReached:
		return fmt.Sprintf("detail: stopped after reaching limit of %d process instance(s)", flagGetPILimit)
	default:
		return "detail: no additional matching process instances remain"
	}
}

// processPISearchPagesWithAction is the shared paging loop for cancel/delete
// style operations that search first and then act page by page. It centralizes
// limit enforcement, continuation prompts, dry-run aggregation, and partial-stop
// reporting so mutating commands do not each encode their own paging contract.
func processPISearchPagesWithAction(
	cmd *cobra.Command,
	cli process.API,
	cfg *config.Config,
	filter process.ProcessInstanceFilter,
	processPage func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error),
) (processInstancePageActionResults, error) {
	pageReq := newPISearchPageRequest(cmd, cfg, 0)
	cumulative := 0
	cumulativeAffected := 0
	firstPage := true
	var results processInstancePageActionResults

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return processInstancePageActionResults{}, err
		}
		if len(page.Items) == 0 {
			if cumulative == 0 {
				renderOutputLine(cmd, "found: %d", 0)
			}
			return results, nil
		}

		limitedPage := limitPIPageItems(page, cumulative)
		result, err := processPage(limitedPage, firstPage)
		if err != nil {
			if !firstPage && isCmdAborted(err) {
				printPISearchProgress(cmd, processInstanceProgressSummary{
					PageSize:          page.Request.Size,
					CurrentPageCount:  len(limitedPage.Items),
					CumulativeCount:   cumulative,
					OverflowState:     page.OverflowState,
					ContinuationState: processInstanceContinuationPartialComplete,
				})
				return results, nil
			}
			return processInstancePageActionResults{}, err
		}

		impact := result.Impact
		results.Reports = append(results.Reports, result.Reports...)
		if result.DryRunPreview != nil {
			results.DryRunPreviews = append(results.DryRunPreviews, *result.DryRunPreview)
		}
		cumulative += len(limitedPage.Items)
		if impact.Affected > 0 {
			cumulativeAffected += impact.Affected
		} else {
			cumulativeAffected += len(limitedPage.Items)
		}
		summary := newPIProgressSummary(limitedPage, cumulative, flagDryRun || shouldAutoContinuePISearchPages(cmd))
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return results, nil
		case processInstanceContinuationAutoContinue:
			firstPage = false
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
			continue
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Processed %d process instance(s) on this page (%d requested so far, %d including dependencies). More matching process instances remain. Continue?", summary.CurrentPageCount, summary.CumulativeCount, cumulativeAffected)
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					return results, nil
				}
				return processInstancePageActionResults{}, err
			}
			firstPage = false
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
		}
	}
}
