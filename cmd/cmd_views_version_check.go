// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/internal/domain"
)

// renderVersionCheckOutcome emits only validated bounded facts through the
// invocation logger, preserving its filtering, destination and activity routing.
func renderVersionCheckOutcome(log *slog.Logger, outcome domain.VersionCheckOutcome) {
	if !outcome.WarningNeeded() {
		return
	}
	if outcome.Status == domain.VersionCheckMismatched {
		log.Warn(fmt.Sprintf("version mismatch: configured %s, gateway %s; using %s", outcome.ConfiguredVersion, outcome.ObservedVersion, outcome.ConfiguredVersion))
		return
	}
	reason := "unavailable"
	switch outcome.FailureReason {
	case domain.VersionCheckAccessDenied:
		reason = "access denied"
	case domain.VersionCheckTimeout:
		reason = "timeout"
	case domain.VersionCheckInvalidVersion:
		reason = "invalid gateway version"
	case domain.VersionCheckCanceled:
		reason = "canceled"
	}
	log.Warn(fmt.Sprintf("version check: %s; using configured %s", reason, outcome.ConfiguredVersion))
}
