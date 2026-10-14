// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCanonicalAndObservedMinorParsing keeps strict request syntax separate
// from full server-version parsing.
func TestCanonicalAndObservedMinorParsing(t *testing.T) {
	t.Parallel()
	for _, minor := range []string{"8.8", "8.9", "8.10"} {
		got, err := CanonicalMinor(minor)
		require.NoError(t, err)
		require.Equal(t, minor, got)
	}
	for _, value := range []string{"v8.10", "8.10.3", "8.10-SNAPSHOT", "18.10", " 8.10 "} {
		_, err := CanonicalMinor(value)
		require.Error(t, err, value)
	}
	got, err := ObservedMinor("v8.10.3")
	require.NoError(t, err)
	require.Equal(t, "8.10", got)
	for _, value := range []string{"8.10-SNAPSHOT", "8.10.3-alpha", "18.10.3", "8"} {
		_, err := ObservedMinor(value)
		require.Error(t, err, value)
	}
}

// TestValidateRequestRejectsUnsafeOrIncompleteReleaseScope proves strict input
// failures are available before mutation starts.
func TestValidateRequestRejectsUnsafeOrIncompleteReleaseScope(t *testing.T) {
	t.Parallel()
	catalog := CoverageCatalog{Revision: "rev"}
	base := ValidationRequest{
		SchemaVersion:          SchemaVersion,
		RunID:                  "run-1",
		Mode:                   ModeRelease,
		RequestedMinor:         "8.10",
		Profiles:               []string{"dev810"},
		RequiredTenantCoverage: TenantCoverageNone,
		CatalogRevision:        "rev",
		StartedAt:              time.Now(),
		EvidenceRoot:           filepath.Join(t.TempDir(), "evidence"),
	}
	require.NoError(t, ValidateRequest(base, catalog))

	cases := map[string]func(*ValidationRequest){
		"missing profile":  func(request *ValidationRequest) { request.Profiles = nil },
		"old strict minor": func(request *ValidationRequest) { request.RequestedMinor = "8.8" },
		"patch request":    func(request *ValidationRequest) { request.RequestedMinor = "8.10.3" },
		"relative root":    func(request *ValidationRequest) { request.EvidenceRoot = "relative" },
		"duplicate profile": func(request *ValidationRequest) {
			request.Profiles = []string{"dev810", "dev810"}
		},
		"profile path": func(request *ValidationRequest) { request.Profiles = []string{"../dev810"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			request := base
			request.Profiles = append([]string(nil), base.Profiles...)
			mutate(&request)
			require.Error(t, ValidateRequest(request, catalog))
		})
	}
}

// TestValidateChildPathRejectsEscapes preserves the invocation-root boundary
// for child directories and observation files.
func TestValidateChildPathRejectsEscapes(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "invocation")
	require.NoError(t, ValidateChildPath(root, "slice/profile"))
	require.Error(t, ValidateChildPath(root, "../foreign"))
	require.Error(t, ValidateChildPath(root, root))
	require.Error(t, ValidateChildPath(root, filepath.Join(filepath.Dir(root), "foreign")))
}
