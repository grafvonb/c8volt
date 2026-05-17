// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

func commandShowTimezoneOffset(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	ctx := cmd.Context()
	if ctx == nil {
		return false
	}
	cfg, err := config.FromContext(ctx)
	if err != nil || cfg == nil {
		return false
	}
	return cfg.App.ShowTimezoneOffset
}

func formatCommandTime(cmd *cobra.Command, value time.Time) string {
	return toolx.FormatTime(value, commandShowTimezoneOffset(cmd))
}

func formatCommandTimestamp(cmd *cobra.Command, value string) string {
	return toolx.FormatTimestamp(value, commandShowTimezoneOffset(cmd))
}
