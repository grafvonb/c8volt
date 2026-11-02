// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runResolveProcessInstanceBusinessID asks once, after all family and incident work has been frozen.
func runResolveProcessInstanceBusinessID(cmd *cobra.Command, cli process.API, cfg *config.Config, log *slog.Logger) {
	opts := append(collectOptions(), foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())
	tenant := attachDiscoveryTenantContext(cmd, cfg)
	plan, err := cli.PlanSelectedProcessInstanceResolution(cmd.Context(), businessIDActionSelection(cmd), flagWorkers, opts...)
	if err != nil {
		finishBusinessIDResolution(cmd, log, cfg, plan, incident.ProcessInstanceResolutionResults{}, err)
		return
	}
	attachProcessInstanceDiscoveryTenantContext(cmd, tenant, plan.Scope.Expansion.TenantEvidence)
	if !flagDryRun && businessIDResolutionIncidentCount(plan) > 0 {
		renderAttachedTenantContext(cmd)
		affected, selected, roots := len(plan.Scope.Expansion.Collected), len(plan.Scope.Selection.Keys), len(plan.Scope.Expansion.Roots)
		prompt := fmt.Sprintf("You are about to inspect %d process instance(s) and resolve active incidents found in that family. Do you want to proceed?", affected)
		if affected > selected {
			prompt = fmt.Sprintf("You have requested to resolve incidents for %d process instance(s), but due to the process-instance family scope, %d instance(s) with %d root instance(s) will be inspected and active incidents found in that family will be resolved. Do you want to proceed?", selected, affected, roots)
		}
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
			finishBusinessIDResolution(cmd, log, cfg, plan, incident.ProcessInstanceResolutionResults{}, err)
			return
		}
	}
	result, err := cli.ExecuteSelectedProcessInstanceResolution(cmd.Context(), plan, opts...)
	finishBusinessIDResolution(cmd, log, cfg, plan, result, err)
}

// finishBusinessIDResolution writes exactly one result and exits using the original operation failure.
func finishBusinessIDResolution(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, plan process.SelectedProcessInstanceResolutionPlan, result incident.ProcessInstanceResolutionResults, operationErr error) {
	if err := renderBusinessIDResolution(cmd, plan, result, operationErr); err != nil {
		log.Error("resolve process instances: could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			message := fmt.Sprintf("resolve process instances: selected %d, affected %d, confirmed %d, submitted %d, failed %d; %s", len(plan.Scope.Selection.Keys), len(plan.Scope.Expansion.Collected), result.Confirmed, result.Submitted, result.Failed, boundedCommandFailureReason(operationErr))
			if result.MutationSubmitted {
				message += "; mutation submitted; outcome incomplete"
			}
			log.Error(message)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}
