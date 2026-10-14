// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var (
	secretAssignmentPattern = regexp.MustCompile(`(?i)(authorization|token|access[_-]?token|refresh[_-]?token|client[_-]?secret|password|secret)(\s*[:=]\s*)([^\s,;]+)`)
	bearerPattern           = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/-]+=*`)
)

// ReductionInput contains a frozen policy plus current-invocation execution
// records. No filesystem discovery participates in completeness decisions.
type ReductionInput struct {
	Request     ValidationRequest
	Catalog     CoverageCatalog
	GapCatalog  GapCatalog
	Targets     []ValidationTarget
	Slices      []SliceExecution
	Evidence    []ScenarioEvidence
	Interrupted bool
}

// Reduce evaluates current-run evidence against the frozen request and policy.
func Reduce(input ReductionInput) (ReleaseSummary, error) {
	if err := ValidatePolicy(input.Catalog, input.GapCatalog); err != nil {
		return ReleaseSummary{}, err
	}
	if err := ValidateRequest(input.Request, input.Catalog); err != nil {
		return ReleaseSummary{}, err
	}
	summary := ReleaseSummary{
		SchemaVersion:  SchemaVersion,
		RunID:          input.Request.RunID,
		Mode:           input.Request.Mode,
		RequestedMinor: input.Request.RequestedMinor,
		FamilyCounters: map[string]OutcomeCounters{},
		Verdict:        VerdictRunning,
	}

	requirements := make(map[string]CoverageRequirement, len(input.Catalog.Requirements))
	for _, requirement := range input.Catalog.Requirements {
		requirements[requirement.ScenarioID] = requirement
	}
	gaps := make(map[string]PrerequisiteGap, len(input.GapCatalog.Gaps))
	for _, gap := range input.GapCatalog.Gaps {
		gaps[gap.ID] = gap
	}

	targetsByProfile := map[string]ValidationTarget{}
	for _, target := range input.Targets {
		if target.SchemaVersion != SchemaVersion {
			appendFailure(&summary, Finding{Profile: target.Profile, FailureClass: FailureHarness, Reason: fmt.Sprintf("target schemaVersion %d is unsupported", target.SchemaVersion)})
			continue
		}
		if target.Profile == "" {
			appendFailure(&summary, Finding{FailureClass: FailureHarness, Reason: "validation target has an empty profile"})
			continue
		}
		if _, duplicate := targetsByProfile[target.Profile]; duplicate {
			appendFailure(&summary, Finding{Profile: target.Profile, FailureClass: FailureHarness, Reason: "duplicate validation target"})
			continue
		}
		target.Reason = RedactSecrets(target.Reason)
		targetsByProfile[target.Profile] = target
		if target.ObservedVersion != "" {
			summary.ActualVersions = append(summary.ActualVersions, target.ObservedVersion)
		}
	}
	summary.ActualVersions = UniqueSorted(summary.ActualVersions)

	selectedTargets := validateSelectedTargets(&summary, input.Request, targetsByProfile)
	slices := validateSlices(&summary, input.Request, input.Slices)
	evidence := validateEvidenceRecords(&summary, input.Request, requirements, gaps, slices, input.Evidence)

	if input.Request.Mode == ModeRelease {
		reduceStrictRequirements(&summary, input.Request, input.Catalog.Requirements, gaps, selectedTargets, evidence)
	} else {
		reduceExploratoryEvidence(&summary, requirements, gaps, evidence)
	}

	incomplete := input.Interrupted
	for _, slice := range input.Slices {
		if slice.State == SlicePlanned || slice.State == SliceRunning || slice.State == SliceInterrupted {
			incomplete = true
		}
	}
	switch {
	case len(summary.Failures) > 0:
		summary.Verdict = VerdictFailed
	case incomplete:
		summary.Verdict = VerdictIncomplete
	case len(summary.MissingRequirements) > 0:
		summary.Verdict = VerdictFailed
	case len(summary.Gaps) > 0:
		summary.Verdict = VerdictPassedWithGaps
	default:
		summary.Verdict = VerdictPassed
	}
	summary.CoverageComplete = summary.Verdict == VerdictPassed
	sortSummary(&summary)
	return summary, nil
}

// validateSelectedTargets establishes strict requested-version readiness before
// evidence is allowed to satisfy release requirements.
func validateSelectedTargets(summary *ReleaseSummary, request ValidationRequest, targets map[string]ValidationTarget) []ValidationTarget {
	var selected []ValidationTarget
	for _, profile := range request.Profiles {
		target, exists := targets[profile]
		if !exists {
			appendFailure(summary, Finding{Profile: profile, FailureClass: FailureVersion, Reason: "selected profile has no validation target"})
			continue
		}
		if target.SanitizedRouteIdentity != "" {
			parsed, err := url.Parse(target.SanitizedRouteIdentity)
			if err != nil || parsed.User != nil || RedactSecrets(target.SanitizedRouteIdentity) != target.SanitizedRouteIdentity {
				appendFailure(summary, Finding{Profile: profile, FailureClass: FailureHarness, Reason: "target route identity contains invalid or sensitive data"})
				continue
			}
		}
		expected, expectedErr := CanonicalMinor(target.ExpectedMinor)
		observed, observedErr := ObservedMinor(target.ObservedVersion)
		if expectedErr != nil || observedErr != nil {
			reason := "target version is not verifiable"
			if target.Reason != "" {
				reason += ": " + target.Reason
			}
			appendFailure(summary, Finding{Profile: profile, FailureClass: FailureVersion, Reason: reason})
			continue
		}
		if expected != observed {
			appendFailure(summary, Finding{Profile: profile, FailureClass: FailureVersion, Reason: fmt.Sprintf("expected minor %s but observed %s", expected, observed)})
			continue
		}
		if request.Mode == ModeRelease && expected != request.RequestedMinor {
			appendFailure(summary, Finding{Profile: profile, FailureClass: FailureVersion, Reason: fmt.Sprintf("target minor %s cannot satisfy requested minor %s", expected, request.RequestedMinor)})
			continue
		}
		if !target.Ready {
			reason := "target is not ready"
			if target.Reason != "" {
				reason += ": " + target.Reason
			}
			appendFailure(summary, Finding{Profile: profile, FailureClass: FailureEnvironment, Reason: reason})
			continue
		}
		selected = append(selected, target)
	}
	return selected
}

// validateSlices rejects stale, unsafe, duplicated, or failed child records.
func validateSlices(summary *ReleaseSummary, request ValidationRequest, records []SliceExecution) map[string]SliceExecution {
	result := map[string]SliceExecution{}
	for _, record := range records {
		key := record.SliceID + "\x00" + record.Profile
		if record.SchemaVersion != SchemaVersion {
			appendFailure(summary, Finding{Profile: record.Profile, FailureClass: FailureHarness, Reason: fmt.Sprintf("slice %q schemaVersion %d is unsupported", record.SliceID, record.SchemaVersion)})
			continue
		}
		if record.RunID != request.RunID {
			appendFailure(summary, Finding{Profile: record.Profile, FailureClass: FailureIncompleteEvidence, Reason: fmt.Sprintf("slice %q belongs to foreign run %q", record.SliceID, record.RunID)})
			continue
		}
		if record.SliceID == "" {
			appendFailure(summary, Finding{Profile: record.Profile, FailureClass: FailureHarness, Reason: "slice identity is empty"})
			continue
		}
		if !contains(request.Profiles, record.Profile) {
			appendFailure(summary, Finding{Profile: record.Profile, FailureClass: FailureIncompleteEvidence, Reason: fmt.Sprintf("slice %q belongs to an unselected profile", record.SliceID)})
			continue
		}
		if _, duplicate := result[key]; duplicate {
			appendFailure(summary, Finding{Profile: record.Profile, FailureClass: FailureIncompleteEvidence, Reason: fmt.Sprintf("duplicate completion record for slice %q", record.SliceID)})
			continue
		}
		if err := ValidateChildPath(request.EvidenceRoot, record.EvidenceDirectory); err != nil {
			appendFailure(summary, Finding{Profile: record.Profile, FailureClass: FailureHarness, Reason: fmt.Sprintf("slice %q: %v", record.SliceID, err)})
			continue
		}
		result[key] = record
		if record.State == SliceFailed || (record.ExitCode != nil && *record.ExitCode != 0) {
			appendFailure(summary, Finding{Profile: record.Profile, FailureClass: FailureHarness, Reason: fmt.Sprintf("slice %q failed: %s", record.SliceID, RedactSecrets(record.TerminationReason))})
		}
	}
	return result
}

// validateEvidenceRecords validates provenance and identity before reduction.
func validateEvidenceRecords(summary *ReleaseSummary, request ValidationRequest, requirements map[string]CoverageRequirement, gaps map[string]PrerequisiteGap, slices map[string]SliceExecution, records []ScenarioEvidence) map[string]ScenarioEvidence {
	result := map[string]ScenarioEvidence{}
	for _, record := range records {
		key := evidenceKey(record.ScenarioID, record.Profile)
		if record.SchemaVersion != SchemaVersion {
			appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Profile: record.Profile, FailureClass: FailureHarness, Reason: fmt.Sprintf("evidence schemaVersion %d is unsupported", record.SchemaVersion)})
			continue
		}
		if record.RunID != request.RunID {
			appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Profile: record.Profile, FailureClass: FailureIncompleteEvidence, Reason: fmt.Sprintf("evidence belongs to foreign run %q", record.RunID)})
			continue
		}
		if !contains(request.Profiles, record.Profile) {
			appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Profile: record.Profile, FailureClass: FailureIncompleteEvidence, Reason: "evidence belongs to an unselected profile"})
			continue
		}
		requirement, exists := requirements[record.ScenarioID]
		if !exists {
			appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Profile: record.Profile, FailureClass: FailureHarness, Reason: "unknown scenario evidence"})
			continue
		}
		if requirement.Slice != record.SliceID {
			appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Family: requirement.Family, Profile: record.Profile, FailureClass: FailureHarness, Reason: fmt.Sprintf("evidence slice %q does not match catalog slice %q", record.SliceID, requirement.Slice)})
			continue
		}
		if _, duplicate := result[key]; duplicate {
			appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Family: requirement.Family, Profile: record.Profile, FailureClass: FailureIncompleteEvidence, Reason: "duplicate scenario evidence"})
			continue
		}
		if _, exists := slices[record.SliceID+"\x00"+record.Profile]; !exists {
			appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Family: requirement.Family, Profile: record.Profile, FailureClass: FailureIncompleteEvidence, Reason: "scenario has no matching child slice record"})
			continue
		}
		if record.ObservationPath != "" {
			if err := ValidateChildPath(request.EvidenceRoot, record.ObservationPath); err != nil {
				appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Family: requirement.Family, Profile: record.Profile, FailureClass: FailureHarness, Reason: err.Error()})
				continue
			}
		}
		if !validOutcome(record.Outcome) || !validEvidenceLevel(record.EvidenceLevel) {
			appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Family: requirement.Family, Profile: record.Profile, FailureClass: FailureHarness, Reason: "scenario has invalid outcome or evidence level"})
			continue
		}
		if record.GapID != "" {
			gap, exists := gaps[record.GapID]
			if !exists || !contains(gap.ScenarioIDs, record.ScenarioID) {
				appendFailure(summary, Finding{ScenarioID: record.ScenarioID, Family: requirement.Family, Profile: record.Profile, GapID: record.GapID, FailureClass: FailureHarness, Reason: "unrecognized scenario gap"})
				continue
			}
		}
		record.Reason = RedactSecrets(record.Reason)
		record.CleanupOutcome = RedactSecrets(record.CleanupOutcome)
		result[key] = record
	}
	return result
}

// reduceStrictRequirements joins every applicable requirement to evidence for
// every ready target in the frozen release scope.
func reduceStrictRequirements(summary *ReleaseSummary, request ValidationRequest, requirements []CoverageRequirement, gaps map[string]PrerequisiteGap, targets []ValidationTarget, evidence map[string]ScenarioEvidence) {
	for _, requirement := range requirements {
		if !Applicable(request.RequestedMinor, requirement.ApplicableMinors) {
			continue
		}
		required := requirementRequired(request, requirement)
		for _, target := range targets {
			if !targetHasRoles(target, requirement.ExpectedTargetRoles) {
				if required {
					addMissing(summary, requirement, target.Profile, "target lacks required role proof")
				} else {
					addAcceptedGap(summary, requirement, target.Profile, gaps, "target lacks optional role proof", true)
				}
				continue
			}
			record, exists := evidence[evidenceKey(requirement.ScenarioID, target.Profile)]
			if !exists {
				if required {
					addMissing(summary, requirement, target.Profile, "required current-run evidence is missing")
				} else {
					addAcceptedGap(summary, requirement, target.Profile, gaps, "optional proof is not available", true)
				}
				continue
			}
			observedMinor, err := ObservedMinor(record.ObservedVersion)
			if err != nil || observedMinor != request.RequestedMinor {
				counters := summary.FamilyCounters[requirement.Family]
				counters.Expected++
				counters.Failed++
				summary.FamilyCounters[requirement.Family] = counters
				appendFailure(summary, Finding{ScenarioID: requirement.ScenarioID, Family: requirement.Family, Profile: target.Profile, FailureClass: FailureVersion, Reason: fmt.Sprintf("scenario evidence version %q does not match requested minor %s", record.ObservedVersion, request.RequestedMinor)})
				continue
			}
			countEvidence(summary, requirement, target.Profile, record, gaps)
		}
	}
}

// requirementRequired promotes logical tenant proof from an optional named
// gap to mandatory coverage when the request explicitly requires tenants.
func requirementRequired(request ValidationRequest, requirement CoverageRequirement) bool {
	if requirement.Required {
		return true
	}
	if contains(requirement.ExpectedTargetRoles, TenantRoleLogical) {
		return request.RequiredTenantCoverage == TenantCoverageLogical || request.RequiredTenantCoverage == TenantCoveragePhysical
	}
	return false
}

// reduceExploratoryEvidence reports supplied results without synthesizing the
// complete release matrix.
func reduceExploratoryEvidence(summary *ReleaseSummary, requirements map[string]CoverageRequirement, gaps map[string]PrerequisiteGap, evidence map[string]ScenarioEvidence) {
	for _, record := range evidence {
		requirement := requirements[record.ScenarioID]
		countEvidence(summary, requirement, record.Profile, record, gaps)
	}
}

// countEvidence updates counters and findings for one joined record.
func countEvidence(summary *ReleaseSummary, requirement CoverageRequirement, profile string, record ScenarioEvidence, gaps map[string]PrerequisiteGap) {
	counters := summary.FamilyCounters[requirement.Family]
	counters.Expected++
	switch record.Outcome {
	case OutcomePassed:
		if !EvidenceMeetsMinimum(record.EvidenceLevel, requirement.MinimumEvidence) {
			counters.Failed++
			appendFailure(summary, Finding{ScenarioID: requirement.ScenarioID, Family: requirement.Family, Profile: profile, FailureClass: FailureIncompleteEvidence, Reason: fmt.Sprintf("evidence level %q does not satisfy minimum %q", record.EvidenceLevel, requirement.MinimumEvidence)})
		} else {
			counters.Passed++
		}
	case OutcomeSkipped:
		counters.Skipped++
		classifyNonPass(summary, requirement, profile, record, gaps)
	case OutcomeBlocked:
		counters.Blocked++
		classifyNonPass(summary, requirement, profile, record, gaps)
	case OutcomeFailed:
		counters.Failed++
		failureClass := record.FailureClass
		if failureClass == "" {
			failureClass = FailureIncompleteEvidence
		}
		appendFailure(summary, Finding{ScenarioID: requirement.ScenarioID, Family: requirement.Family, Profile: profile, FailureClass: failureClass, Reason: nonempty(record.Reason, "scenario failed")})
	case OutcomeUnsupported:
		counters.Unsupported++
		classifyNonPass(summary, requirement, profile, record, gaps)
	}
	summary.FamilyCounters[requirement.Family] = counters
}

// classifyNonPass permits only predeclared optional gaps; required evidence can
// never be converted to success by a runtime gap identifier.
func classifyNonPass(summary *ReleaseSummary, requirement CoverageRequirement, profile string, record ScenarioEvidence, gaps map[string]PrerequisiteGap) {
	if !requirement.Required && requirement.AcceptedGapID != "" && record.GapID == requirement.AcceptedGapID {
		addAcceptedGap(summary, requirement, profile, gaps, nonempty(record.Reason, "optional proof is blocked"), false)
		return
	}
	appendFailure(summary, Finding{ScenarioID: requirement.ScenarioID, Family: requirement.Family, Profile: profile, GapID: record.GapID, FailureClass: FailureIncompleteEvidence, Reason: nonempty(record.Reason, fmt.Sprintf("required scenario outcome is %s", record.Outcome))})
}

// addMissing records an expected catalog row with no satisfying evidence.
func addMissing(summary *ReleaseSummary, requirement CoverageRequirement, profile string, reason string) {
	counters := summary.FamilyCounters[requirement.Family]
	counters.Expected++
	counters.Missing++
	summary.FamilyCounters[requirement.Family] = counters
	summary.MissingRequirements = append(summary.MissingRequirements, Finding{ScenarioID: requirement.ScenarioID, Family: requirement.Family, Profile: profile, FailureClass: FailureIncompleteEvidence, Reason: reason})
}

// addAcceptedGap records a known optional limitation without certifying it.
func addAcceptedGap(summary *ReleaseSummary, requirement CoverageRequirement, profile string, gaps map[string]PrerequisiteGap, reason string, countMissingEvidence bool) {
	gap, exists := gaps[requirement.AcceptedGapID]
	if !exists {
		appendFailure(summary, Finding{ScenarioID: requirement.ScenarioID, Family: requirement.Family, Profile: profile, GapID: requirement.AcceptedGapID, FailureClass: FailureHarness, Reason: "catalog references an unknown gap"})
		return
	}
	if countMissingEvidence {
		counters := summary.FamilyCounters[requirement.Family]
		counters.Expected++
		counters.Blocked++
		summary.FamilyCounters[requirement.Family] = counters
	}
	summary.Gaps = append(summary.Gaps, Finding{ScenarioID: requirement.ScenarioID, Family: requirement.Family, Profile: profile, GapID: gap.ID, Reason: RedactSecrets(reason + ": " + gap.BlockedProof)})
}

// appendFailure centralizes redaction for summary failure text.
func appendFailure(summary *ReleaseSummary, finding Finding) {
	finding.Reason = RedactSecrets(finding.Reason)
	summary.Failures = append(summary.Failures, finding)
}

// RedactSecrets removes common credential assignments, bearer tokens, and URL
// user information from persisted summary text.
func RedactSecrets(value string) string {
	value = bearerPattern.ReplaceAllString(value, "Bearer [REDACTED]")
	value = secretAssignmentPattern.ReplaceAllString(value, `$1$2[REDACTED]`)
	for _, field := range strings.Fields(value) {
		candidate := strings.Trim(field, `"'(),;`)
		parsed, err := url.Parse(candidate)
		if err == nil && parsed.IsAbs() && parsed.User != nil {
			redacted := *parsed
			redacted.User = url.User("[REDACTED]")
			value = strings.ReplaceAll(value, candidate, redacted.String())
		}
	}
	return value
}

// validOutcome recognizes every schema-supported outcome value.
func validOutcome(outcome Outcome) bool {
	switch outcome {
	case OutcomePassed, OutcomeSkipped, OutcomeBlocked, OutcomeFailed, OutcomeUnsupported:
		return true
	default:
		return false
	}
}

// targetHasRoles requires every catalog role when a scenario is role-specific.
func targetHasRoles(target ValidationTarget, wanted []string) bool {
	for _, role := range wanted {
		if !contains(target.Roles, role) {
			return false
		}
	}
	return true
}

// evidenceKey joins stable scenario and target identities.
func evidenceKey(scenarioID string, profile string) string {
	return scenarioID + "\x00" + profile
}

// contains reports exact membership in a small policy slice.
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// nonempty supplies stable fallback wording for incomplete evidence.
func nonempty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// sortSummary makes machine output reproducible regardless of input ordering.
func sortSummary(summary *ReleaseSummary) {
	less := func(findings []Finding) {
		sort.Slice(findings, func(i, j int) bool {
			left := findings[i].Family + "\x00" + findings[i].ScenarioID + "\x00" + findings[i].Profile + "\x00" + findings[i].GapID + "\x00" + findings[i].Reason
			right := findings[j].Family + "\x00" + findings[j].ScenarioID + "\x00" + findings[j].Profile + "\x00" + findings[j].GapID + "\x00" + findings[j].Reason
			return left < right
		})
	}
	less(summary.MissingRequirements)
	less(summary.Gaps)
	less(summary.Failures)
}
