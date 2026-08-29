// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// configureOpsSlowProcessAnalysisPreflight wires command-owned rendering and prompting into service-owned discovery.
func configureOpsSlowProcessAnalysisPreflight(cmd *cobra.Command, request *ops.SlowProcessAnalysisRequest) {
	configureOpsSlowProcessAnalysisPreflightWithPacer(cmd, request, newOpsProgressMilestonePacer(nil))
}

// configureOpsSlowProcessAnalysisPreflightWithPacer allows progress tests to inject milestone pacing deterministically.
func configureOpsSlowProcessAnalysisPreflightWithPacer(cmd *cobra.Command, request *ops.SlowProcessAnalysisRequest, pacer *opsProgressMilestonePacer) {
	if request == nil || request.SelectionMode != ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	request.Progress = func(event ops.ProgressEvent) {
		switch event.Kind {
		case ops.ProgressEventKindPreflight:
			if event.Preflight != nil {
				printOpsPreflightScope(cmd, *event.Preflight, channel)
			}
		case ops.ProgressEventKindPage:
			if event.Page != nil {
				printOpsSlowProcessAnalysisProgressMilestone(cmd, formatOpsPageProgress(*event.Page, "process instance(s)"), event, channel, pacer)
			}
		case ops.ProgressEventKindFrozenScope:
			if event.FrozenScope != nil {
				printOpsSlowProcessAnalysisProgressMilestone(cmd, formatOpsFrozenScopeProgress(*event.FrozenScope), event, channel, pacer)
			}
		case ops.ProgressEventKindETA:
			if event.ETA != nil && opsETAAllowed(*event.ETA) {
				printOpsSlowProcessAnalysisProgressMilestone(cmd, formatOpsETASampleWindow(*event.ETA), event, channel, pacer)
			}
		}
	}
	request.ConfirmPreflight = func(scope ops.PreflightScope) error {
		if !opsSlowProcessAnalysisPreflightConfirmationAllowed(channel) || !scope.RequiresConfirmation {
			return nil
		}
		prompt := strings.TrimSpace(scope.ConsequenceSummary.ConfirmationText)
		if prompt == "" {
			prompt = "Continue slow analysis?"
		}
		return confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt)
	}
}

// printOpsSlowProcessAnalysisProgress routes transient and verbose progress without touching command stdout.
func printOpsSlowProcessAnalysisProgress(cmd *cobra.Command, line string, channel ops.ProgressChannel) {
	printOpsSlowProcessAnalysisProgressMilestone(cmd, line, ops.ProgressEvent{}, channel, nil)
}

// printOpsSlowProcessAnalysisProgressMilestone keeps default progress transient while allowing paced durable milestones.
func printOpsSlowProcessAnalysisProgressMilestone(cmd *cobra.Command, line string, event ops.ProgressEvent, channel ops.ProgressChannel, pacer *opsProgressMilestonePacer) {
	if cmd == nil || strings.TrimSpace(line) == "" {
		return
	}
	if channel.TransientAllowed {
		logging.UpdateActivityWithImportance(cmd.Context(), line, logging.ActivityImportanceWorkflow)
	}
	if !opsSlowProcessAnalysisDurableProgressAllowed(channel) {
		if pacer != nil && pacer.AllowDurableMilestone(event, channel) {
			printOpsDurableLine(cmd, line, false)
		}
		return
	}
	printOpsDurableLine(cmd, line, false)
}

// printOpsPreflightScope writes durable preflight lines only to the command's stderr/activity channel.
func printOpsPreflightScope(cmd *cobra.Command, scope ops.PreflightScope, channel ops.ProgressChannel) {
	if cmd == nil || !channel.DurableAllowed || !channel.StderrAllowed {
		return
	}
	if scope.TenantContext != nil {
		attachTenantContext(cmd, *scope.TenantContext)
		printOpsTenantContext(cmd, *scope.TenantContext, channel)
	}
	printOpsPreflightLines(cmd, scope)
}

// opsSlowProcessAnalysisDurableProgressAllowed keeps page/counter detail behind verbose or debug while activity remains compact.
func opsSlowProcessAnalysisDurableProgressAllowed(channel ops.ProgressChannel) bool {
	return channel.DurableAllowed && channel.StderrAllowed && (channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug)
}

// opsSlowProcessAnalysisPreflightConfirmationAllowed limits prompts to human progress modes.
func opsSlowProcessAnalysisPreflightConfirmationAllowed(channel ops.ProgressChannel) bool {
	return channel.Mode == ops.ProgressModeHuman || channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug
}
