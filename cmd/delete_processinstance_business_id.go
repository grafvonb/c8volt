// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runDeleteProcessInstanceBusinessID freezes a complete cross-family plan before the one destructive prompt.
// Services own selection, expansion and execution; the command owns only confirmation and presentation.
func runDeleteProcessInstanceBusinessID(cmd *cobra.Command, cli process.API, cfg *config.Config, log *slog.Logger) {
	filter := populatePISearchFilterOpts(cmd)
	if err := cli.ValidateProcessInstanceSearchFilter(filter); err != nil {
		handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
		return
	}
	if flagGetPIBpmnProcessID != "" {
		result, err := validateProcessDefinitionSelectorsForCommand(cmd.Context(), cmd, cli, newPIProcessDefinitionSelectorValidationRequest(), collectOptions()...)
		if err != nil {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
		if !result.Valid() {
			handleProcessDefinitionSelectorValidationError(cmd, log, cfg.App.NoErrCodes, cli, result)
			return
		}
	}
	opts := append(collectOptions(), foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())
	tenantCtx := attachDiscoveryTenantContext(cmd, cfg)
	plan, err := cli.PlanProcessInstanceSelectedMutation(cmd.Context(), process.ProcessInstanceSelectedMutationRequest{
		Operation: "delete", Workers: flagWorkers, Selection: process.ProcessInstanceSelectionRequest{Cardinality: process.ProcessInstanceSelectionCollection, SearchRequest: newProcessInstanceSearchRequest(cmd, cfg, filter)},
	}, opts...)
	if err != nil {
		finishBusinessIDDeletion(cmd, log, cfg, plan, process.ProcessInstanceDeletionResult{}, true, err)
		return
	}
	attachProcessInstanceDiscoveryTenantContext(cmd, tenantCtx, plan.Expansion.TenantEvidence)
	if len(plan.Selection.Keys) == 0 {
		if err := renderEmptyProcessInstanceSelectorResult(cmd, "delete", flagDryRun); err != nil {
			log.Error("delete process instances: could not write empty result")
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		return
	}
	if flagDryRun {
		finishBusinessIDDeletion(cmd, log, cfg, plan, process.ProcessInstanceDeletionResult{}, true, nil)
		return
	}
	if err := rejectDeletePlanRequiringForce(plan.Expansion); err != nil {
		finishBusinessIDDeletion(cmd, log, cfg, plan, process.ProcessInstanceDeletionResult{}, true, err)
		return
	}
	affected, roots, selected := len(plan.Expansion.Collected), len(plan.Expansion.Roots), len(plan.Selection.Keys)
	renderAttachedTenantContext(cmd)
	prompt := fmt.Sprintf("You are about to delete %d process instance(s). Do you want to proceed?", affected)
	if affected > selected {
		prompt = fmt.Sprintf("You have requested to delete %d process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be deleted. Do you want to proceed?", selected, affected, roots)
	}
	if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
		finishBusinessIDDeletion(cmd, log, cfg, plan, process.ProcessInstanceDeletionResult{}, true, err)
		return
	}
	reports, operationErr := cli.ExecuteProcessInstanceDeletionPlan(cmd.Context(), plan, opts...)
	finishBusinessIDDeletion(cmd, log, cfg, plan, reports, false, operationErr)
}

// finishBusinessIDDeletion preserves the original failure class after the sole result write attempt.
func finishBusinessIDDeletion(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, plan process.ProcessInstanceSelectedMutationPlan, reports process.ProcessInstanceDeletionResult, preview bool, operationErr error) {
	if err := renderBusinessIDDeletion(cmd, plan, reports, preview, operationErr); err != nil {
		log.Error("delete process instances: could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatBusinessIDDeletionFailure(plan, reports, operationErr))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}
