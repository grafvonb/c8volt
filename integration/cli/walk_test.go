// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestWalkFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "walk", []string{
		"walk",
		"walk process-instance",
	})
}
