// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

// TestDirtyClusterProcessInstanceAssertionIgnoresUnrelatedResults verifies seeded checks do not assume exact global counts.
func TestDirtyClusterProcessInstanceAssertionIgnoresUnrelatedResults(t *testing.T) {
	requireNoExactGlobalCountAssertion(t, "assert seeded keys are present among unrelated search results")
	results := []seededProcessInstance{
		{Key: "100", BpmnProcessID: "preexisting"},
		{Key: "200", BpmnProcessID: "suite-owned"},
		{Key: "300", BpmnProcessID: "preexisting"},
	}
	if missing := missingSeededProcessInstanceKeys([]string{"200"}, results); len(missing) > 0 {
		t.Fatalf("seeded key reported missing among unrelated results: %v", missing)
	}
}

// TestDirtyClusterProcessInstanceAssertionReportsOnlyMissingSeededKeys keeps failure output focused on suite-owned data.
func TestDirtyClusterProcessInstanceAssertionReportsOnlyMissingSeededKeys(t *testing.T) {
	results := []seededProcessInstance{
		{Key: "100", BpmnProcessID: "preexisting"},
		{Key: "300", BpmnProcessID: "preexisting"},
	}
	missing := missingSeededProcessInstanceKeys([]string{"200", "300"}, results)
	assertStringSlicesEqual(t, missing, []string{"200"})
}

// missingSeededProcessInstanceKeys returns seeded keys absent from a dirty-cluster search result set.
func missingSeededProcessInstanceKeys(want []string, got []seededProcessInstance) []string {
	seen := make(map[string]struct{}, len(got))
	for _, item := range got {
		if item.Key != "" {
			seen[item.Key] = struct{}{}
		}
	}
	var missing []string
	for _, key := range want {
		if _, ok := seen[key]; !ok {
			missing = append(missing, key)
		}
	}
	return missing
}
