// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runOpsRepairProcessInstanceBusinessID confirms a complete plan and retains the original search provenance.
func runOpsRepairProcessInstanceBusinessID(cmd *cobra.Command, cli ops.API, log *slog.Logger, cfg *config.Config, request ops.RepairRequest) {
	opts := append(collectOptions(), foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())
	initializeOpsTenantContextHumanReporting(cmd, cfg, false)
	planRequest := request
	planRequest.DryRun = true
	planned, err := repairProcessInstanceWithCommandActivity(cmd, planRequest, func() (ops.RepairResult, error) {
		return cli.RepairProcessInstances(cmd.Context(), planRequest, opts...)
	})
	result := planned
	if err == nil && !request.DryRun {
		if opsRepairPlanHasRepairTargets(planned) {
			attachOpsRepairResultTenantContext(cmd, cfg, planned)
			if !shouldImplicitlyConfirm(cmd) {
				renderAttachedTenantContext(cmd)
			}
			err = confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), opsRepairConfirmationPrompt(planned))
			if err == nil {
				request = opsRepairConfirmedRequestFromPlan(request, planned)
				result, err = repairProcessInstanceWithCommandActivity(cmd, request, func() (ops.RepairResult, error) { return cli.RepairProcessInstances(cmd.Context(), request, opts...) })
			}
		} else {
			result = opsRepairResultWithoutMutation(request, planned)
		}
	}
	result = attachOpsRepairResultTenantContext(cmd, cfg, result)
	if reportErr := writeOpsRepairReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
		err = errors.Join(err, fmt.Errorf("write repair audit report: %w", reportErr))
	}
	finishBusinessIDRepair(cmd, log, cfg, result, err)
}

// finishBusinessIDRepair keeps partial evidence in one envelope and preserves the original failure exit.
func finishBusinessIDRepair(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, result ops.RepairResult, operationErr error) {
	if err := renderBusinessIDRepair(cmd, result, operationErr); err != nil {
		log.Error("repair process instances: could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			submitted, confirmed, failed := businessIDRepairOutcomeCounts(result)
			log.Error(fmt.Sprintf("repair process instances: selected %d, incidents %d, submitted %d, confirmed %d, failed %d; %s", len(result.FrozenSet.ProcessInstanceKeys)+len(result.FrozenSet.SkippedProcessInstanceKeys), len(result.FrozenSet.IncidentKeys), submitted, confirmed, failed, boundedCommandFailureReason(operationErr)))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}
