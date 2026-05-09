// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

func resolveIncidentSearchSize(cmd *cobra.Command, cfg *config.Config) int32 {
	if cmd != nil && cmd.Flags().Changed("batch-size") {
		return pickIncidentSearchSize()
	}
	if cfg != nil && cfg.App.ProcessInstancePageSize > 0 && cfg.App.ProcessInstancePageSize <= consts.MaxPISearchSize {
		return cfg.App.ProcessInstancePageSize
	}
	return consts.MaxPISearchSize
}

func pickIncidentSearchSize() int32 {
	if flagGetIncidentSize <= 0 || flagGetIncidentSize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return flagGetIncidentSize
}

func newIncidentSearchPageRequest(cmd *cobra.Command, cfg *config.Config, from int32) process.IncidentPageRequest {
	return process.IncidentPageRequest{
		From: from,
		Size: resolveIncidentSearchSize(cmd, cfg),
	}
}

func shouldRenderIncidentSearchPageIncrementally(cmd *cobra.Command) bool {
	if flagCmdAutoConfirm {
		return false
	}
	mode := pickMode()
	if automationModeEnabled(cmd) {
		return mode == RenderModeOneLine || mode == RenderModeKeysOnly
	}
	return mode == RenderModeOneLine || mode == RenderModeKeysOnly
}

func shouldAutoContinueIncidentSearchPages(cmd *cobra.Command) bool {
	return shouldImplicitlyConfirm(cmd) || pickMode() == RenderModeJSON
}

func limitIncidentItems(items []process.ProcessInstanceIncidentDetail, cumulative int) []process.ProcessInstanceIncidentDetail {
	if flagGetIncidentLimit <= 0 {
		return items
	}
	remaining := int(flagGetIncidentLimit) - cumulative
	if remaining <= 0 {
		return nil
	}
	if len(items) > remaining {
		return items[:remaining]
	}
	return items
}

func isIncidentLimitReached(cumulative int) bool {
	return flagGetIncidentLimit > 0 && cumulative >= int(flagGetIncidentLimit)
}

func nextIncidentSearchPageRequest(cmd *cobra.Command, cfg *config.Config, current process.IncidentPageRequest, page process.IncidentPage) process.IncidentPageRequest {
	if page.EndCursor != "" {
		req := newIncidentSearchPageRequest(cmd, cfg, 0)
		req.After = page.EndCursor
		return req
	}
	return newIncidentSearchPageRequest(cmd, cfg, current.From+current.Size)
}

func incidentSearchContinuationState(page process.IncidentPage, cumulative int, autoContinue bool) processInstanceContinuationState {
	if isIncidentLimitReached(cumulative) {
		return processInstanceContinuationLimitReached
	}
	return pickPIContinuationState(page.OverflowState, autoContinue)
}

func renderIncidentSearchPage(cmd *cobra.Command, items []process.ProcessInstanceIncidentDetail) error {
	switch pickMode() {
	case RenderModeKeysOnly:
		for _, item := range items {
			renderOutputLine(cmd, "%s", item.IncidentKey)
		}
	default:
		for _, item := range items {
			renderOutputLine(cmd, "%s", incidentListHumanLineWithMessageLimit(item, flagGetIncidentMessageLimit))
		}
	}
	return nil
}

func canUseIncidentExactReportedTotal(page process.IncidentPage) bool {
	return page.ReportedTotal != nil && page.ReportedTotal.Kind == process.IncidentReportedTotalKindExact
}

func searchIncidentsTotal(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.IncidentFilter) (int64, error) {
	pageReq := newIncidentSearchPageRequest(cmd, cfg, 0)
	total := int64(0)
	for {
		page, err := cli.SearchIncidentsPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return 0, err
		}
		if canUseIncidentExactReportedTotal(page) {
			return page.ReportedTotal.Count, nil
		}
		total += int64(len(page.Items))
		if page.OverflowState == process.ProcessInstanceOverflowStateNoMore {
			return total, nil
		}
		pageReq = nextIncidentSearchPageRequest(cmd, cfg, pageReq, page)
	}
}

// searchIncidentsWithPaging runs the list/search path for `get incident`.
// Human and keys-only output may render page-by-page in interactive mode; JSON
// collects all bounded results so the command emits one valid document.
func searchIncidentsWithPaging(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.IncidentFilter) (process.Incidents, bool, error) {
	pageReq := newIncidentSearchPageRequest(cmd, cfg, 0)
	var collected process.Incidents
	incremental := shouldRenderIncidentSearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinueIncidentSearchPages(cmd)
	processedTotal := 0
	printFoundAndReturn := func() (process.Incidents, bool, error) {
		if incremental {
			if pickMode() == RenderModeOneLine {
				renderOutputLine(cmd, "found: %d", processedTotal)
			}
			return process.Incidents{}, true, nil
		}
		return collected, false, nil
	}

	for {
		page, err := cli.SearchIncidentsPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return process.Incidents{}, false, err
		}
		items := limitIncidentItems(page.Items, processedTotal)
		if incremental {
			if err := renderIncidentSearchPage(cmd, items); err != nil {
				return process.Incidents{}, false, err
			}
		} else {
			collected.Items = append(collected.Items, items...)
			collected.Total = int32(len(collected.Items))
		}
		processedTotal += len(items)

		continuation := incidentSearchContinuationState(page, processedTotal, autoContinue)
		switch continuation {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return printFoundAndReturn()
		case processInstanceContinuationAutoContinue:
			pageReq = nextIncidentSearchPageRequest(cmd, cfg, pageReq, page)
			continue
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Fetched %d incident(s) on this page (%d total so far). More matching incidents remain. Continue?", len(items), processedTotal)
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					return printFoundAndReturn()
				}
				return process.Incidents{}, false, err
			}
			pageReq = nextIncidentSearchPageRequest(cmd, cfg, pageReq, page)
		}
	}
}
