// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/incident"
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

func newIncidentSearchPageRequest(cmd *cobra.Command, cfg *config.Config, from int32) incident.PageRequest {
	return incident.PageRequest{
		From: from,
		Size: resolveIncidentSearchSize(cmd, cfg),
	}
}

func shouldRenderIncidentSearchPageIncrementally(cmd *cobra.Command) bool {
	if flagCmdAutoConfirm {
		return false
	}
	if flagGetIncidentPIKeysOnly {
		return true
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

func limitIncidentItems(items []incident.ProcessInstanceIncidentDetail, cumulative int) []incident.ProcessInstanceIncidentDetail {
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

func nextIncidentSearchPageRequest(cmd *cobra.Command, cfg *config.Config, current incident.PageRequest, page incident.Page) incident.PageRequest {
	if page.EndCursor != "" {
		req := newIncidentSearchPageRequest(cmd, cfg, 0)
		req.After = page.EndCursor
		return req
	}
	return newIncidentSearchPageRequest(cmd, cfg, current.From+current.Size)
}

func incidentSearchContinuationState(page incident.Page, cumulative int, autoContinue bool) processInstanceContinuationState {
	if isIncidentLimitReached(cumulative) {
		return processInstanceContinuationLimitReached
	}
	switch page.OverflowState {
	case incident.OverflowStateHasMore:
		if autoContinue {
			return processInstanceContinuationAutoContinue
		}
		return processInstanceContinuationPrompt
	case incident.OverflowStateIndeterminate:
		return processInstanceContinuationWarningStop
	default:
		return processInstanceContinuationCompleted
	}
}

func renderIncidentSearchPage(cmd *cobra.Command, items []incident.ProcessInstanceIncidentDetail) error {
	if flagGetIncidentPIKeysOnly {
		return renderIncidentProcessInstanceKeys(cmd, items)
	}
	switch pickMode() {
	case RenderModeKeysOnly:
		for _, item := range items {
			renderOutputLine(cmd, "%s", item.IncidentKey)
		}
	default:
		for _, line := range formatIncidentListRowsWithTimezone(items, flagGetIncidentMessageLimit, flagGetIncidentNoErrorMessage, commandShowTimezoneOffset(cmd)) {
			renderOutputLine(cmd, "%s", line)
		}
	}
	return nil
}

func canUseIncidentExactReportedTotal(page incident.Page) bool {
	return page.ReportedTotal != nil && page.ReportedTotal.Kind == incident.ReportedTotalKindExact
}

func searchIncidentsTotal(cmd *cobra.Command, cli incident.API, cfg *config.Config, filter incident.Filter) (int64, error) {
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
		if page.OverflowState == incident.OverflowStateNoMore {
			return total, nil
		}
		pageReq = nextIncidentSearchPageRequest(cmd, cfg, pageReq, page)
	}
}

// searchIncidentsWithPaging runs the list/search path for `get incident`.
// Human and keys-only output may render page-by-page in interactive mode; JSON
// collects all bounded results so the command emits one valid document.
func searchIncidentsWithPaging(cmd *cobra.Command, cli incident.API, cfg *config.Config, filter incident.Filter) (incident.Incidents, bool, error) {
	pageReq := newIncidentSearchPageRequest(cmd, cfg, 0)
	var collected incident.Incidents
	incremental := shouldRenderIncidentSearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinueIncidentSearchPages(cmd)
	processedTotal := 0
	printFoundAndReturn := func() (incident.Incidents, bool, error) {
		if incremental {
			if pickMode() == RenderModeOneLine && !flagGetIncidentPIKeysOnly {
				renderOutputLine(cmd, "found: %d", processedTotal)
			}
			return incident.Incidents{}, true, nil
		}
		return collected, false, nil
	}

	for {
		page, err := cli.SearchIncidentsPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return incident.Incidents{}, false, err
		}
		items := limitIncidentItems(page.Items, processedTotal)
		if incremental {
			if err := renderIncidentSearchPage(cmd, items); err != nil {
				return incident.Incidents{}, false, err
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
			prompt := fmt.Sprintf("Fetched %d incident(s) on this page (%s). More matching incidents remain. Continue?", len(items), formatIncidentPagingProgress(page, processedTotal, "loaded"))
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					return printFoundAndReturn()
				}
				return incident.Incidents{}, false, err
			}
			pageReq = nextIncidentSearchPageRequest(cmd, cfg, pageReq, page)
		}
	}
}
