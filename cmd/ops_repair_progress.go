// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// configureOpsRepairProgress installs command-owned repair progress rendering on the facade request.
func configureOpsRepairProgress(cmd *cobra.Command, request *ops.RepairRequest) {
	if request == nil {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	request.Progress = func(event ops.ProgressEvent) {
		printOpsRepairProgressEvent(cmd, event, channel)
	}
}

// printOpsRepairProgressEvent routes repair preflight and counters without writing to result stdout.
func printOpsRepairProgressEvent(cmd *cobra.Command, event ops.ProgressEvent, channel ops.ProgressChannel) {
	switch event.Kind {
	case ops.ProgressEventKindPreflight:
		if event.Preflight != nil {
			printOpsPreflightScope(cmd, *event.Preflight, channel)
		}
	case ops.ProgressEventKindPage:
		if event.Page != nil {
			printOpsSlowProcessAnalysisProgress(cmd, formatOpsPageProgress(*event.Page, ""), channel)
		}
	case ops.ProgressEventKindFrozenScope:
		if event.FrozenScope != nil {
			printOpsSlowProcessAnalysisProgress(cmd, formatProcessInstanceMutationFrozenProgress(*event.FrozenScope), channel)
		}
	}
}
