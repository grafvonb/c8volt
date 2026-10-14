// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// CanonicalMinor accepts only the operator-facing supported minor spellings.
func CanonicalMinor(value string) (string, error) {
	switch value {
	case "8.8", "8.9", "8.10":
		return value, nil
	default:
		return "", fmt.Errorf("unsupported non-canonical Camunda minor %q", value)
	}
}

// ObservedMinor extracts a supported major/minor from a full observed server
// version without treating arbitrary substring matches as version proof.
func ObservedMinor(value string) (string, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	parts := strings.Split(value, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return "", fmt.Errorf("invalid observed Camunda version %q", value)
	}
	minor := parts[0] + "." + parts[1]
	if _, err := CanonicalMinor(minor); err != nil {
		return "", fmt.Errorf("invalid observed Camunda version %q: %w", value, err)
	}
	for _, part := range parts[2:] {
		if part == "" || strings.IndexFunc(part, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			return "", fmt.Errorf("invalid observed Camunda version %q", value)
		}
	}
	return minor, nil
}

// ValidateRequest rejects invalid strict scope before any child process can
// mutate a target.
func ValidateRequest(request ValidationRequest, catalog CoverageCatalog) error {
	var problems []error
	if request.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Errorf("request schemaVersion %d is unsupported", request.SchemaVersion))
	}
	if !runIDPattern.MatchString(request.RunID) {
		problems = append(problems, fmt.Errorf("runId %q is invalid", request.RunID))
	}
	if request.CatalogRevision != catalog.Revision {
		problems = append(problems, fmt.Errorf("catalog revision %q does not match %q", request.CatalogRevision, catalog.Revision))
	}
	if request.StartedAt.IsZero() {
		problems = append(problems, errors.New("startedAt is required"))
	}
	if err := ValidateEvidenceRoot(request.EvidenceRoot); err != nil {
		problems = append(problems, err)
	}
	switch request.Mode {
	case ModeRelease:
		minor, err := CanonicalMinor(request.RequestedMinor)
		if err != nil {
			problems = append(problems, err)
		} else if minor == "8.8" {
			problems = append(problems, errors.New("release mode requires Camunda 8.9 or 8.10; 8.8 supports baseline and volume exploration only"))
		}
		if len(request.Profiles) == 0 {
			problems = append(problems, errors.New("release mode requires at least one selected profile"))
		}
	case ModeExploratory:
		if request.RequestedMinor != "" {
			if _, err := CanonicalMinor(request.RequestedMinor); err != nil {
				problems = append(problems, err)
			}
		}
	default:
		problems = append(problems, fmt.Errorf("unsupported validation mode %q", request.Mode))
	}
	switch request.RequiredTenantCoverage {
	case TenantCoverageNone, TenantCoverageLogical, TenantCoveragePhysical:
	default:
		problems = append(problems, fmt.Errorf("unsupported required tenant coverage %q", request.RequiredTenantCoverage))
	}
	seen := map[string]struct{}{}
	for _, profile := range request.Profiles {
		if strings.TrimSpace(profile) != profile || profile == "" || strings.ContainsAny(profile, `/\\`) {
			problems = append(problems, fmt.Errorf("profile name %q is invalid", profile))
			continue
		}
		if _, exists := seen[profile]; exists {
			problems = append(problems, fmt.Errorf("profile %q is selected more than once", profile))
		}
		seen[profile] = struct{}{}
	}
	return errors.Join(problems...)
}

// ValidateEvidenceRoot requires an explicit absolute directory and rejects
// broad filesystem roots as evidence destinations.
func ValidateEvidenceRoot(root string) error {
	if root == "" || !filepath.IsAbs(root) {
		return fmt.Errorf("evidence root %q must be absolute", root)
	}
	clean := filepath.Clean(root)
	if clean != root {
		return fmt.Errorf("evidence root %q must be clean", root)
	}
	volume := filepath.VolumeName(clean)
	if clean == string(filepath.Separator) || clean == volume+string(filepath.Separator) {
		return fmt.Errorf("evidence root %q is too broad", root)
	}
	return nil
}

// ValidateChildPath ensures a child evidence path remains beneath the frozen
// invocation root, including after lexical cleanup.
func ValidateChildPath(root string, child string) error {
	if err := ValidateEvidenceRoot(root); err != nil {
		return err
	}
	if child == "" {
		return errors.New("evidence child path is empty")
	}
	if !filepath.IsAbs(child) {
		child = filepath.Join(root, child)
	}
	cleanChild := filepath.Clean(child)
	rel, err := filepath.Rel(root, cleanChild)
	if err != nil {
		return fmt.Errorf("resolve evidence path: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("evidence path %q must remain beneath %q", child, root)
	}
	return nil
}

// UniqueSorted returns a deterministic copy without blank or duplicate values.
func UniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
