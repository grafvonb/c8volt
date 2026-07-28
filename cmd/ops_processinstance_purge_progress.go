// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

func configureOpsExecuteRetentionPolicyProgress(cmd *cobra.Command, request *ops.RetentionPolicyRequest) {
	if request == nil {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	request.Progress = func(event ops.ProgressEvent) {
		printOpsProcessInstancePurgeProgressEvent(cmd, event, channel)
	}
}

func configureOpsPurgeOrphanProcessInstancesProgress(cmd *cobra.Command, request *ops.OrphanPurgeRequest) {
	if request == nil {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	request.Progress = func(event ops.ProgressEvent) {
		printOpsProcessInstancePurgeProgressEvent(cmd, event, channel)
	}
}

func configureOpsPurgeProcessInstancesWithIncidentsProgress(cmd *cobra.Command, request *ops.IncidentPurgeRequest) {
	if request == nil {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	request.Progress = func(event ops.ProgressEvent) {
		printOpsProcessInstancePurgeProgressEvent(cmd, event, channel)
	}
}

func printOpsProcessInstancePurgeProgressEvent(cmd *cobra.Command, event ops.ProgressEvent, channel ops.ProgressChannel) {
	switch event.Kind {
	case ops.ProgressEventKindPreflight:
		if event.Preflight != nil {
			printOpsPreflightScope(cmd, *event.Preflight, channel)
		}
	case ops.ProgressEventKindPage:
		if event.Page != nil {
			printOpsSlowProcessAnalysisProgress(cmd, formatOpsPageProgress(*event.Page, "process instance(s)"), channel)
		}
	case ops.ProgressEventKindFrozenScope:
		if event.FrozenScope != nil {
			printOpsSlowProcessAnalysisProgress(cmd, formatProcessInstanceMutationFrozenProgress(*event.FrozenScope), channel)
		}
	}
}
