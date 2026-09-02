// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// configureOpsAPILatencyProgress wires aggregate service progress into command-owned output channels.
func configureOpsAPILatencyProgress(cmd *cobra.Command, request *ops.APILatencyRequest) *opsAPILatencyProgress {
	if request == nil {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	progress := &opsAPILatencyProgress{cmd: cmd, channel: channel}
	request.Progress = progress.Report
	return progress
}

// opsAPILatencyProgress owns aggregate stage progress for API-latency diagnostics.
type opsAPILatencyProgress struct {
	cmd     *cobra.Command
	channel ops.ProgressChannel
	closed  bool
}

// Report prints only aggregate stage progress allowed by the active output mode.
func (p *opsAPILatencyProgress) Report(event ops.ProgressEvent) {
	if p == nil || p.closed || event.Kind != ops.ProgressEventKindFrozenScope || event.FrozenScope == nil {
		return
	}
	printOpsSlowProcessAnalysisProgressMilestone(p.cmd, formatOpsFrozenScopeProgress(*event.FrozenScope), event, p.channel, nil)
}

// Close prevents late progress events from writing after command completion.
func (p *opsAPILatencyProgress) Close() {
	if p != nil {
		p.closed = true
	}
}
