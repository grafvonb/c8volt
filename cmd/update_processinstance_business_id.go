// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runUpdateProcessInstanceBusinessID reviews the same root payloads consumed after approval.
func runUpdateProcessInstanceBusinessID(cmd *cobra.Command, cli process.API, log *slog.Logger, cfg *config.Config, args []string, variables map[string]any) {
	if err := validateBusinessIDActionInput(cmd, args); err != nil {
		handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
		return
	}
	if err := validateUpdateProcessInstanceJSONConfirmation(cmd); err != nil {
		handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
		return
	}
	opts := collectOptions()
	tenant := attachDiscoveryTenantContext(cmd, cfg)
	plan, err := cli.PlanSelectedProcessInstanceVariableUpdates(cmd.Context(), businessIDActionSelection(cmd), variables, opts...)
	if plan.Selection != nil {
		attachProcessInstanceDiscoveryTenantContext(cmd, tenant, plan.Selection.TenantEvidence)
	}
	preview := processInstanceVariableUpdatePreviewFromFacadePlan(plan)
	if len(plan.RequestedKeys) > 0 || err != nil {
		preview.Selection = plan.Selection
	}
	results := process.ProcessInstanceVariableUpdateResults{}
	planned := true
	if err == nil && !flagDryRun && preview.HasPlannedChanges() {
		if !shouldImplicitlyConfirm(cmd) {
			if renderErr := renderUpdateProcessInstanceVariablePlan(cmd, preview); renderErr != nil {
				log.Error("update process-instance variables: could not write plan")
				exitAfterRenderedResult(cfg.App.NoErrCodes, renderErr)
				return
			}
		}
		count := preview.VariableAddCount + preview.VariableChangeCount
		prompt := fmt.Sprintf("You are about to update %d requested variable value(s) on %d process instance(s). Do you want to proceed?", count, preview.UpdateCount)
		err = confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt)
		if err == nil {
			planned = false
			results, err = cli.ExecuteProcessInstanceVariableUpdates(cmd.Context(), plan, flagWorkers, opts...)
			results.Selection = plan.Selection
		}
	}
	if renderErr := renderBusinessIDVariableUpdate(cmd, preview, results, planned, err); renderErr != nil {
		log.Error("update process-instance variables: could not write result")
		if err != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, renderErr)
	}
	if err != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatBusinessIDVariableUpdateFailure(plan, results, err))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
}
