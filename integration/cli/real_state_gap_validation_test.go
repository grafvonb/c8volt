// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRealStateGapArtifactDocumentsCurrentPrerequisites keeps real-state gap
// tracking in specs instead of runtime-generated proposal evidence.
func TestRealStateGapArtifactDocumentsCurrentPrerequisites(t *testing.T) {
	path := filepath.Join(suite.repoRoot, "specs", "257-c89-real-state-integration", "gaps.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read real-state gap artifact: %v", err)
	}
	content := string(data)
	required := []string{
		"Job timeout mutation",
		"BPMN error mutation",
		"BPMN error setup",
		"Listener variants",
		"Deterministic retention",
		"Purge and delete candidates",
		"Repair partial failures",
		"Expect state-only identity",
		"Runtime Behavior Until Closed",
		realStateTargetVersion,
	}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Fatalf("gaps.md missing %q", want)
		}
	}
	if strings.Contains(content, "proposals-command.json") || strings.Contains(content, "proposals-embedded-bpmn.json") {
		t.Fatal("gaps.md must not depend on runtime proposal JSON files")
	}
}
