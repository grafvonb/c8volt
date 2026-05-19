// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// searchProcessInstancesWithPaging runs the list/search path for `get pi`,
// applying local filters and --limit after each backend page. In interactive
// one-line/key modes it can render pages as they arrive, while JSON and other
// aggregate modes collect the full bounded result before rendering.
func searchProcessInstancesWithPaging(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (process.ProcessInstances, bool, error) {
	if canSearchProcessInstancesViaDirectIncidentIndex(cmd, cli, filter) {
		pis, err := searchProcessInstancesViaDirectIncidentIndex(cmd, cli, cfg, filter)
		return pis, false, err
	}

	pageReq := newPISearchPageRequest(cmd, cfg, 0)
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

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return process.ProcessInstances{}, false, err
		}

		filtered, err := applyPISearchResultFilters(cmd, cli, process.ProcessInstances{
			Total: int32(len(page.Items)),
			Items: page.Items,
		})
		if err != nil {
			return process.ProcessInstances{}, false, err
		}
		filtered.Items = limitPIItems(filtered.Items, processedTotal)
		filtered.Total = int32(len(filtered.Items))
		if incremental {
			if flagGetPIWithIncidents && flagGetPIWithVars && pickMode() == RenderModeOneLine {
				incidentEnriched, err := enrichProcessInstancesWithIncidentActivity(cmd, cli, filtered)
				if err != nil {
					return process.ProcessInstances{}, false, fmt.Errorf("get process instance incidents: %w", err)
				}
				variableEnriched, err := enrichProcessInstancesWithVariableActivity(cmd, cli, filtered)
				if err != nil {
					return process.ProcessInstances{}, false, fmt.Errorf("get process instance variables: %w", err)
				}
				pageNeedsIndirectIncidentWarning := renderProcessInstanceActivityRows(cmd, mergeIncidentAndVariableActivity(incidentEnriched, variableEnriched).Items)
				needsIndirectIncidentWarning = needsIndirectIncidentWarning || pageNeedsIndirectIncidentWarning
			} else if flagGetPIWithIncidents && pickMode() == RenderModeOneLine {
				enriched, err := enrichProcessInstancesWithIncidentActivity(cmd, cli, filtered)
				if err != nil {
					return process.ProcessInstances{}, false, fmt.Errorf("get process instance incidents: %w", err)
				}
				pageNeedsIndirectIncidentWarning, err := renderIncidentEnrichedProcessInstanceRows(cmd, enriched)
				if err != nil {
					return process.ProcessInstances{}, false, err
				}
				needsIndirectIncidentWarning = needsIndirectIncidentWarning || pageNeedsIndirectIncidentWarning
			} else if flagGetPIWithVars && pickMode() == RenderModeOneLine {
				enriched, err := enrichProcessInstancesWithVariableActivity(cmd, cli, filtered)
				if err != nil {
					return process.ProcessInstances{}, false, fmt.Errorf("get process instance variables: %w", err)
				}
				renderProcessInstanceActivityRows(cmd, activityFromVariableEnriched(enriched).Items)
			} else if pickMode() == RenderModeOneLine {
				if err := renderProcessInstanceFlatRows(cmd, filtered.Items); err != nil {
					return process.ProcessInstances{}, false, err
				}
			} else {
				for _, item := range filtered.Items {
					if err := processInstanceView(cmd, item); err != nil {
						return process.ProcessInstances{}, false, err
					}
				}
			}
		} else {
			collected.Items = append(collected.Items, filtered.Items...)
			collected.Total = int32(len(collected.Items))
		}
		processedTotal += len(filtered.Items)

		summaryPage := page
		summaryPage.Items = filtered.Items
		summary := newPIProgressSummary(summaryPage, processedTotal, autoContinue)
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return printFoundAndReturn()
		case processInstanceContinuationAutoContinue:
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
			continue
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
					return printFoundAndReturn()
				}
				return process.ProcessInstances{}, false, err
			}
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
		}
	}
}

type processIncidentSearchAPI interface {
	process.API
	SearchIncidentsPage(ctx context.Context, filter incident.Filter, page incident.PageRequest, opts ...options.FacadeOption) (incident.Page, error)
}

func canSearchProcessInstancesViaDirectIncidentIndex(cmd *cobra.Command, cli process.API, filter process.ProcessInstanceFilter) bool {
	if !flagGetPIDirectIncidentsOnly || flagGetPILimit <= 0 || !hasPIIncidentDetailFilters() {
		return false
	}
	if _, ok := cli.(processIncidentSearchAPI); !ok {
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

func searchProcessInstancesViaDirectIncidentIndex(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (process.ProcessInstances, error) {
	searcher, ok := cli.(processIncidentSearchAPI)
	if !ok {
		return process.ProcessInstances{}, fmt.Errorf("direct incident index search requires incident lookup support")
	}
	keys, err := collectDirectIncidentProcessInstanceKeys(cmd, searcher, cfg, filter)
	if err != nil {
		return process.ProcessInstances{}, err
	}
	if len(keys) == 0 {
		return process.ProcessInstances{}, nil
	}
	pis, err := cli.GetProcessInstances(cmd.Context(), keys, len(keys), collectOptions()...)
	if err != nil {
		return process.ProcessInstances{}, fmt.Errorf("get process instances for direct incident matches: %w", err)
	}
	pis = filterDirectIncidentIndexedProcessInstances(pis, filter)
	pis.Items = limitPIItems(pis.Items, 0)
	pis.Total = int32(len(pis.Items))
	return pis, nil
}

func collectDirectIncidentProcessInstanceKeys(cmd *cobra.Command, cli processIncidentSearchAPI, cfg *config.Config, filter process.ProcessInstanceFilter) (typex.Keys, error) {
	pageReq := incident.PageRequest{Size: resolvePISearchSize(cmd, cfg)}
	incidentFilter := directIncidentSearchFilter(filter)
	limit := int(flagGetPILimit)
	keys := make(typex.Keys, 0, limit)
	seen := make(map[string]struct{}, limit)
	for {
		page, err := cli.SearchIncidentsPage(cmd.Context(), incidentFilter, pageReq, collectOptions()...)
		if err != nil {
			return nil, fmt.Errorf("search direct incidents: %w", err)
		}
		for _, item := range page.Items {
			if item.ProcessInstanceKey == "" {
				continue
			}
			if _, ok := seen[item.ProcessInstanceKey]; ok {
				continue
			}
			seen[item.ProcessInstanceKey] = struct{}{}
			keys = append(keys, item.ProcessInstanceKey)
			if len(keys) >= limit {
				return keys, nil
			}
		}
		if page.OverflowState == incident.OverflowStateNoMore {
			return keys, nil
		}
		pageReq = nextDirectIncidentSearchPageRequest(pageReq, page)
	}
}

func directIncidentSearchFilter(filter process.ProcessInstanceFilter) incident.Filter {
	errorType, _ := incidentfilter.NormalizeErrorType(flagGetPIIncidentErrorType)
	return incident.Filter{
		State:                flagGetPIIncidentState,
		ErrorType:            errorType,
		ErrorMessage:         flagGetPIIncidentErrorMessage,
		ProcessDefinitionId:  filter.BpmnProcessId,
		ProcessDefinitionKey: filter.ProcessDefinitionKey,
	}
}

func nextDirectIncidentSearchPageRequest(current incident.PageRequest, page incident.Page) incident.PageRequest {
	if page.EndCursor != "" {
		return incident.PageRequest{Size: current.Size, After: page.EndCursor}
	}
	return incident.PageRequest{From: current.From + current.Size, Size: current.Size}
}

func filterDirectIncidentIndexedProcessInstances(pis process.ProcessInstances, filter process.ProcessInstanceFilter) process.ProcessInstances {
	out := make([]process.ProcessInstance, 0, len(pis.Items))
	for _, item := range pis.Items {
		if filter.State != "" && item.State != filter.State {
			continue
		}
		if filter.BpmnProcessId != "" && item.BpmnProcessId != filter.BpmnProcessId {
			continue
		}
		if filter.ProcessDefinitionKey != "" && item.ProcessDefinitionKey != filter.ProcessDefinitionKey {
			continue
		}
		out = append(out, item)
	}
	return process.ProcessInstances{Total: int32(len(out)), Items: out}
}
