// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// appendExplicitLargeWorkProgressOption installs stdout-safe progress routing for explicit-count or explicit-key workflows.
func appendExplicitLargeWorkProgressOption(cmd *cobra.Command, opts []processOptions.FacadeOption) []processOptions.FacadeOption {
	out := append([]processOptions.FacadeOption{}, opts...)
	return append(out, processOptions.WithProgress(func(event processOptions.ProgressEvent) {
		if event.Kind != processOptions.ProgressEventKindFrozenScope || event.FrozenScope == nil {
			return
		}
		progress := opsFrozenScopeProgressFromProcessOption(*event.FrozenScope)
		printExplicitLargeWorkProgressEvent(cmd, ops.ProgressEvent{
			Kind:        ops.ProgressEventKindFrozenScope,
			FrozenScope: &progress,
		})
	}))
}

// configureOpsExecuteSmokeTestProgress installs command-owned smoke-test progress rendering on the facade request.
func configureOpsExecuteSmokeTestProgress(cmd *cobra.Command, request *ops.SmokeTestRequest) {
	if request == nil {
		return
	}
	request.Progress = func(event ops.ProgressEvent) {
		printExplicitLargeWorkProgressEvent(cmd, event)
	}
}

// printExplicitLargeWorkProgressEvent renders exact explicit-work counters using the shared command progress gate.
func printExplicitLargeWorkProgressEvent(cmd *cobra.Command, event ops.ProgressEvent) {
	if event.Kind != ops.ProgressEventKindFrozenScope || event.FrozenScope == nil {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	printOpsSlowProcessAnalysisProgress(cmd, formatProcessInstanceMutationFrozenProgress(*event.FrozenScope), channel)
}
