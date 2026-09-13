// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth"
	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// installRemoteCommandServices wires invocation-scoped HTTP diagnostics before authentication can issue bootstrap traffic.
func installRemoteCommandServices(ctx context.Context, cfg *config.Config, log *slog.Logger) (context.Context, error) {
	activity := logging.ActivityFromContext(ctx)
	httpSvc, err := httpc.New(
		cfg,
		log,
		httpc.WithCookieJar(),
		httpc.WithActivitySink(activity),
		httpc.WithDiagnostics(),
	)
	if err != nil {
		return ctx, bootstrapLocalPrecondition(fmt.Errorf("create http service: %w", err))
	}
	ator, err := auth.BuildAuthenticator(cfg, httpSvc.Client(), log)
	if err != nil {
		return ctx, bootstrapLocalPrecondition(fmt.Errorf("create authenticator: %w", err))
	}
	if err := ator.Init(ctx); err != nil {
		return ctx, normalizeBootstrapError(fmt.Errorf("initialize authenticator: %w", err))
	}
	httpSvc.InstallAuthEditor(ator.Editor())
	ctx = httpSvc.ToContext(ctx)
	ctx = authenticator.ToContext(ctx, ator)
	return ctx, nil
}

func automationModeEnabled(cmd *cobra.Command) bool {
	if cmd != nil {
		if ctx := cmd.Context(); ctx != nil {
			if cfg, err := config.FromContext(ctx); err == nil && cfg != nil {
				return cfg.App.Automation
			}
		}
		if flag := cmd.Flags().Lookup("automation"); flag != nil {
			if value, err := strconv.ParseBool(flag.Value.String()); err == nil {
				return value
			}
		}
	}
	return flagCmdAutomation
}

func indicatorEnabled(cmd *cobra.Command, cfg *config.Config) bool {
	if flagNoIndicator || flagQuiet || flagViewAsJson || flagViewKeysOnly {
		return false
	}
	if cfg != nil {
		if strings.EqualFold(cfg.Log.Format, "json") {
			return false
		}
		return !cfg.App.Automation
	}
	return !automationModeEnabled(cmd)
}
