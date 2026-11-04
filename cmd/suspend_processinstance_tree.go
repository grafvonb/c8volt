// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// runSuspendProcessInstanceTree freezes the service-owned plan before approval.
// Only the existing collection executor submits mutations; rendering never discovers scope.
func runSuspendProcessInstanceTree(cmd *cobra.Command, cli process.API, log *slog.Logger, cfg *config.Config, keys typex.Keys) {
	plan, err := cli.PlanProcessInstanceSuspension(cmd.Context(), keys, collectExplicitPIAdminInputOptions()...)
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err, "plan process instance suspension: "+boundedCommandFailureReason(err))
		return
	}
	frozen := slices.Clone(plan.TargetKeys)
	if flagDryRun || len(frozen) == 0 {
		results := process.SuspensionResults{}
		if flagDryRun {
			results = previewProcessInstanceSuspensionTree(plan)
		}
		if err := renderProcessInstanceSuspensionTree(cmd, plan, results, flagDryRun, nil); err != nil {
			message := "suspend process instances: could not write result"
			if flagDryRun {
				message = "suspend process instances: could not write preview"
			}
			log.Error(message)
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		return
	}
	renderProcessInstanceSuspensionTreeApproval(cmd, plan)
	prompt := fmt.Sprintf("You are about to suspend %d process instance(s). Do you want to proceed?", len(frozen))
	if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err, "suspend process instances: aborted by user")
		return
	}
	results, operationErr := func() (process.SuspensionResults, error) {
		if indicatorEnabled(cmd, cfg) {
			stop := startCommandActivity(cmd, fmt.Sprintf("suspending %d process instance(s)", len(frozen)))
			defer stop()
		}
		return cli.SuspendProcessInstances(cmd.Context(), frozen, flagWorkers, collectExplicitPIAdminInputOptions()...)
	}()
	if err := renderProcessInstanceSuspensionTree(cmd, plan, results, false, operationErr); err != nil {
		log.Error("suspend process instances: could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatProcessInstanceSuspensionFailure(results, operationErr))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}
