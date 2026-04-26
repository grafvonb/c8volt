// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"log/slog"

	"github.com/grafvonb/c8volt/internal/services"
)

func VerboseLog(ctx context.Context, callCfg *services.CallCfg, log *slog.Logger, msg string, args ...any) {
	if ctx == nil || callCfg == nil || !callCfg.Verbose || log == nil {
		return
	}
	log.InfoContext(ctx, msg, args...)
}
