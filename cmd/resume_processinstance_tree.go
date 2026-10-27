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

// runResumeProcessInstanceTree freezes the service-owned plan before approval.
// Only the existing collection executor submits mutations; rendering never discovers scope.
func runResumeProcessInstanceTree(cmd *cobra.Command, cli process.API, log *slog.Logger, cfg *config.Config, keys typex.Keys) {
	plan, err := cli.PlanProcessInstanceResumption(cmd.Context(), keys, collectExplicitPIAdminInputOptions()...)
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err, "plan process instance resumption: "+boundedCommandFailureReason(err))
		return
	}
	if len(plan.RequestedKeys) == 0 || len(plan.Instances) == 0 {
		err := localPreconditionError(fmt.Errorf("resumption planning returned no scope"))
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err, "plan process instance resumption: no resolved scope")
		return
	}
	// Own the approval and reporting snapshot even if a custom facade reuses storage.
	plan.RequestedKeys = slices.Clone(plan.RequestedKeys)
	plan.Instances = slices.Clone(plan.Instances)
	plan.TargetKeys = slices.Clone(plan.TargetKeys)
	frozen := slices.Clone(plan.TargetKeys)
	if flagDryRun || len(frozen) == 0 {
		results := process.ResumptionResults{}
		if flagDryRun {
			results = previewProcessInstanceResumptionTree(plan)
		}
		if err := renderProcessInstanceResumptionTree(cmd, plan, results, flagDryRun, nil); err != nil {
			message := "resume process instances: could not write result"
			if flagDryRun {
				message = "resume process instances: could not write preview"
			}
			log.Error(message)
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		return
	}
	renderProcessInstanceResumptionTreeApproval(cmd, plan)
	prompt := fmt.Sprintf("You are about to resume %d process instance(s). Do you want to proceed?", len(frozen))
	if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err, "resume process instances: aborted by user")
		return
	}
	results, operationErr := func() (process.ResumptionResults, error) {
		if indicatorEnabled(cmd, cfg) {
			stop := startCommandActivity(cmd, fmt.Sprintf("resuming %d process instance(s)", len(frozen)))
			defer stop()
		}
		return cli.ResumeProcessInstances(cmd.Context(), frozen, flagWorkers, collectExplicitPIAdminInputOptions()...)
	}()
	if err := renderProcessInstanceResumptionTree(cmd, plan, results, false, operationErr); err != nil {
		log.Error("resume process instances: could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatProcessInstanceResumptionFailure(results, operationErr))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}
