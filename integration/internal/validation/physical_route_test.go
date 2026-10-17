// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	"github.com/stretchr/testify/require"
)

// TestV810ClientPreservesPhysicalTenantRoute proves generated client request
// resolution appends operation paths beneath the selected physical v2 prefix.
func TestV810ClientPreservesPhysicalTenantRoute(t *testing.T) {
	t.Parallel()
	requestedPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPath = request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"brokers":[],"clusterSize":0,"gateways":[],"partitionsCount":0,"replicationFactor":0}`))
	}))
	t.Cleanup(server.Close)

	client, err := camundav810.NewClientWithResponses(server.URL + "/physical-tenants/tenant-a/v2")
	require.NoError(t, err)
	response, err := client.GetTopologyWithResponse(context.Background())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode())
	require.Equal(t, "/physical-tenants/tenant-a/v2/topology", requestedPath)
}
