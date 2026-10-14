// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// TenantRoleLogical identifies the logical-tenant proof group in catalog and
// target records.
const TenantRoleLogical = "logical"

// TenantRoles is credential-free metadata that maps validation roles to
// profiles already present in the operator's default local configuration.
type TenantRoles struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Logical       *LogicalTenantRoles `json:"logical,omitempty"`
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

// TenantProfileMetadata contains only the non-secret local profile fields
// needed to reject missing, conflicting, or foreign-version role assignments.
type TenantProfileMetadata struct {
	Name            string
	ExpectedMinor   string
	LogicalTenantID string
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
		if previous, duplicate := seenProfiles[assignment.profile]; duplicate {
			problems = append(problems, fmt.Errorf("profile %q has conflicting roles %s and %s", assignment.profile, previous, assignment.role))
			continue
		}
		seenProfiles[assignment.profile] = assignment.role
		profile, exists := profiles[assignment.profile]
		if !exists {
			problems = append(problems, fmt.Errorf("logical role %s references missing default-local profile %q", assignment.role, assignment.profile))
			continue
		}
		if profile.LogicalTenantID != assignment.wantTenant {
			problems = append(problems, fmt.Errorf("logical role %s profile %q configures tenant %q, want %q", assignment.role, assignment.profile, profile.LogicalTenantID, assignment.wantTenant))
		}
		if _, err := CanonicalMinor(profile.ExpectedMinor); err != nil {
			problems = append(problems, fmt.Errorf("logical role %s profile %q: %w", assignment.role, assignment.profile, err))
		}
		if requestedMinor != "" && profile.ExpectedMinor != requestedMinor {
			problems = append(problems, fmt.Errorf("logical role %s profile %q targets minor %q, want %q", assignment.role, assignment.profile, profile.ExpectedMinor, requestedMinor))
		}
		if groupMinor == "" {
			groupMinor = profile.ExpectedMinor
		} else if profile.ExpectedMinor != groupMinor {
			problems = append(problems, fmt.Errorf("logical role %s profile %q targets minor %q, conflicting with group minor %q", assignment.role, assignment.profile, profile.ExpectedMinor, groupMinor))
		}
	}
	return errors.Join(problems...)
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
