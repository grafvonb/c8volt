// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

// TestTenantProfileMetadataPreservesInheritedTenantAndVersion verifies role
// validation observes the same base/profile overlay fields as command runs.
func TestTenantProfileMetadataPreservesInheritedTenantAndVersion(t *testing.T) {
	t.Parallel()
	metadata, err := tenantProfileMetadata(tenantConfigFile{
		App:  tenantConfigApp{CamundaVersion: "8.10.3", Tenant: "tenant-a"},
		APIs: config.APIs{Camunda: config.API{BaseURL: "https://gateway.example.test"}},
		Profiles: map[string]tenantConfigProfile{
			"inherited": {},
			"override": {
				App:  tenantConfigApp{CamundaVersion: "8.9", Tenant: "tenant-b"},
				APIs: config.APIs{Camunda: config.API{BaseURL: "https://gateway.example.test/physical-tenants/b"}},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "8.10", metadata["inherited"].ExpectedMinor)
	require.Equal(t, "tenant-a", metadata["inherited"].LogicalTenantID)
	require.Equal(t, "https://gateway.example.test/v2", metadata["inherited"].CamundaBaseURL)
	require.Equal(t, "8.9", metadata["override"].ExpectedMinor)
	require.Equal(t, "tenant-b", metadata["override"].LogicalTenantID)
	require.Equal(t, "https://gateway.example.test/physical-tenants/b/v2", metadata["override"].CamundaBaseURL)
}
