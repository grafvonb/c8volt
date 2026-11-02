// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"fmt"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// assignmentInput consumes all explicit input before the service validates and
// freezes it. Deliberate empty stdin is a valid scope, unlike missing input.
func assignmentInput(cmd *cobra.Command, args []string) (typex.Keys, error) {
	for _, name := range []string{"bid", "vars", "vars-file", "limit", "batch-size"} {
		if cmd.Flags().Changed(name) {
			return nil, mutuallyExclusiveFlagsf("--set-bid cannot be combined with --%s", name)
		}
	}
	keys := append(typex.Keys(nil), flagUpdatePIKeys...)
	stdin := len(args) == 1 && args[0] == "-"
	if stdin {
		scanner := bufio.NewScanner(cmd.InOrStdin())
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			if key := strings.TrimSpace(scanner.Text()); key != "" {
				keys = append(keys, key)
			}
		}
		if scanner.Err() != nil {
			return nil, invalidFlagValuef("could not read process instance keys from stdin")
		}
	}
	if len(keys) == 0 && !stdin {
		return nil, missingDependentFlagsf("--set-bid requires --key or stdin '-'")
	}
	return keys, nil
}

// runUpdateProcessInstanceAssignment approves one copied plan and renders once.
// Backend execution, deduplication and observation remain service-owned.
func runUpdateProcessInstanceAssignment(cmd *cobra.Command, cli process.API, log *slog.Logger, cfg *config.Config, args []string) {
	keys, err := assignmentInput(cmd, args)
	if err == nil {
		err = validateUpdateProcessInstanceJSONConfirmation(cmd)
	}
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		return
	}
	opts := collectExplicitPIAdminInputOptions()
	plan, err := cli.PlanProcessInstanceBusinessIDAssignments(cmd.Context(), keys, &flagUpdatePISetBID, opts...)
	results := process.BusinessIDAssignmentResults{}
	planned := true
	if err == nil && !flagDryRun && len(plan.Items) > 0 {
		if !shouldImplicitlyConfirm(cmd) && pickMode() == RenderModeOneLine {
			if renderErr := renderBusinessIDAssignment(cmd, plan, results, true, nil); renderErr != nil {
				log.Error("business-ID assignment: could not write plan")
				exitAfterRenderedResult(cfg.App.NoErrCodes, renderErr)
				return
			}
		}
		prompt := fmt.Sprintf("You are about to assign a business ID to %d process instance(s). Do you want to proceed?", len(plan.Items))
		err = confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt)
		if err == nil {
			planned = false
			results, err = cli.ExecuteProcessInstanceBusinessIDAssignments(cmd.Context(), plan, flagWorkers, opts...)
		}
	}
	if renderErr := renderBusinessIDAssignment(cmd, plan, results, planned, err); renderErr != nil {
		log.Error("business-ID assignment: could not write result")
		if err != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, renderErr)
	}
	if err != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatBusinessIDAssignmentFailure(plan, results, planned, err))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
}
