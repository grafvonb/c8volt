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

// runSuspendProcessInstanceBusinessID freezes tenant-scoped selection before approval and exact submission.
func runSuspendProcessInstanceBusinessID(cmd *cobra.Command, cli process.API, log *slog.Logger, cfg *config.Config) {
	opts := collectOptions()
	tenant := attachDiscoveryTenantContext(cmd, cfg)
	plan, err := cli.PlanSelectedProcessInstanceSuspension(cmd.Context(), businessIDActionSelection(cmd), flagSuspendPITree, opts...)
	if plan.Selection != nil {
		attachProcessInstanceDiscoveryTenantContext(cmd, tenant, plan.Selection.TenantEvidence)
	}
	results := process.SuspensionResults{}
	preview := flagDryRun
	if err == nil && flagDryRun {
		results = previewProcessInstanceSuspensionTree(plan)
	}
	if err == nil && !flagDryRun && len(plan.TargetKeys) > 0 {
		if flagSuspendPITree {
			renderProcessInstanceSuspensionTreeApproval(cmd, plan)
		}
		prompt := fmt.Sprintf("You are about to suspend %d process instance(s). Do you want to proceed?", len(plan.TargetKeys))
		err = confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt)
		if err == nil {
			results, err = func() (process.SuspensionResults, error) {
				if indicatorEnabled(cmd, cfg) {
					stop := startCommandActivity(cmd, fmt.Sprintf("suspending %d process instance(s)", len(plan.TargetKeys)))
					defer stop()
				}
				return cli.ExecuteProcessInstanceSuspensionPlan(cmd.Context(), plan, flagWorkers, opts...)
			}()
		}
	}
	// Empty exact results retain the established payload; nonempty results retain provenance.
	if len(plan.RequestedKeys) > 0 || err != nil {
		results.Selection = plan.Selection
	}
	var renderErr error
	if flagSuspendPITree {
		renderErr = renderProcessInstanceSuspensionTree(cmd, plan, results, preview, err)
	} else {
		renderErr = renderProcessInstanceSuspension(cmd, results, preview, err)
	}
	if renderErr != nil {
		log.Error("suspend process instances: could not write result")
		if err != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, renderErr)
	}
	if err != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatProcessInstanceSuspensionFailure(results, err))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
}
