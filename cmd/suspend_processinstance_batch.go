// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// runSuspendProcessInstances freezes locally validated keys before approval. Services own
// scheduling, submission and observation; rendering cannot trigger backend work.
func runSuspendProcessInstances(cmd *cobra.Command, cli process.API, log *slog.Logger, cfg *config.Config, keys typex.Keys) {
	preview, err := cli.SuspendProcessInstances(cmd.Context(), keys, flagWorkers, append(collectExplicitPIAdminInputOptions(), foptions.WithDryRun())...)
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err, "suspend process instances: "+boundedCommandFailureReason(err))
		return
	}
	if flagDryRun || len(preview.Items) == 0 {
		if err := renderProcessInstanceSuspension(cmd, preview, true, nil); err != nil {
			log.Error("suspend process instances: could not write preview")
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		return
	}
	frozen := make(typex.Keys, len(preview.Items))
	for i, item := range preview.Items {
		frozen[i] = item.Key
	}
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
	if err := renderProcessInstanceSuspension(cmd, results, false, operationErr); err != nil {
		// A partially written result must not cause a second envelope or hide an operation failure.
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
