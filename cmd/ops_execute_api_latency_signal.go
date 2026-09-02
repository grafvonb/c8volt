// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// newOpsExecuteAPILatencyInterruptContext installs the active-run interrupt signal hook.
var newOpsExecuteAPILatencyInterruptContext = func(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt)
}

// withOpsExecuteAPILatencyInterruptContext scopes operator interrupts to the active mutation window.
func withOpsExecuteAPILatencyInterruptContext(cmd *cobra.Command, request ops.APILatencyRequest, run func() (ops.APILatencyResult, error)) (ops.APILatencyResult, error) {
	if request.DryRun || cmd == nil {
		return run()
	}
	parent := cmd.Context()
	ctx, stop := newOpsExecuteAPILatencyInterruptContext(parent)
	cmd.SetContext(ctx)
	defer func() {
		stop()
		cmd.SetContext(parent)
	}()
	return run()
}
