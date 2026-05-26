// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

func renderOpsProcessInstanceDependencyExpansion(cmd *cobra.Command, candidateCount int, affectedCount int) {
	additionalCount := affectedCount - candidateCount
	if additionalCount <= 0 {
		return
	}
	renderHumanLine(cmd, "dependency expansion: %d additional process instance(s) due to dependencies", additionalCount)
}
