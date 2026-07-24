// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/spf13/cobra"
)

// searchProcessInstancesWithPaging runs the list/search path for `get pi`,
// applying local filters and --limit after each backend page. In interactive
// one-line/key modes it can render pages as they arrive, while JSON and other
// aggregate modes collect the full bounded result before rendering.
func searchProcessInstancesWithPaging(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (process.ProcessInstances, bool, error) {
	request := newProcessInstanceSearchRequest(cmd, cfg, filter)
	var collected process.ProcessInstances
	incremental := shouldRenderPISearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinuePISearchPages(cmd)
	processedTotal := 0
	needsIndirectIncidentWarning := false
	printFoundAndReturn := func() (process.ProcessInstances, bool, error) {
		if incremental {
			if pickMode() == RenderModeOneLine {
				if needsIndirectIncidentWarning {
					renderHumanWarningLine(cmd, indirectProcessTreeIncidentWarning)
				}
				renderOutputLine(cmd, "found: %d", processedTotal)
			}
			return process.ProcessInstances{}, true, nil
		}
		return collected, false, nil
	}

	result, err := cli.SearchProcessInstancesPages(cmd.Context(), request, func(step process.ProcessInstanceSearchPageStep) (process.ProcessInstanceSearchPageAction, error) {
		page := step.Page
		filtered := process.ProcessInstances{
			Total: int32(len(page.Items)),
			Items: page.Items,
		}
		if incremental {
			if (flagGetPIWithIncidents || flagGetPIWithVars || flagGetPIWithElements) && pickMode() == RenderModeOneLine {
				activity, err := collectRequestedProcessInstanceActivity(cmd, cli, filtered)
				if err != nil {
					return process.ProcessInstanceSearchPageActionStop, err
				}
				pageNeedsIndirectIncidentWarning := renderProcessInstanceActivityRows(cmd, activity.Items)
				needsIndirectIncidentWarning = needsIndirectIncidentWarning || pageNeedsIndirectIncidentWarning
			} else if pickMode() == RenderModeOneLine {
				if err := renderProcessInstanceFlatRows(cmd, filtered.Items); err != nil {
					return process.ProcessInstanceSearchPageActionStop, err
				}
			} else {
				for _, item := range filtered.Items {
					if err := processInstanceView(cmd, item); err != nil {
						return process.ProcessInstanceSearchPageActionStop, err
					}
				}
			}
		} else {
			collected.Items = append(collected.Items, filtered.Items...)
			collected.Total = int32(len(collected.Items))
		}
		processedTotal = int(step.CumulativeCount)

		summaryPage := page
		summary := newPIProgressSummary(summaryPage, processedTotal, autoContinue)
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return process.ProcessInstanceSearchPageActionStop, nil
		case processInstanceContinuationAutoContinue:
			return process.ProcessInstanceSearchPageActionContinue, nil
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Fetched %d process instance(s) on this page (%s). More matching process instances remain. Continue?", summary.CurrentPageCount, formatProcessInstancePagingProgress(summaryPage, summary.CumulativeCount, "loaded"))
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					return process.ProcessInstanceSearchPageActionStop, nil
				}
				return process.ProcessInstanceSearchPageActionStop, err
			}
			return process.ProcessInstanceSearchPageActionContinue, nil
		}
		return process.ProcessInstanceSearchPageActionStop, nil
	}, collectOptions()...)
	if err != nil {
		return process.ProcessInstances{}, false, err
	}
	if incremental {
		return printFoundAndReturn()
	}
	return process.ProcessInstances{Total: int32(len(result.Items)), Items: result.Items}, false, nil
}

func canSearchProcessInstancesViaDirectIncidentIndex(cmd *cobra.Command, filter process.ProcessInstanceFilter) bool {
	if !flagGetPIDirectIncidentsOnly || flagGetPILimit <= 0 || !hasPIIncidentDetailFilters() {
		return false
	}
	if flagGetPIWithIncidents || flagGetPIWithVars || flagGetPIIncidentsOnly || flagGetPINoIncidentsOnly {
		return false
	}
	if flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly {
		return false
	}
	return filter.ProcessVersion == 0 &&
		filter.ProcessVersionTag == "" &&
		filter.ParentKey == "" &&
		filter.StartDateAfter == "" &&
		filter.StartDateBefore == "" &&
		filter.EndDateAfter == "" &&
		filter.EndDateBefore == ""
}

func directIncidentSearchFilter(filter process.ProcessInstanceFilter) process.ProcessInstanceIncidentSearchFilter {
	errorType, _ := incidentfilter.NormalizeErrorType(flagGetPIIncidentErrorType)
	state, _ := incidentfilter.NormalizeState(flagGetPIIncidentState)
	return process.ProcessInstanceIncidentSearchFilter{
		State:                state,
		ErrorType:            errorType,
		ErrorMessage:         flagGetPIIncidentErrorMessage,
		ProcessDefinitionId:  filter.BpmnProcessId,
		ProcessDefinitionKey: filter.ProcessDefinitionKey,
	}
}

// newProcessInstanceSearchRequest translates command flags into a service-owned
// process-instance search traversal request without taking over rendering policy.
func newProcessInstanceSearchRequest(cmd *cobra.Command, cfg *config.Config, filter process.ProcessInstanceFilter) process.ProcessInstanceSearchRequest {
	state, _ := incidentfilter.NormalizeState(flagGetPIIncidentState)
	errorType, _ := incidentfilter.NormalizeErrorType(flagGetPIIncidentErrorType)
	return process.ProcessInstanceSearchRequest{
		Filter: filter,
		Page:   newPISearchPageRequest(cmd, cfg, 0),
		Limit:  flagGetPILimit,
		LocalFilters: process.ProcessInstanceSearchLocalFilters{
			ChildrenOnly:         flagGetPIChildrenOnly,
			RootsOnly:            flagGetPIRootsOnly,
			OrphanChildrenOnly:   flagGetPIOrphanChildrenOnly,
			IncidentsOnly:        flagGetPIIncidentsOnly,
			DirectIncidentsOnly:  flagGetPIDirectIncidentsOnly,
			NoIncidentsOnly:      flagGetPINoIncidentsOnly,
			IncidentState:        state,
			IncidentErrorType:    errorType,
			IncidentErrorMessage: flagGetPIIncidentErrorMessage,
		},
		DirectIncidentIndex:  canSearchProcessInstancesViaDirectIncidentIndex(cmd, filter),
		DirectIncidentFilter: directIncidentSearchFilter(filter),
		ReportedTotalAllowed: canUsePIReportedTotal(),
	}
}
