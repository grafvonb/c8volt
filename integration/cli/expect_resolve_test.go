// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

func TestExpectResolveFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "expect", []string{
		"expect",
		"expect process-instance",
	})
	runFamilyCoverageScenarios(t, "resolve", []string{
		"resolve",
		"resolve incident",
		"resolve process-instance",
	})
}
