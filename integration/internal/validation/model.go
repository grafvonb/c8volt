// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

// Package validation defines the version-aware release validation policy used
// by integration tooling. It intentionally has no dependency on live tests.
package validation

import "time"

// SchemaVersion is the machine-record schema emitted by this policy.
const SchemaVersion = 1

// Mode controls whether the complete release catalog or only supplied
// exploratory evidence is evaluated.
type Mode string

const (
	// ModeExploratory evaluates only explicitly selected evidence.
	ModeExploratory Mode = "exploratory"
	// ModeRelease requires the complete applicable release catalog.
	ModeRelease Mode = "release"
)

// TenantCoverage is the requested tenant proof boundary.
type TenantCoverage string

const (
	// TenantCoverageNone does not require tenant scenarios.
	TenantCoverageNone TenantCoverage = "none"
	// TenantCoverageLogical requires logical tenant scenarios.
	TenantCoverageLogical TenantCoverage = "logical"
	// TenantCoveragePhysical requires physical and logical tenant scenarios.
	TenantCoveragePhysical TenantCoverage = "physical"
)

// SliceState describes the durable lifecycle of one child invocation.
type SliceState string

const (
	// SlicePlanned is a child recorded before it starts.
	SlicePlanned SliceState = "planned"
	// SliceRunning is a child that has started but not finalized.
	SliceRunning SliceState = "running"
	// SliceFinished is a successfully finalized child.
	SliceFinished SliceState = "finished"
	// SliceFailed is a child finalized with failure.
	SliceFailed SliceState = "failed"
	// SliceInterrupted is a child stopped by interruption.
	SliceInterrupted SliceState = "interrupted"
)

// Outcome describes what happened in one scenario.
type Outcome string

const (
	// OutcomePassed indicates the scenario satisfied its assertion.
	OutcomePassed Outcome = "passed"
	// OutcomeSkipped indicates the scenario did not execute.
	OutcomeSkipped Outcome = "skipped"
	// OutcomeBlocked indicates a prerequisite prevented execution.
	OutcomeBlocked Outcome = "blocked"
	// OutcomeFailed indicates the scenario executed unsuccessfully.
	OutcomeFailed Outcome = "failed"
	// OutcomeUnsupported indicates the selected version lacks the behavior.
	OutcomeUnsupported Outcome = "unsupported"
)

// EvidenceLevel identifies the strongest proof produced by a scenario.
type EvidenceLevel string

const (
	// EvidenceLivePostState proves an independently observed resulting state.
	EvidenceLivePostState EvidenceLevel = "live-post-state"
	// EvidenceSubmission proves only accepted mutation submission.
	EvidenceSubmission EvidenceLevel = "submission-only"
	// EvidenceDryRun proves a preview without mutation.
	EvidenceDryRun EvidenceLevel = "dry-run"
	// EvidenceControlled proves a deterministic compatibility case.
	EvidenceControlled EvidenceLevel = "controlled"
	// EvidenceReadOnly proves state through nonmutating observation.
	EvidenceReadOnly EvidenceLevel = "read-only"
)

// FailureClass separates product failures from harness and environment faults.
type FailureClass string

const (
	// FailureProduct identifies observed product behavior failure.
	FailureProduct FailureClass = "product"
	// FailureHarness identifies validation tooling failure.
	FailureHarness FailureClass = "harness"
	// FailureFixture identifies invalid or missing test state.
	FailureFixture FailureClass = "fixture"
	// FailureEnvironment identifies unavailable target infrastructure.
	FailureEnvironment FailureClass = "environment"
	// FailureAuthorization identifies insufficient target permissions.
	FailureAuthorization FailureClass = "authorization"
	// FailureVersion identifies an unverifiable or mismatched version.
	FailureVersion FailureClass = "version"
	// FailureIncompleteEvidence identifies missing or invalid proof.
	FailureIncompleteEvidence FailureClass = "incomplete-evidence"
)

// Verdict is the aggregate release outcome.
type Verdict string

const (
	// VerdictRunning indicates evaluation is not finalized.
	VerdictRunning Verdict = "running"
	// VerdictPassed indicates complete applicable release proof.
	VerdictPassed Verdict = "passed"
	// VerdictPassedWithGaps indicates required proof passed with optional gaps.
	VerdictPassedWithGaps Verdict = "passed_with_gaps"
	// VerdictFailed indicates a failed check or invalid record.
	VerdictFailed Verdict = "failed"
	// VerdictIncomplete indicates interrupted or unfinished execution.
	VerdictIncomplete Verdict = "incomplete"
)

// ValidationRequest freezes the operator's intended validation scope.
type ValidationRequest struct {
	SchemaVersion          int            `json:"schemaVersion"`
	RunID                  string         `json:"runId"`
	Mode                   Mode           `json:"mode"`
	RequestedMinor         string         `json:"requestedMinor,omitempty"`
	Profiles               []string       `json:"profiles,omitempty"`
	RequiredTenantCoverage TenantCoverage `json:"requiredTenantCoverage"`
	CatalogRevision        string         `json:"catalogRevision"`
	StartedAt              time.Time      `json:"startedAt"`
	EvidenceRoot           string         `json:"evidenceRoot"`
}

// ValidationTarget records the non-secret identity and readiness of a selected
// disposable environment.
type ValidationTarget struct {
	SchemaVersion          int      `json:"schemaVersion"`
	Profile                string   `json:"profile"`
	ExpectedMinor          string   `json:"expectedMinor"`
	ObservedVersion        string   `json:"observedVersion,omitempty"`
	Ready                  bool     `json:"ready"`
	Reason                 string   `json:"reason,omitempty"`
	ClusterID              string   `json:"clusterId,omitempty"`
	LogicalTenantID        string   `json:"logicalTenantId,omitempty"`
	PhysicalTenantID       string   `json:"physicalTenantId,omitempty"`
	Roles                  []string `json:"roles,omitempty"`
	SanitizedRouteIdentity string   `json:"sanitizedRouteIdentity,omitempty"`
}

// CoverageFamily defines the minimum scenario count for a versioned family.
type CoverageFamily struct {
	ID               string   `json:"id"`
	Slice            string   `json:"slice"`
	ApplicableMinors []string `json:"applicableMinors"`
	MinimumScenarios int      `json:"minimumScenarios"`
}

// CoverageRequirement declares one stable scenario and its minimum proof.
type CoverageRequirement struct {
	ScenarioID          string        `json:"scenarioId"`
	Slice               string        `json:"slice"`
	Family              string        `json:"family"`
	ApplicableMinors    []string      `json:"applicableMinors"`
	Required            bool          `json:"required"`
	MinimumEvidence     EvidenceLevel `json:"minimumEvidence"`
	ExpectedTargetRoles []string      `json:"expectedTargetRoles,omitempty"`
	AcceptedGapID       string        `json:"acceptedGapId,omitempty"`
}

// CoverageCatalog is the complete pre-execution release policy.
type CoverageCatalog struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Revision      string                `json:"revision"`
	Families      []CoverageFamily      `json:"families"`
	Requirements  []CoverageRequirement `json:"requirements"`
}

// PrerequisiteGap describes named additional proof which is not yet required.
type PrerequisiteGap struct {
	ID                  string   `json:"id"`
	ScenarioIDs         []string `json:"scenarioIds"`
	ApplicableMinors    []string `json:"applicableMinors"`
	MissingPrerequisite string   `json:"missingPrerequisite"`
	BlockedProof        string   `json:"blockedProof"`
	Rationale           string   `json:"rationale"`
	Owner               string   `json:"owner,omitempty"`
	ReviewStatus        string   `json:"reviewStatus"`
}

// GapCatalog contains the predeclared prerequisite gaps accepted by policy.
type GapCatalog struct {
	SchemaVersion int               `json:"schemaVersion"`
	Revision      string            `json:"revision"`
	Gaps          []PrerequisiteGap `json:"gaps"`
}

// SliceExecution records one isolated child invocation.
type SliceExecution struct {
	SchemaVersion     int        `json:"schemaVersion"`
	RunID             string     `json:"runId"`
	SliceID           string     `json:"sliceId"`
	Profile           string     `json:"profile,omitempty"`
	StartedAt         time.Time  `json:"startedAt"`
	FinishedAt        *time.Time `json:"finishedAt,omitempty"`
	EvidenceDirectory string     `json:"evidenceDirectory"`
	State             SliceState `json:"state"`
	ExitCode          *int       `json:"exitCode,omitempty"`
	TerminationReason string     `json:"terminationReason,omitempty"`
	SelectedTests     []string   `json:"selectedTests,omitempty"`
}

// ScenarioEvidence is normalized proof from one scenario in one child slice.
type ScenarioEvidence struct {
	SchemaVersion   int           `json:"schemaVersion"`
	RunID           string        `json:"runId"`
	SliceID         string        `json:"sliceId"`
	ScenarioID      string        `json:"scenarioId"`
	Profile         string        `json:"profile,omitempty"`
	RoleGroup       string        `json:"roleGroup,omitempty"`
	ObservedVersion string        `json:"observedVersion,omitempty"`
	Outcome         Outcome       `json:"outcome"`
	EvidenceLevel   EvidenceLevel `json:"evidenceLevel"`
	ObservationPath string        `json:"observationPath,omitempty"`
	ResourceIDs     []string      `json:"resourceIds,omitempty"`
	GapID           string        `json:"gapId,omitempty"`
	FailureClass    FailureClass  `json:"failureClass,omitempty"`
	Reason          string        `json:"reason,omitempty"`
	CleanupOutcome  string        `json:"cleanupOutcome,omitempty"`
}

// OutcomeCounters summarizes expected requirements joined to current evidence.
type OutcomeCounters struct {
	Expected    int `json:"expected"`
	Passed      int `json:"passed"`
	Skipped     int `json:"skipped"`
	Blocked     int `json:"blocked"`
	Failed      int `json:"failed"`
	Unsupported int `json:"unsupported"`
	Missing     int `json:"missing"`
}

// Finding gives a stable, redacted explanation for a gap, failure, or missing
// requirement.
type Finding struct {
	ScenarioID   string       `json:"scenarioId,omitempty"`
	Family       string       `json:"family,omitempty"`
	Profile      string       `json:"profile,omitempty"`
	GapID        string       `json:"gapId,omitempty"`
	FailureClass FailureClass `json:"failureClass,omitempty"`
	Reason       string       `json:"reason"`
}

// ReleaseSummary is the deterministic aggregate result.
type ReleaseSummary struct {
	SchemaVersion       int                        `json:"schemaVersion"`
	RunID               string                     `json:"runId"`
	Mode                Mode                       `json:"mode"`
	RequestedMinor      string                     `json:"requestedMinor,omitempty"`
	ActualVersions      []string                   `json:"actualVersions,omitempty"`
	FamilyCounters      map[string]OutcomeCounters `json:"familyCounters"`
	MissingRequirements []Finding                  `json:"missingRequirements,omitempty"`
	Gaps                []Finding                  `json:"gaps,omitempty"`
	Failures            []Finding                  `json:"failures,omitempty"`
	Verdict             Verdict                    `json:"verdict"`
	CoverageComplete    bool                       `json:"coverageComplete"`
}
