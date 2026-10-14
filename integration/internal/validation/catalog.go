// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	validationassets "github.com/grafvonb/c8volt/integration/assets"
)

// LoadDefaultPolicy loads embedded checked-in catalogs independently of the
// caller's current working directory.
func LoadDefaultPolicy() (CoverageCatalog, GapCatalog, error) {
	var catalog CoverageCatalog
	if err := decodeStrictJSON(validationassets.ReleaseCoverageJSON, &catalog); err != nil {
		return CoverageCatalog{}, GapCatalog{}, fmt.Errorf("decode release coverage: %w", err)
	}
	var gaps GapCatalog
	if err := decodeStrictJSON(validationassets.ReleaseGapsJSON, &gaps); err != nil {
		return CoverageCatalog{}, GapCatalog{}, fmt.Errorf("decode release gaps: %w", err)
	}
	if err := ValidatePolicy(catalog, gaps); err != nil {
		return CoverageCatalog{}, GapCatalog{}, err
	}
	return catalog, gaps, nil
}

// ValidatePolicy checks catalog identity, applicability, family minima, and gap
// references before the policy can be frozen into a request.
func ValidatePolicy(catalog CoverageCatalog, gaps GapCatalog) error {
	var problems []error
	if catalog.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Errorf("coverage schemaVersion %d is unsupported", catalog.SchemaVersion))
	}
	if gaps.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Errorf("gap schemaVersion %d is unsupported", gaps.SchemaVersion))
	}
	if strings.TrimSpace(catalog.Revision) == "" || catalog.Revision != gaps.Revision {
		problems = append(problems, fmt.Errorf("coverage revision %q and gap revision %q must be equal and nonempty", catalog.Revision, gaps.Revision))
	}
	families := map[string]CoverageFamily{}
	for _, family := range catalog.Families {
		if family.ID == "" || family.Slice == "" || family.MinimumScenarios < 1 || len(family.ApplicableMinors) == 0 {
			problems = append(problems, fmt.Errorf("invalid coverage family %+v", family))
			continue
		}
		if _, exists := families[family.ID]; exists {
			problems = append(problems, fmt.Errorf("duplicate coverage family %q", family.ID))
		}
		families[family.ID] = family
		for _, minor := range family.ApplicableMinors {
			if _, err := CanonicalMinor(minor); err != nil {
				problems = append(problems, fmt.Errorf("family %q: %w", family.ID, err))
			}
		}
	}
	requirements := map[string]CoverageRequirement{}
	counts := map[string]map[string]int{}
	for _, requirement := range catalog.Requirements {
		if requirement.ScenarioID == "" || requirement.Slice == "" || requirement.Family == "" || len(requirement.ApplicableMinors) == 0 || !validEvidenceLevel(requirement.MinimumEvidence) {
			problems = append(problems, fmt.Errorf("invalid coverage requirement %+v", requirement))
			continue
		}
		if _, exists := requirements[requirement.ScenarioID]; exists {
			problems = append(problems, fmt.Errorf("duplicate scenario identity %q", requirement.ScenarioID))
		}
		requirements[requirement.ScenarioID] = requirement
		family, exists := families[requirement.Family]
		if !exists {
			problems = append(problems, fmt.Errorf("scenario %q references unknown family %q", requirement.ScenarioID, requirement.Family))
		} else if family.Slice != requirement.Slice {
			problems = append(problems, fmt.Errorf("scenario %q slice %q does not match family slice %q", requirement.ScenarioID, requirement.Slice, family.Slice))
		}
		if requirement.Required && requirement.AcceptedGapID != "" {
			problems = append(problems, fmt.Errorf("required scenario %q cannot be waived by gap %q", requirement.ScenarioID, requirement.AcceptedGapID))
		}
		for _, minor := range requirement.ApplicableMinors {
			if _, err := CanonicalMinor(minor); err != nil {
				problems = append(problems, fmt.Errorf("scenario %q: %w", requirement.ScenarioID, err))
			}
			if exists && !Applicable(minor, family.ApplicableMinors) {
				problems = append(problems, fmt.Errorf("scenario %q applies to %s outside family %q applicability", requirement.ScenarioID, minor, requirement.Family))
			}
			if requirement.Required {
				if counts[requirement.Family] == nil {
					counts[requirement.Family] = map[string]int{}
				}
				counts[requirement.Family][minor]++
			}
		}
	}
	for _, family := range catalog.Families {
		for _, minor := range family.ApplicableMinors {
			if counts[family.ID][minor] < family.MinimumScenarios {
				problems = append(problems, fmt.Errorf("family %q has %d required scenarios for %s, minimum is %d", family.ID, counts[family.ID][minor], minor, family.MinimumScenarios))
			}
		}
	}
	gapByID := map[string]PrerequisiteGap{}
	for _, gap := range gaps.Gaps {
		if gap.ID == "" || len(gap.ScenarioIDs) == 0 || len(gap.ApplicableMinors) == 0 || gap.MissingPrerequisite == "" || gap.BlockedProof == "" || gap.ReviewStatus == "" {
			problems = append(problems, fmt.Errorf("invalid prerequisite gap %+v", gap))
			continue
		}
		if _, exists := gapByID[gap.ID]; exists {
			problems = append(problems, fmt.Errorf("duplicate prerequisite gap %q", gap.ID))
		}
		gapByID[gap.ID] = gap
		for _, scenarioID := range gap.ScenarioIDs {
			requirement, exists := requirements[scenarioID]
			if !exists {
				problems = append(problems, fmt.Errorf("gap %q references unknown scenario %q", gap.ID, scenarioID))
				continue
			}
			if requirement.Required || requirement.AcceptedGapID != gap.ID {
				problems = append(problems, fmt.Errorf("gap %q does not match optional scenario %q", gap.ID, scenarioID))
			}
		}
		for _, minor := range gap.ApplicableMinors {
			if _, err := CanonicalMinor(minor); err != nil {
				problems = append(problems, fmt.Errorf("gap %q: %w", gap.ID, err))
			}
		}
	}
	for _, requirement := range catalog.Requirements {
		if requirement.AcceptedGapID != "" {
			if _, exists := gapByID[requirement.AcceptedGapID]; !exists {
				problems = append(problems, fmt.Errorf("scenario %q references unknown gap %q", requirement.ScenarioID, requirement.AcceptedGapID))
			}
		}
	}
	return errors.Join(problems...)
}

// Applicable reports whether a versioned record applies to the given minor.
func Applicable(minor string, applicableMinors []string) bool {
	for _, candidate := range applicableMinors {
		if candidate == minor {
			return true
		}
	}
	return false
}

// EvidenceMeetsMinimum prevents weaker or incomparable proof from satisfying a
// catalog requirement.
func EvidenceMeetsMinimum(actual EvidenceLevel, minimum EvidenceLevel) bool {
	if actual == minimum {
		return true
	}
	switch minimum {
	case EvidenceSubmission:
		return actual == EvidenceLivePostState
	case EvidenceDryRun:
		return actual == EvidenceSubmission || actual == EvidenceLivePostState
	case EvidenceReadOnly:
		return actual == EvidenceLivePostState
	default:
		return false
	}
}

// validEvidenceLevel recognizes every schema-supported evidence value.
func validEvidenceLevel(level EvidenceLevel) bool {
	switch level {
	case EvidenceLivePostState, EvidenceSubmission, EvidenceDryRun, EvidenceControlled, EvidenceReadOnly:
		return true
	default:
		return false
	}
}

// decodeStrictJSON rejects unknown fields and trailing JSON values in policy
// assets so stale schema content cannot be ignored silently.
func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
