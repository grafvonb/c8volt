// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestUpdateFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "update", []string{
		"update",
		"update job",
		"update process-instance",
	})
}
