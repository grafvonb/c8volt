// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/cluster"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

// versionCheckContextKey keeps outcome access local to this invocation.
type versionCheckContextKey struct{}

// installVersionCheck composes the lazy service after authentication; no I/O is
// performed until a generated operational client reaches its pre-request gate.
func installVersionCheck(ctx context.Context, cfg *config.Config, client *http.Client, log *slog.Logger) context.Context {
	if flagNoVersionCheck {
		return ctx
	}
	checker := cluster.NewVersionCheck(cfg, client, log, func(outcome domain.VersionCheckOutcome) { renderVersionCheckOutcome(log, outcome) })
	ctx = context.WithValue(ctx, versionCheckContextKey{}, checker)
	return httpc.WithBeforeRequestGate(ctx, checker)
}

// versionCheckWarned uses outcome facts, never logger visibility, for deduplication.
func versionCheckWarned(ctx context.Context) bool {
	checker, ok := ctx.Value(versionCheckContextKey{}).(*cluster.VersionCheck)
	if !ok {
		return false
	}
	outcome, complete := checker.Outcome()
	return complete && outcome.WarningNeeded()
}
