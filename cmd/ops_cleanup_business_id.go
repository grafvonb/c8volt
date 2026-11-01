// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runBusinessIDRetention confirms one complete reviewed scope and consumes its fixed boundary and membership.
func runBusinessIDRetention(cmd *cobra.Command, cli ops.API, log *slog.Logger, cfg *config.Config, request ops.RetentionPolicyRequest) {
	opts := append(collectOptions(), foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())
	initializeOpsTenantContextHumanReporting(cmd, cfg, false)
	plannedRequest := request
	plannedRequest.DryRun = true
	result, err := executeRetentionPolicyWithCommandActivity(cmd, plannedRequest, func() (ops.RetentionPolicyResult, error) {
		return cli.ExecuteRetentionPolicy(cmd.Context(), plannedRequest, opts...)
	})
	if err == nil && !request.DryRun {
		err = rejectOpsExecuteRetentionPolicyPlanRequiringForce(result.DeletePlan)
		failureStatus := ops.WorkflowStepStatusConfirmationFailed
		if err != nil {
			failureStatus = ops.WorkflowStepStatusBlocked
		}
		if err == nil && len(result.DeletePlan.ResolvedRootKeys) > 0 {
			result = attachOpsExecuteRetentionPolicyResultTenantContext(cmd, cfg, result)
			if !shouldImplicitlyConfirm(cmd) {
				renderAttachedTenantContext(cmd)
			}
			err = confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), opsExecuteRetentionPolicyConfirmationPrompt(result))
		}
		if err == nil {
			request.FrozenPlan = result.FrozenPlan
			result, err = executeRetentionPolicyWithCommandActivity(cmd, request, func() (ops.RetentionPolicyResult, error) {
				return cli.ExecuteRetentionPolicy(cmd.Context(), request, opts...)
			})
		} else {
			result = markOpsExecuteRetentionPolicyLocalFailure(result, failureStatus, err)
		}
	}
	result = attachOpsExecuteRetentionPolicyResultTenantContext(cmd, cfg, result)
	reportErr := writeOpsExecuteRetentionPolicyReport(result, cfg, opsExecuteRetentionPolicyReportWriteMode(result))
	if err != nil || reportErr != nil {
		if err == nil {
			err = reportErr
			reportErr = nil
		}
		if renderErr := renderOpsProcessInstancePurgeFailure(cmd, log, result, "retention policy", result.Discovery.Count, result.Deletion.Items, err, reportErr); renderErr != nil {
			log.Error("retention policy: could not write result")
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		return
	}
	if renderErr := renderOpsExecuteRetentionPolicyResult(cmd, result); renderErr != nil {
		log.Error("retention policy: could not write result")
		exitAfterRenderedResult(cfg.App.NoErrCodes, renderErr)
	}
}

// runBusinessIDOrphanPurge keeps candidate classification and reviewed deletion scope through approval.
func runBusinessIDOrphanPurge(cmd *cobra.Command, cli ops.API, log *slog.Logger, cfg *config.Config, request ops.OrphanPurgeRequest) {
	opts := append(collectOptions(), foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())
	initializeOpsTenantContextHumanReporting(cmd, cfg, false)
	plannedRequest := request
	plannedRequest.DryRun = true
	result, err := purgeOrphanProcessInstancesWithCommandActivity(cmd, plannedRequest, func() (ops.OrphanPurgeResult, error) {
		return cli.PurgeOrphanProcessInstances(cmd.Context(), plannedRequest, opts...)
	})
	if err == nil && !request.DryRun {
		err = rejectDeletePlanRequiringForce(result.DeletionPlan.DryRunPreview)
		failureStatus := ops.WorkflowStepStatusConfirmationFailed
		if err != nil {
			failureStatus = ops.WorkflowStepStatusBlocked
		}
		if err == nil && result.Discovery.Count > 0 {
			result = attachOpsPurgeOrphanProcessInstancesResultTenantContext(cmd, cfg, result)
			if !shouldImplicitlyConfirm(cmd) {
				renderAttachedTenantContext(cmd)
			}
			err = confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), opsPurgeOrphanProcessInstancesConfirmationPrompt(result))
		}
		if err == nil {
			request.FrozenPlan = result.FrozenPlan
			result, err = purgeOrphanProcessInstancesWithCommandActivity(cmd, request, func() (ops.OrphanPurgeResult, error) {
				return cli.PurgeOrphanProcessInstances(cmd.Context(), request, opts...)
			})
		} else {
			result = markOpsPurgeOrphanProcessInstancesLocalFailure(result, failureStatus, err)
		}
	}
	result = attachOpsPurgeOrphanProcessInstancesResultTenantContext(cmd, cfg, result)
	reportErr := writeOpsPurgeOrphanProcessInstancesReport(result, cfg, opsPurgeOrphanProcessInstancesReportWriteMode(result))
	if err != nil || reportErr != nil {
		if err == nil {
			err = reportErr
			reportErr = nil
		}
		if renderErr := renderOpsProcessInstancePurgeFailure(cmd, log, result, "orphan purge", result.Discovery.Count, result.Deletion.Items, err, reportErr); renderErr != nil {
			log.Error("orphan purge: could not write result")
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		return
	}
	if renderErr := renderOpsPurgeOrphanProcessInstancesResult(cmd, result); renderErr != nil {
		log.Error("orphan purge: could not write result")
		exitAfterRenderedResult(cfg.App.NoErrCodes, renderErr)
	}
}
