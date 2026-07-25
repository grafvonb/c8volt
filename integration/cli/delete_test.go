// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestDeleteFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "delete", []string{
		"delete",
		"delete process-definition",
		"delete process-instance",
	})
}
