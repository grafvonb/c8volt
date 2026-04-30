// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/internal/services"
)

func VerboseLog(ctx context.Context, callCfg *services.CallCfg, log *slog.Logger, msg string, args ...any) {
	if ctx == nil || callCfg == nil || !callCfg.Verbose || log == nil {
		return
	}
	log.InfoContext(ctx, msg, args...)
}

// ProcessDefinitionStatsActivity returns the user-facing activity text for process-definition statistics.
func ProcessDefinitionStatsActivity(bpmnProcessId, key string) string {
	switch {
	case bpmnProcessId != "" && key != "":
		return fmt.Sprintf("retrieving process definition stats for %s (%s)", bpmnProcessId, key)
	case bpmnProcessId != "":
		return fmt.Sprintf("retrieving process definition stats for %s", bpmnProcessId)
	case key != "":
		return fmt.Sprintf("retrieving process definition stats for key %s", key)
	default:
		return "retrieving process definition stats"
	}
}
