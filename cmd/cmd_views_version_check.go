// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/internal/domain"
)

// renderVersionCheckOutcome emits only validated bounded facts through the
// invocation logger, preserving its filtering, destination and activity routing.
func renderVersionCheckOutcome(log *slog.Logger, outcome domain.VersionCheckOutcome) {
	gateway := outcome.ObservedVersion
	if gateway == "" {
		gateway = "unknown"
	}
	summary := fmt.Sprintf("version check: result=%s configured=%s gateway=%s using=%s comparison=major.minor", outcome.Status, outcome.ConfiguredVersion, gateway, outcome.ConfiguredVersion)
	if outcome.Status == domain.VersionCheckFailed {
		summary += " reason=" + string(outcome.FailureReason)
	}
	if len(outcome.BrokerVersions) > 0 {
		summary += " broker_versions=[" + strings.Join(outcome.BrokerVersions, ",") + "]"
	}
	if outcome.BrokerCount > 0 {
		summary += fmt.Sprintf(" brokers=%d", outcome.BrokerCount)
	}
	if outcome.ClusterSize > 0 {
		summary += fmt.Sprintf(" cluster_size=%d", outcome.ClusterSize)
	}
	if outcome.PartitionsCount > 0 {
		summary += fmt.Sprintf(" partitions=%d", outcome.PartitionsCount)
	}
	if outcome.ReplicationFactor > 0 {
		summary += fmt.Sprintf(" replication_factor=%d", outcome.ReplicationFactor)
	}
	log.Debug(summary)
	if !outcome.WarningNeeded() {
		return
	}
	if outcome.Status == domain.VersionCheckMismatched {
		log.Warn(fmt.Sprintf("version mismatch: configured %s, gateway %s; results may be incorrect; match config to gateway unless mismatch is intentional", outcome.ConfiguredVersion, outcome.ObservedVersion))
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
