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

// incidentSearchContinuationState translates incident overflow metadata into the next CLI paging action.
func incidentSearchContinuationState(page incident.Page, limitReached bool, autoContinue bool) processInstanceContinuationState {
	if limitReached {
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

// describeIncidentOverflowState maps incident overflow metadata to verbose progress wording.
func describeIncidentOverflowState(state incident.OverflowState) string {
	switch state {
	case incident.OverflowStateHasMore:
		return "yes"
	case incident.OverflowStateIndeterminate:
		return "unknown"
	default:
		return "no"
	}
}

func searchIncidentsTotal(cmd *cobra.Command, cli incident.API, cfg *config.Config, filter incident.Filter) (int64, error) {
	pageReq := newIncidentSearchPageRequest(cmd, cfg, 0)
	return cli.SearchIncidentsTotal(cmd.Context(), filter, pageReq, collectOptions()...)
}

// searchIncidentsWithPaging runs the list/search path for `get incident`.
// Human and keys-only output may render page-by-page in interactive mode; JSON
// collects all bounded results so the command emits one valid document.
func searchIncidentsWithPaging(cmd *cobra.Command, cli incident.API, cfg *config.Config, filter incident.Filter) (incident.Incidents, bool, error) {
	pageReq := newIncidentSearchPageRequest(cmd, cfg, 0)
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
		return incident.Incidents{}, false, nil
	}

	result, err := cli.SearchIncidentsPages(cmd.Context(), filter, pageReq, flagGetIncidentLimit, func(step incident.SearchPageStep) (incident.SearchPageAction, error) {
		page := step.Page
		items := page.Items
		if incremental {
			if err := renderIncidentSearchPage(cmd, items); err != nil {
				return incident.SearchPageActionStop, err
			}
		}
		processedTotal = int(step.CumulativeCount)

		continuation := incidentSearchContinuationState(page, step.LimitReached, autoContinue)
		printSearchPageProgress(cmd, searchPageProgressSummary{
			PageSize:          page.Request.Size,
			CurrentPageCount:  len(items),
			CumulativeCount:   processedTotal,
			MoreMatches:       describeIncidentOverflowState(page.OverflowState),
			ContinuationState: continuation,
		})
		switch continuation {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return incident.SearchPageActionStop, nil
		case processInstanceContinuationAutoContinue:
			return incident.SearchPageActionContinue, nil
		case processInstanceContinuationPrompt:
			if len(items) == 0 {
				return incident.SearchPageActionContinue, nil
			}
			prompt := fmt.Sprintf("Fetched %d incident(s) on this page (%s). More matching incidents remain. Continue?", len(items), formatIncidentPagingProgress(page, processedTotal, "loaded"))
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					return incident.SearchPageActionStop, nil
				}
				return incident.SearchPageActionStop, err
			}
			return incident.SearchPageActionContinue, nil
		}
		return incident.SearchPageActionStop, nil
	}, collectOptions()...)
	if err != nil {
		return incident.Incidents{}, false, err
	}
	if incremental {
		return printFoundAndReturn()
	}
	return incident.Incidents{Items: result.Items, Total: int32(len(result.Items))}, false, nil
}
