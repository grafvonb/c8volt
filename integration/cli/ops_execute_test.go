// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestOpsExecuteFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "ops execute", []string{
		"ops execute",
		"ops execute retention-policy",
		"ops execute smoke-test",
	})
}
