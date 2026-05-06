// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// searchProcessInstancesWithPaging runs the list/search path for `get pi`,
// applying local filters and --limit after each backend page. In interactive
// one-line/key modes it can render pages as they arrive, while JSON and other
// aggregate modes collect the full bounded result before rendering.
func searchProcessInstancesWithPaging(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (process.ProcessInstances, bool, error) {
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
			prompt := fmt.Sprintf("Fetched %d process instance(s) on this page (%d total so far). More matching process instances remain. Continue?", summary.CurrentPageCount, summary.CumulativeCount)
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
