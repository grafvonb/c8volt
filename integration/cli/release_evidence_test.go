// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/integration/internal/validation"
)

const (
	envITRunID           = "C8VOLT_IT_RUN_ID"
	envITSliceID         = "C8VOLT_IT_SLICE_ID"
	envITEvidenceFile    = "C8VOLT_IT_EVIDENCE_FILE"
	envITObservedVersion = "C8VOLT_IT_OBSERVED_VERSION"
)

type releaseEvidenceControl struct {
	runID           string
	sliceID         string
	profile         string
	observedVersion string
	evidenceFile    string
}

type releaseScenarioResult struct {
	failed  bool
	skipped bool
}

var normalizedReleaseEvidence = struct {
	sync.Mutex
	records    []validation.ScenarioEvidence
	registered map[string]struct{}
}{registered: map[string]struct{}{}}

// registerReleaseScenario binds one catalog-owned top-level test to runner
// provenance and records its final testing outcome during cleanup.
func registerReleaseScenario(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(os.Getenv(envITEvidenceFile)) == "" {
		return
	}
	control, err := releaseEvidenceControlFromEnvironment()
	if err != nil {
		t.Fatalf("release evidence controls: %v", err)
	}
	scenarioID := strings.SplitN(t.Name(), "/", 2)[0]
	requirement, err := releaseRequirement(scenarioID, control.sliceID)
	if err != nil {
		t.Fatalf("release evidence scenario: %v", err)
	}

	normalizedReleaseEvidence.Lock()
	if _, exists := normalizedReleaseEvidence.registered[scenarioID]; exists {
		normalizedReleaseEvidence.Unlock()
		return
	}
	normalizedReleaseEvidence.registered[scenarioID] = struct{}{}
	normalizedReleaseEvidence.Unlock()

	t.Cleanup(func() {
		record := buildReleaseScenarioEvidence(control, requirement, releaseScenarioResult{
			failed: t.Failed(), skipped: t.Skipped(),
		})
		normalizedReleaseEvidence.Lock()
		normalizedReleaseEvidence.records = append(normalizedReleaseEvidence.records, record)
		normalizedReleaseEvidence.Unlock()
	})
}

// releaseEvidenceControlFromEnvironment validates the runner-owned child
// contract and refuses ambiguous multi-profile evidence.
func releaseEvidenceControlFromEnvironment() (releaseEvidenceControl, error) {
	profiles := splitProfileNames(os.Getenv(envITProfiles))
	control := releaseEvidenceControl{
		runID:           strings.TrimSpace(os.Getenv(envITRunID)),
		sliceID:         strings.TrimSpace(os.Getenv(envITSliceID)),
		observedVersion: strings.TrimSpace(os.Getenv(envITObservedVersion)),
		evidenceFile:    strings.TrimSpace(os.Getenv(envITEvidenceFile)),
	}
	if len(profiles) != 1 {
		return releaseEvidenceControl{}, fmt.Errorf("runner child requires exactly one profile, got %d", len(profiles))
	}
	control.profile = profiles[0]
	if control.runID == "" || control.sliceID == "" || control.observedVersion == "" || control.evidenceFile == "" {
		return releaseEvidenceControl{}, fmt.Errorf("runner provenance is incomplete")
	}
	if !filepath.IsAbs(control.evidenceFile) || filepath.Clean(control.evidenceFile) != control.evidenceFile {
		return releaseEvidenceControl{}, fmt.Errorf("evidence file must be a clean absolute path")
	}
	return control, nil
}

// releaseRequirement resolves proof levels from the embedded catalog so the
// harness cannot drift from the reducer's stable scenario identities.
func releaseRequirement(scenarioID string, sliceID string) (validation.CoverageRequirement, error) {
	catalog, _, err := validation.LoadDefaultPolicy()
	if err != nil {
		return validation.CoverageRequirement{}, err
	}
	for _, requirement := range catalog.Requirements {
		if requirement.ScenarioID != scenarioID {
			continue
		}
		if requirement.Slice != sliceID {
			return validation.CoverageRequirement{}, fmt.Errorf("scenario %q belongs to slice %q, not %q", scenarioID, requirement.Slice, sliceID)
		}
		return requirement, nil
	}
	return validation.CoverageRequirement{}, fmt.Errorf("scenario %q is not in the release catalog", scenarioID)
}

// buildReleaseScenarioEvidence converts testing outcomes without allowing a
// skip or failure to inherit a passing classification.
func buildReleaseScenarioEvidence(control releaseEvidenceControl, requirement validation.CoverageRequirement, result releaseScenarioResult) validation.ScenarioEvidence {
	record := validation.ScenarioEvidence{
		SchemaVersion: validation.SchemaVersion,
		RunID:         control.runID, SliceID: control.sliceID,
		ScenarioID: requirement.ScenarioID, Profile: control.profile,
		ObservedVersion: control.observedVersion,
		Outcome:         validation.OutcomePassed, EvidenceLevel: requirement.MinimumEvidence,
	}
	switch {
	case result.skipped:
		record.Outcome = validation.OutcomeSkipped
		record.FailureClass = validation.FailureIncompleteEvidence
		record.Reason = "catalog-owned test skipped"
	case result.failed:
		record.Outcome = validation.OutcomeFailed
		record.FailureClass = validation.FailureHarness
		record.Reason = "catalog-owned test failed; inspect retained child events and legacy reports"
	}
	return record
}

// writeNormalizedReleaseEvidence persists exactly one JSON array for the
// runner after all registered test cleanup callbacks have completed.
func writeNormalizedReleaseEvidence() error {
	path := strings.TrimSpace(os.Getenv(envITEvidenceFile))
	if path == "" {
		return nil
	}
	if _, err := releaseEvidenceControlFromEnvironment(); err != nil {
		return err
	}
	normalizedReleaseEvidence.Lock()
	records := append([]validation.ScenarioEvidence(nil), normalizedReleaseEvidence.records...)
	normalizedReleaseEvidence.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create normalized evidence directory: %w", err)
	}
	if err := writeJSONFile(path, records); err != nil {
		return fmt.Errorf("write normalized evidence: %w", err)
	}
	return nil
}

// TestBuildReleaseScenarioEvidencePreservesResults verifies fake pass, skip,
// and failure outcomes retain provenance and never collapse to success.
func TestBuildReleaseScenarioEvidencePreservesResults(t *testing.T) {
	control := releaseEvidenceControl{runID: "run-1", sliceID: "baseline-get", profile: "dev810", observedVersion: "8.10.3"}
	requirement := validation.CoverageRequirement{ScenarioID: "TestGetFamily", MinimumEvidence: validation.EvidenceReadOnly}
	cases := []struct {
		name      string
		result    releaseScenarioResult
		want      validation.Outcome
		wantClass validation.FailureClass
	}{
		{name: "passed", want: validation.OutcomePassed},
		{name: "skipped", result: releaseScenarioResult{skipped: true}, want: validation.OutcomeSkipped, wantClass: validation.FailureIncompleteEvidence},
		{name: "failed", result: releaseScenarioResult{failed: true}, want: validation.OutcomeFailed, wantClass: validation.FailureHarness},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := buildReleaseScenarioEvidence(control, requirement, tc.result)
			if record.Outcome != tc.want || record.FailureClass != tc.wantClass {
				t.Fatalf("outcome/class = %q/%q, want %q/%q", record.Outcome, record.FailureClass, tc.want, tc.wantClass)
			}
			if record.RunID != control.runID || record.SliceID != control.sliceID || record.Profile != control.profile || record.ObservedVersion != control.observedVersion {
				t.Fatalf("record lost runner provenance: %+v", record)
			}
		})
	}
}
