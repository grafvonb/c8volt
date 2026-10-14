// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestReduceStrictReleasePassesOnlyCompleteEvidence proves optional named gaps
// remain visible and prevent a complete-coverage claim.
func TestReduceStrictReleasePassesOnlyCompleteEvidence(t *testing.T) {
	t.Parallel()
	input := completeTestInput(t)
	input.Evidence = input.Evidence[:1]

	summary, err := Reduce(input)
	require.NoError(t, err)
	require.Equal(t, VerdictPassedWithGaps, summary.Verdict)
	require.False(t, summary.CoverageComplete)
	require.Len(t, summary.Gaps, 1)
	require.Empty(t, summary.MissingRequirements)

	input = completeTestInput(t)
	summary, err = Reduce(input)
	require.NoError(t, err)
	require.Equal(t, VerdictPassed, summary.Verdict)
	require.True(t, summary.CoverageComplete)
}

// TestReduceRejectsMissingSkippedAndWeakRequiredEvidence covers the main ways a
// required family can otherwise appear falsely green.
func TestReduceRejectsMissingSkippedAndWeakRequiredEvidence(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*ReductionInput){
		"missing": func(input *ReductionInput) { input.Evidence = input.Evidence[1:] },
		"skipped": func(input *ReductionInput) { input.Evidence[0].Outcome = OutcomeSkipped },
		"weak proof": func(input *ReductionInput) {
			input.Evidence[0].EvidenceLevel = EvidenceDryRun
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := completeTestInput(t)
			mutate(&input)
			summary, err := Reduce(input)
			require.NoError(t, err)
			require.Equal(t, VerdictFailed, summary.Verdict)
			require.False(t, summary.CoverageComplete)
		})
	}
}

// TestReduceRejectsAbsentAndMismatchedTargets prevents foreign-minor evidence
// from satisfying a strict requested release.
func TestReduceRejectsAbsentAndMismatchedTargets(t *testing.T) {
	t.Parallel()
	for name, targets := range map[string][]ValidationTarget{
		"absent":              nil,
		"mismatched observed": {{SchemaVersion: SchemaVersion, Profile: "dev810", ExpectedMinor: "8.10", ObservedVersion: "8.9.9", Ready: true}},
		"foreign expected":    {{SchemaVersion: SchemaVersion, Profile: "dev810", ExpectedMinor: "8.9", ObservedVersion: "8.9.9", Ready: true}},
		"unparseable":         {{SchemaVersion: SchemaVersion, Profile: "dev810", ExpectedMinor: "8.10", ObservedVersion: "unknown", Ready: true}},
		"credential route":    {{SchemaVersion: SchemaVersion, Profile: "dev810", ExpectedMinor: "8.10", ObservedVersion: "8.10.3", Ready: true, SanitizedRouteIdentity: "https://user:pass@example.test/v2"}},
	} {
		t.Run(name, func(t *testing.T) {
			input := completeTestInput(t)
			input.Targets = targets
			summary, err := Reduce(input)
			require.NoError(t, err)
			require.Equal(t, VerdictFailed, summary.Verdict)
			require.NotEmpty(t, summary.Failures)
		})
	}
}

// TestReduceRejectsUnknownDuplicateAndStaleEvidence preserves current-run
// provenance and unique completion records.
func TestReduceRejectsUnknownDuplicateAndStaleEvidence(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*ReductionInput){
		"unknown": func(input *ReductionInput) {
			input.Evidence = append(input.Evidence, ScenarioEvidence{SchemaVersion: SchemaVersion, RunID: "run-1", SliceID: "family", ScenarioID: "unknown", Profile: "dev810", Outcome: OutcomePassed, EvidenceLevel: EvidenceLivePostState})
		},
		"duplicate": func(input *ReductionInput) { input.Evidence = append(input.Evidence, input.Evidence[0]) },
		"stale":     func(input *ReductionInput) { input.Evidence[0].RunID = "older-run" },
		"foreign minor": func(input *ReductionInput) {
			input.Evidence[0].ObservedVersion = "8.9.9"
		},
		"unselected profile": func(input *ReductionInput) {
			input.Evidence[0].Profile = "other810"
		},
		"unsafe path": func(input *ReductionInput) {
			input.Evidence[0].ObservationPath = "../foreign.json"
		},
		"unknown gap": func(input *ReductionInput) {
			input.Evidence[1].Outcome = OutcomeBlocked
			input.Evidence[1].GapID = "runtime-waiver"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := completeTestInput(t)
			mutate(&input)
			summary, err := Reduce(input)
			require.NoError(t, err)
			require.Equal(t, VerdictFailed, summary.Verdict)
			require.NotEmpty(t, summary.Failures)
		})
	}
}

// TestReducePreservesChildFailureAndInterruptionStatus ensures later evidence
// cannot overwrite child failure and unfinished runs remain incomplete.
func TestReducePreservesChildFailureAndInterruptionStatus(t *testing.T) {
	t.Parallel()
	input := completeTestInput(t)
	exit := 1
	input.Slices[0].ExitCode = &exit
	input.Slices[0].State = SliceFailed
	summary, err := Reduce(input)
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, summary.Verdict)

	input = completeTestInput(t)
	input.Interrupted = true
	input.Evidence = input.Evidence[:1]
	summary, err = Reduce(input)
	require.NoError(t, err)
	require.Equal(t, VerdictIncomplete, summary.Verdict)

	input = completeTestInput(t)
	input.Slices[0].State = SliceRunning
	input.Slices[0].ExitCode = nil
	summary, err = Reduce(input)
	require.NoError(t, err)
	require.Equal(t, VerdictIncomplete, summary.Verdict)
}

// TestReduceExploratoryModeDoesNotInventMissingReleaseEvidence confirms an
// intentionally partial exploratory run is judged only on supplied records.
func TestReduceExploratoryModeDoesNotInventMissingReleaseEvidence(t *testing.T) {
	t.Parallel()
	input := completeTestInput(t)
	input.Request.Mode = ModeExploratory
	input.Request.RequestedMinor = ""
	input.Evidence = input.Evidence[:1]

	summary, err := Reduce(input)
	require.NoError(t, err)
	require.Equal(t, VerdictPassed, summary.Verdict)
	require.Empty(t, summary.MissingRequirements)
}

// TestReduceRedactsSecretsFromPersistedFindings covers credential assignments,
// bearer tokens, and credential-bearing URLs.
func TestReduceRedactsSecretsFromPersistedFindings(t *testing.T) {
	t.Parallel()
	input := completeTestInput(t)
	input.Evidence[0].Outcome = OutcomeFailed
	input.Evidence[0].FailureClass = FailureEnvironment
	input.Evidence[0].Reason = "token=abc Bearer xyz123 https://user:pass@example.test/v2"

	summary, err := Reduce(input)
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, summary.Verdict)
	encoded := summary.Failures[0].Reason
	require.NotContains(t, encoded, "abc")
	require.NotContains(t, encoded, "xyz123")
	require.NotContains(t, encoded, "user:pass")
	require.True(t, strings.Contains(encoded, "[REDACTED]"))
}

// testPolicy builds the smallest valid catalog with one required scenario and
// one named optional proof.
func testPolicy() (CoverageCatalog, GapCatalog) {
	catalog := CoverageCatalog{
		SchemaVersion: SchemaVersion,
		Revision:      "test-revision",
		Families: []CoverageFamily{{
			ID: "family", Slice: "family", ApplicableMinors: []string{"8.9", "8.10"}, MinimumScenarios: 1,
		}},
		Requirements: []CoverageRequirement{
			{ScenarioID: "required", Slice: "family", Family: "family", ApplicableMinors: []string{"8.9", "8.10"}, Required: true, MinimumEvidence: EvidenceLivePostState},
			{ScenarioID: "optional", Slice: "family", Family: "family", ApplicableMinors: []string{"8.9", "8.10"}, MinimumEvidence: EvidenceControlled, AcceptedGapID: "optional-gap"},
		},
	}
	gaps := GapCatalog{
		SchemaVersion: SchemaVersion,
		Revision:      catalog.Revision,
		Gaps: []PrerequisiteGap{{
			ID: "optional-gap", ScenarioIDs: []string{"optional"}, ApplicableMinors: []string{"8.9", "8.10"},
			MissingPrerequisite: "fixture", BlockedProof: "additional controlled proof", Rationale: "test", ReviewStatus: "accepted",
		}},
	}
	return catalog, gaps
}

// completeTestInput returns a strict request with matching target, child, and
// satisfying evidence for both catalog scenarios.
func completeTestInput(t *testing.T) ReductionInput {
	t.Helper()
	catalog, gaps := testPolicy()
	root := filepath.Join(t.TempDir(), "evidence")
	finished := time.Now()
	exit := 0
	return ReductionInput{
		Request: ValidationRequest{
			SchemaVersion: SchemaVersion, RunID: "run-1", Mode: ModeRelease,
			RequestedMinor: "8.10", Profiles: []string{"dev810"}, RequiredTenantCoverage: TenantCoverageNone,
			CatalogRevision: catalog.Revision, StartedAt: time.Now(), EvidenceRoot: root,
		},
		Catalog:    catalog,
		GapCatalog: gaps,
		Targets: []ValidationTarget{{
			SchemaVersion: SchemaVersion, Profile: "dev810", ExpectedMinor: "8.10", ObservedVersion: "8.10.3", Ready: true,
		}},
		Slices: []SliceExecution{{
			SchemaVersion: SchemaVersion, RunID: "run-1", SliceID: "family", Profile: "dev810", StartedAt: time.Now(), FinishedAt: &finished,
			EvidenceDirectory: filepath.Join(root, "family", "dev810"), State: SliceFinished, ExitCode: &exit,
		}},
		Evidence: []ScenarioEvidence{
			{SchemaVersion: SchemaVersion, RunID: "run-1", SliceID: "family", ScenarioID: "required", Profile: "dev810", ObservedVersion: "8.10.3", Outcome: OutcomePassed, EvidenceLevel: EvidenceLivePostState, ObservationPath: filepath.Join(root, "family", "dev810", "required.json")},
			{SchemaVersion: SchemaVersion, RunID: "run-1", SliceID: "family", ScenarioID: "optional", Profile: "dev810", ObservedVersion: "8.10.3", Outcome: OutcomePassed, EvidenceLevel: EvidenceControlled, ObservationPath: filepath.Join(root, "family", "dev810", "optional.json")},
		},
	}
}
