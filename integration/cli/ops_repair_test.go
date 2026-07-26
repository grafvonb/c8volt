// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestOpsRepairFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "ops repair", []string{
		"ops repair",
		"ops repair incident",
		"ops repair process-instance",
	})
	runBehavioralCoverageScenarios(t, "ops repair")
}
