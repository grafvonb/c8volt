// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

const (
	// TenantRoleLogical identifies the logical-tenant proof group in catalog and target records.
	TenantRoleLogical = "logical"
	// TenantRolePhysical identifies physical-route proof within one declared cluster.
	TenantRolePhysical = "physical"
)

// TenantRoles is credential-free metadata that maps validation roles to
// profiles already present in the operator's default local configuration.
type TenantRoles struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Logical       *LogicalTenantRoles  `json:"logical,omitempty"`
	Physical      *PhysicalTenantRoles `json:"physical,omitempty"`
}

// LogicalTenantRoles declares four principals for two logical tenants sharing
// one physical endpoint. Credentials remain exclusively in named profiles.
type LogicalTenantRoles struct {
	ClusterID          string `json:"clusterId"`
	PhysicalTenantID   string `json:"physicalTenantId"`
	TenantAID          string `json:"tenantAId"`
	TenantBID          string `json:"tenantBId"`
	TenantAProfile     string `json:"tenantAProfile"`
	TenantBProfile     string `json:"tenantBProfile"`
	BothTenantsProfile string `json:"bothTenantsProfile"`
	RestrictedAProfile string `json:"restrictedAProfile"`
}

// PhysicalTenantRoles declares two provisioned physical endpoints plus an
// explicit default-route control for one of those endpoint identities.
type PhysicalTenantRoles struct {
	ClusterID                    string `json:"clusterId"`
	PhysicalTenantAID            string `json:"physicalTenantAId"`
	PhysicalTenantBID            string `json:"physicalTenantBId"`
	LogicalTenantID              string `json:"logicalTenantId"`
	TenantAProfile               string `json:"tenantAProfile"`
	TenantBProfile               string `json:"tenantBProfile"`
	DefaultRouteProfile          string `json:"defaultRouteProfile"`
	DefaultRoutePhysicalTenantID string `json:"defaultRoutePhysicalTenantId"`
}

// TenantProfileMetadata contains only the non-secret local profile fields
// needed to reject missing, conflicting, or foreign-version role assignments.
type TenantProfileMetadata struct {
	Name            string
	ExpectedMinor   string
	LogicalTenantID string
	CamundaBaseURL  string
}

// LoadTenantRoles strictly decodes credential-free role metadata from an
// explicit path; unknown fields prevent accidental credential serialization.
func LoadTenantRoles(path string) (TenantRoles, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TenantRoles{}, fmt.Errorf("read tenant roles: %w", err)
	}
	var roles TenantRoles
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&roles); err != nil {
		return TenantRoles{}, fmt.Errorf("decode tenant roles: %w", err)
	}
	if err := requireTenantRolesEOF(decoder); err != nil {
		return TenantRoles{}, fmt.Errorf("decode tenant roles: %w", err)
	}
	return roles, nil
}

// ValidateTenantRoles verifies logical identities and their default-local
// profile bindings without reading or returning authentication configuration.
func ValidateTenantRoles(roles TenantRoles, profiles map[string]TenantProfileMetadata, requestedMinor string) error {
	var problems []error
	if roles.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Errorf("tenant role schemaVersion %d is unsupported", roles.SchemaVersion))
	}
	if roles.Logical == nil {
		problems = append(problems, errors.New("logical tenant role group is required"))
		return errors.Join(problems...)
	}
	logical := *roles.Logical
	for name, value := range map[string]string{
		"clusterId": logical.ClusterID, "physicalTenantId": logical.PhysicalTenantID,
		"tenantAId": logical.TenantAID, "tenantBId": logical.TenantBID,
		"tenantAProfile": logical.TenantAProfile, "tenantBProfile": logical.TenantBProfile,
		"bothTenantsProfile": logical.BothTenantsProfile, "restrictedAProfile": logical.RestrictedAProfile,
	} {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			problems = append(problems, fmt.Errorf("logical %s must be nonempty and trimmed", name))
		}
	}
	if logical.TenantAID == logical.TenantBID {
		problems = append(problems, errors.New("logical tenant IDs must be distinct"))
	}

	assignments := []struct {
		role       string
		profile    string
		wantTenant string
	}{
		{role: "tenantA", profile: logical.TenantAProfile, wantTenant: logical.TenantAID},
		{role: "tenantB", profile: logical.TenantBProfile, wantTenant: logical.TenantBID},
		{role: "bothTenants", profile: logical.BothTenantsProfile, wantTenant: logical.TenantAID},
		{role: "restrictedA", profile: logical.RestrictedAProfile, wantTenant: logical.TenantAID},
	}
	seenProfiles := map[string]string{}
	groupMinor := ""
	for _, assignment := range assignments {
		profile, nextMinor := validateTenantProfileAssignment(assignment.role, assignment.profile, assignment.wantTenant, "", profiles, seenProfiles, requestedMinor, groupMinor)
		problems = append(problems, profile...)
		groupMinor = nextMinor
	}

	if roles.Physical != nil {
		physical := *roles.Physical
		for name, value := range map[string]string{
			"clusterId": physical.ClusterID, "physicalTenantAId": physical.PhysicalTenantAID,
			"physicalTenantBId": physical.PhysicalTenantBID, "logicalTenantId": physical.LogicalTenantID,
			"tenantAProfile": physical.TenantAProfile, "tenantBProfile": physical.TenantBProfile,
			"defaultRouteProfile":          physical.DefaultRouteProfile,
			"defaultRoutePhysicalTenantId": physical.DefaultRoutePhysicalTenantID,
		} {
			if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
				problems = append(problems, fmt.Errorf("physical %s must be nonempty and trimmed", name))
			}
		}
		if physical.ClusterID != logical.ClusterID {
			problems = append(problems, errors.New("logical and physical role groups must declare the same clusterId"))
		}
		if physical.PhysicalTenantAID == physical.PhysicalTenantBID {
			problems = append(problems, errors.New("physical tenant IDs must be distinct"))
		}
		if physical.DefaultRoutePhysicalTenantID != physical.PhysicalTenantAID && physical.DefaultRoutePhysicalTenantID != physical.PhysicalTenantBID {
			problems = append(problems, errors.New("default route physical tenant ID must identify physical tenant A or B"))
		}
		for _, assignment := range []struct {
			role      string
			profile   string
			wantRoute string
		}{
			{role: "physicalTenantA", profile: physical.TenantAProfile, wantRoute: physical.PhysicalTenantAID},
			{role: "physicalTenantB", profile: physical.TenantBProfile, wantRoute: physical.PhysicalTenantBID},
			{role: "physicalDefaultRoute", profile: physical.DefaultRouteProfile},
		} {
			profileProblems, nextMinor := validateTenantProfileAssignment(assignment.role, assignment.profile, physical.LogicalTenantID, assignment.wantRoute, profiles, seenProfiles, requestedMinor, groupMinor)
			problems = append(problems, profileProblems...)
			groupMinor = nextMinor
		}
	}
	return errors.Join(problems...)
}

// ValidateRequiredTenantCoverage rejects a required physical request unless
// both logical proof roles and the provisioned physical role group exist.
func ValidateRequiredTenantCoverage(roles TenantRoles, coverage TenantCoverage) error {
	switch coverage {
	case TenantCoverageNone, TenantCoverageLogical:
		return nil
	case TenantCoveragePhysical:
		if roles.Physical == nil {
			return errors.New("physical tenant role group is required for physical tenant coverage")
		}
		return nil
	default:
		return fmt.Errorf("unsupported required tenant coverage %q", coverage)
	}
}

// LogicalProfileNames returns the deterministic role-profile set used for
// readiness checks and child execution.
func (roles TenantRoles) LogicalProfileNames() []string {
	if roles.Logical == nil {
		return nil
	}
	return UniqueSorted([]string{
		roles.Logical.TenantAProfile,
		roles.Logical.TenantBProfile,
		roles.Logical.BothTenantsProfile,
		roles.Logical.RestrictedAProfile,
	})
}

// ProfileNames returns every distinct role profile that must pass topology
// readiness before any tenant scenario may mutate its provisioned endpoints.
func (roles TenantRoles) ProfileNames() []string {
	names := roles.LogicalProfileNames()
	if roles.Physical != nil {
		names = append(names, roles.Physical.TenantAProfile, roles.Physical.TenantBProfile, roles.Physical.DefaultRouteProfile)
	}
	return UniqueSorted(names)
}

// PhysicalRouteIdentity validates a normalized v2 base URL and returns its
// explicit physical tenant ID; an empty ID denotes the default route.
func PhysicalRouteIdentity(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("Camunda base URL does not identify a safe absolute route")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 || parts[len(parts)-1] != "v2" {
		return "", fmt.Errorf("Camunda base URL must end in /v2")
	}
	if len(parts) >= 3 && parts[len(parts)-3] == "physical-tenants" {
		identity, decodeErr := url.PathUnescape(parts[len(parts)-2])
		if decodeErr != nil || identity == "" || identity != parts[len(parts)-2] || strings.ContainsAny(identity, `/\\`) {
			return "", fmt.Errorf("Camunda base URL has an invalid physical tenant route")
		}
		return identity, nil
	}
	for _, part := range parts {
		if part == "physical-tenants" {
			return "", fmt.Errorf("Camunda base URL has a misplaced physical tenant route")
		}
	}
	return "", nil
}

// validateTenantProfileAssignment applies the shared credential-free profile,
// minor, logical identity, and physical-route invariants for one named role.
func validateTenantProfileAssignment(role, profileName, wantTenant, wantRoute string, profiles map[string]TenantProfileMetadata, seenProfiles map[string]string, requestedMinor, groupMinor string) ([]error, string) {
	var problems []error
	if previous, duplicate := seenProfiles[profileName]; duplicate {
		return []error{fmt.Errorf("profile %q has conflicting roles %s and %s", profileName, previous, role)}, groupMinor
	}
	seenProfiles[profileName] = role
	profile, exists := profiles[profileName]
	if !exists {
		return []error{fmt.Errorf("tenant role %s references missing default-local profile %q", role, profileName)}, groupMinor
	}
	if profile.LogicalTenantID != wantTenant {
		problems = append(problems, fmt.Errorf("tenant role %s profile %q configures tenant %q, want %q", role, profileName, profile.LogicalTenantID, wantTenant))
	}
	if _, err := CanonicalMinor(profile.ExpectedMinor); err != nil {
		problems = append(problems, fmt.Errorf("tenant role %s profile %q: %w", role, profileName, err))
	}
	if requestedMinor != "" && profile.ExpectedMinor != requestedMinor {
		problems = append(problems, fmt.Errorf("tenant role %s profile %q targets minor %q, want %q", role, profileName, profile.ExpectedMinor, requestedMinor))
	}
	if groupMinor == "" {
		groupMinor = profile.ExpectedMinor
	} else if profile.ExpectedMinor != groupMinor {
		problems = append(problems, fmt.Errorf("tenant role %s profile %q targets minor %q, conflicting with group minor %q", role, profileName, profile.ExpectedMinor, groupMinor))
	}
	if wantRoute != "" || role == "physicalDefaultRoute" {
		route, err := PhysicalRouteIdentity(profile.CamundaBaseURL)
		if err != nil {
			problems = append(problems, fmt.Errorf("tenant role %s profile %q: %w", role, profileName, err))
		} else if route != wantRoute {
			problems = append(problems, fmt.Errorf("tenant role %s profile %q routes to physical tenant %q, want %q", role, profileName, route, wantRoute))
		}
	}
	return problems, groupMinor
}

// requireTenantRolesEOF rejects concatenated JSON values in role metadata.
func requireTenantRolesEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
