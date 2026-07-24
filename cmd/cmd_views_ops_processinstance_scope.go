// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// renderOpsDiscoveryStatus exposes user-limited discovery in compact output
// and keeps complete page details verbose-only across ops workflows.
func renderOpsDiscoveryStatus(cmd *cobra.Command, status ops.DiscoveryScopeStatus) {
	if status.Limited {
		renderHumanLine(cmd, "discovery user-limited: limit %d; pages %d; batch size %d", status.Limit, status.Pages, status.BatchSize)
		return
	}
	if status.Complete && flagVerbose {
		renderHumanLine(cmd, "discovery complete: pages %d; batch size %d", status.Pages, status.BatchSize)
	}
}

func renderOpsProcessInstanceDependencyExpansion(cmd *cobra.Command, candidateCount int, affectedCount int) {
	additionalCount := affectedCount - candidateCount
	if additionalCount <= 0 {
		return
	}
	renderHumanLine(cmd, "dependency expansion: %d additional process instance(s) due to dependencies", additionalCount)
}
