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

// TestRealStateGapFamily keeps real-state planning gaps in specs instead of
// runtime-generated proposal evidence.
func TestRealStateGapFamily(t *testing.T) {
	t.Run("gap artifact", validateRealStateGapArtifact)
	t.Run("coverage matrix", validateRealStateCoverageMatrix)
	t.Run("follow-up roadmap", validateRealStateFollowUpRoadmap)
}

// TestRealStateGapArtifactDocumentsCurrentPrerequisites keeps the older focused
// helper name available for direct compile-time validation commands.
func TestRealStateGapArtifactDocumentsCurrentPrerequisites(t *testing.T) {
	validateRealStateGapArtifact(t)
}

func validateRealStateGapArtifact(t *testing.T) {
	t.Helper()
	content := readRealStateSpecArtifact(t, "gaps.md")
	rejectRuntimeProposalArtifactReferences(t, "gaps.md", content)

	table := requireMarkdownTable(t, content, []string{
		"Topic",
		"Gap Type",
		"Required State Or Capability",
		"Blocked Proof",
		"Affected Commands",
		"Runtime Behavior Until Closed",
		"Affected Versions",
	})
	requiredTopics := []string{
		"Job timeout mutation",
		"BPMN error mutation",
		"BPMN error setup",
		"Listener variants",
		"Process-definition and orphan purge candidates",
		"Durable standalone resolve candidate",
		"Repair partial failures",
		"Expect state-only identity",
	}
	rowsByTopic := mapRowsByColumn(t, table, "Topic")
	for topic, row := range rowsByTopic {
		requireNonEmptyMarkdownCell(t, topic, row, "Gap Type")
		requireNonEmptyMarkdownCell(t, topic, row, "Required State Or Capability")
		requireNonEmptyMarkdownCell(t, topic, row, "Blocked Proof")
		requireNonEmptyMarkdownCell(t, topic, row, "Affected Commands")
		requireNonEmptyMarkdownCell(t, topic, row, "Runtime Behavior Until Closed")
		if !strings.Contains(row["Affected Versions"], realStateTargetVersion) {
			t.Fatalf("gaps.md topic %q affected versions = %q, want %q", topic, row["Affected Versions"], realStateTargetVersion)
		}
	}
	for _, topic := range requiredTopics {
		if _, ok := rowsByTopic[topic]; !ok {
			t.Fatalf("gaps.md missing topic %q", topic)
		}
	}
	if !strings.Contains(content, "future Camunda minor releases") {
		t.Fatal("gaps.md must describe how future Camunda minor releases extend affected-version rows")
	}
}

func validateRealStateCoverageMatrix(t *testing.T) {
	t.Helper()
	content := readRealStateSpecArtifact(t, "coverage-matrix.md")
	rejectRuntimeProposalArtifactReferences(t, "coverage-matrix.md", content)

	table := requireMarkdownTable(t, content, []string{
		"Topic",
		"Current Evidence Level",
		"Target Real-State Proof",
		"First Follow-Up",
	})
	requiredTopics := []string{
		"Gap artifact validation",
		"Consolidated follow-up roadmap",
		"Real `get job` rows",
		"Job retries, timeout, fail, no-wait",
		"`update job --throw-bpmn-error`",
		"Incidents with related jobs",
		"Listener jobs and `--with-listeners`",
		"Deterministic retention candidates",
		"Real purge semantics",
		"Process-definition delete semantics",
		"Cancel/delete/resolve post-state",
		"Partial failure and fail-fast",
		"Ops report parity",
		"Pipeline semantics",
		"Version extensibility",
	}
	rowsByTopic := mapRowsByColumn(t, table, "Topic")
	for topic, row := range rowsByTopic {
		requireNonEmptyMarkdownCell(t, topic, row, "Target Real-State Proof")
		requireNonEmptyMarkdownCell(t, topic, row, "First Follow-Up")
		if status := coverageEvidenceStatus(row["Current Evidence Level"]); status == "" {
			t.Fatalf("coverage-matrix.md topic %q has invalid evidence level %q", topic, row["Current Evidence Level"])
		}
	}
	for _, topic := range requiredTopics {
		if _, ok := rowsByTopic[topic]; !ok {
			t.Fatalf("coverage-matrix.md missing topic %q", topic)
		}
	}
	for _, status := range allowedCoverageEvidenceStatuses() {
		if !strings.Contains(content, status) {
			t.Fatalf("coverage-matrix.md evidence rules missing status %q", status)
		}
	}
}

func validateRealStateFollowUpRoadmap(t *testing.T) {
	t.Helper()
	content := readRealStateSpecArtifact(t, "follow-ups.md")
	rejectRuntimeProposalArtifactReferences(t, "follow-ups.md", content)

	table := requireMarkdownTable(t, content, []string{
		"Group",
		"Follow-Up Candidate",
		"Source Context",
		"Blocks",
		"Suggested First Spec",
	})
	requiredGroups := []string{
		"Embedded BPMN assets",
		"c8volt setup commands",
		"Product output contract",
		"Ops report semantics",
		"Pipeline semantics",
	}
	seenGroups := map[string]struct{}{}
	for _, row := range table.rows {
		group := row["Group"]
		seenGroups[group] = struct{}{}
		requireNonEmptyMarkdownCell(t, group, row, "Follow-Up Candidate")
		requireNonEmptyMarkdownCell(t, group, row, "Source Context")
		requireNonEmptyMarkdownCell(t, group, row, "Blocks")
		requireNonEmptyMarkdownCell(t, group, row, "Suggested First Spec")
	}
	for _, group := range requiredGroups {
		if _, ok := seenGroups[group]; !ok {
			t.Fatalf("follow-ups.md missing group %q", group)
		}
	}
	for _, phrase := range []string{
		"255 and 256 artifacts remain historical context",
		"not the legacy runtime-output pattern",
		"update `coverage-matrix.md`, `gaps.md`, and this file",
	} {
		if !strings.Contains(content, phrase) {
			t.Fatalf("follow-ups.md missing maintenance phrase %q", phrase)
		}
	}
}

type markdownTable struct {
	headers []string
	rows    []map[string]string
}

func readRealStateSpecArtifact(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(suite.repoRoot, "specs", "257-c89-real-state-integration", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read real-state artifact %s: %v", path, err)
	}
	return string(data)
}

func rejectRuntimeProposalArtifactReferences(t *testing.T, artifact string, content string) {
	t.Helper()
	for _, forbidden := range []string{"proposals-command.json", "proposals-embedded-bpmn.json", "real-state-proposals"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("%s must not depend on runtime proposal artifact %q", artifact, forbidden)
		}
	}
}

func requireMarkdownTable(t *testing.T, content string, wantHeaders []string) markdownTable {
	t.Helper()
	lines := strings.Split(content, "\n")
	for i := 0; i+1 < len(lines); i++ {
		headers := markdownCells(lines[i])
		if !sameStringSlice(headers, wantHeaders) || !isMarkdownSeparator(markdownCells(lines[i+1])) {
			continue
		}
		table := markdownTable{headers: headers}
		for j := i + 2; j < len(lines); j++ {
			cells := markdownCells(lines[j])
			if len(cells) == 0 {
				break
			}
			if len(cells) != len(headers) {
				t.Fatalf("markdown table row has %d cells, want %d: %q", len(cells), len(headers), lines[j])
			}
			row := make(map[string]string, len(headers))
			for idx, header := range headers {
				row[header] = cells[idx]
			}
			table.rows = append(table.rows, row)
		}
		if len(table.rows) == 0 {
			t.Fatalf("markdown table %v has no rows", wantHeaders)
		}
		return table
	}
	t.Fatalf("markdown table with headers %v not found", wantHeaders)
	return markdownTable{}
}

func markdownCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return nil
	}
	raw := strings.Split(strings.Trim(trimmed, "|"), "|")
	cells := make([]string, 0, len(raw))
	for _, cell := range raw {
		cells = append(cells, strings.TrimSpace(cell))
	}
	return cells
}

func isMarkdownSeparator(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if strings.Trim(cell, " :-") != "" || !strings.Contains(cell, "-") {
			return false
		}
	}
	return true
}

func mapRowsByColumn(t *testing.T, table markdownTable, column string) map[string]map[string]string {
	t.Helper()
	rowsByTopic := make(map[string]map[string]string, len(table.rows))
	for _, row := range table.rows {
		key := strings.TrimSpace(row[column])
		if key == "" {
			t.Fatalf("markdown table has empty %s cell: %+v", column, row)
		}
		if _, exists := rowsByTopic[key]; exists {
			t.Fatalf("markdown table has duplicate %s %q", column, key)
		}
		rowsByTopic[key] = row
	}
	return rowsByTopic
}

func requireNonEmptyMarkdownCell(t *testing.T, topic string, row map[string]string, column string) {
	t.Helper()
	if strings.TrimSpace(row[column]) == "" {
		t.Fatalf("topic %q has empty %s cell", topic, column)
	}
}

func coverageEvidenceStatus(value string) string {
	trimmed := strings.TrimSpace(value)
	for _, status := range allowedCoverageEvidenceStatuses() {
		if trimmed == status || strings.HasPrefix(trimmed, status+":") || strings.HasPrefix(trimmed, status+" ") {
			return status
		}
	}
	return ""
}

func allowedCoverageEvidenceStatuses() []string {
	return []string{
		"live-covered",
		"partially live-covered",
		"dry-run-covered",
		"skipped-prerequisite",
		"no-match only",
		"not yet started",
	}
}

func sameStringSlice(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
