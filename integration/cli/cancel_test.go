// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestCancelFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "cancel", []string{
		"cancel",
		"cancel process-instance",
	})
	runBehavioralCoverageScenarios(t, "cancel")
}
