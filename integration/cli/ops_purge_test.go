// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestOpsPurgeFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "ops purge", []string{
		"ops purge",
		"ops purge all-process-definitions",
		"ops purge orphan-process-instances",
		"ops purge process-instances-with-incidents",
	})
	runBehavioralCoverageScenarios(t, "ops purge")
}
