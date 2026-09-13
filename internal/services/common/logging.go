// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

func VerboseLog(ctx context.Context, callCfg *services.CallCfg, log *slog.Logger, msg string, args ...any) {
	if ctx == nil || callCfg == nil || !callCfg.Verbose || callCfg.SuppressWorkflowDetailLogs || log == nil {
		return
	}
	log.InfoContext(ctx, msg, args...)
}

// VerboseProcessInstanceWaitLog explains one process-instance workflow wait without owning its polling lifecycle.
func VerboseProcessInstanceWaitLog(ctx context.Context, callCfg *services.CallCfg, cfg *config.Config, log *slog.Logger, phase, root string, scope []string, states []d.State) {
	if ctx == nil || cfg == nil {
		return
	}
	timeout := EffectiveProcessInstanceWaitTimeout(ctx, cfg.App.Backoff.Timeout)
	VerboseLog(ctx, callCfg, log, fmt.Sprintf(
		"pi wait: phase=%s root=%s scope=%v states=%v timeout=%s backoff=%s initial_delay=%s max_retries=%d",
		phase, root, scope, states, timeout, cfg.App.Backoff.Strategy, cfg.App.Backoff.InitialDelay, cfg.App.Backoff.MaxRetries,
	))
}

// EffectiveProcessInstanceWaitTimeout returns the shorter configured or parent
// context budget so diagnostics describe the wait that can actually occur.
func EffectiveProcessInstanceWaitTimeout(ctx context.Context, configured time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if configured <= 0 || remaining < configured {
			return max(remaining, 0)
		}
	}
	return configured
}

// ProcessDefinitionStatsActivity returns the user-facing activity text for process-definition statistics.
func ProcessDefinitionStatsActivity(bpmnProcessId, key string) string {
	switch {
	case bpmnProcessId != "" && key != "":
		return fmt.Sprintf("getting pd stats %s (%s)", bpmnProcessId, key)
	case bpmnProcessId != "":
		return fmt.Sprintf("getting pd stats %s", bpmnProcessId)
	case key != "":
		return fmt.Sprintf("getting pd stats %s", key)
	default:
		return "getting pd stats"
	}
}
