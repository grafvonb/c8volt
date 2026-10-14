// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"strings"
	"testing"
)

// TestCoverageManifest records manifest parity independently from inventory
// evidence so both common release categories must complete.
func TestCoverageManifest(t *testing.T) {
	registerReleaseScenario(t)
	doc := readCommandInventory(t, "coverage-manifest")
	report := validateCoverageManifest(flattenCapabilities(doc.Commands), commandCoverageManifest)
	writeJSON(t, "coverage.json", report)

	if report.ActualCount != expectedCommandInventoryCount {
		t.Fatalf("command inventory count = %d, want %d", report.ActualCount, expectedCommandInventoryCount)
	}
	if len(report.MissingPaths) > 0 || len(report.StalePaths) > 0 || len(report.Mismatches) > 0 {
		t.Fatalf("coverage manifest drift:\nmissing: %v\nstale: %v\nmismatches:\n%s", report.MissingPaths, report.StalePaths, strings.Join(report.Mismatches, "\n"))
	}
}
