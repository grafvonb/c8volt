// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/resource"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	"github.com/stretchr/testify/require"
)

// TestGetFamilyResourceMetadata uses a real generic-resource key, rather than
// treating a BPMN deployment key as a resource key and only testing rejection.
func TestGetFamilyResourceMetadata(t *testing.T) {
	for _, profile := range selectedBatchProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			client, err := newLiveBatchClient(profile.Name)
			require.NoError(t, err)
			name := suite.marker + "-resource.txt"
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("resources", name)
			require.NoError(t, err)
			_, err = part.Write([]byte(suite.marker))
			require.NoError(t, err)
			tenant := profile.Tenant
			if tenant == "" {
				tenant = "<default>"
			}
			require.NoError(t, writer.WriteField("tenantId", tenant))
			require.NoError(t, writer.Close())
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			// CLI deployment accepts BPMN, so a native generic-resource deployment
			// is the prerequisite. Authentication reuses the selected local profile.
			deployed, err := client.generated.CreateDeploymentWithBodyWithResponse(ctx, writer.FormDataContentType(), &body)
			require.NoError(t, err)
			require.NotNil(t, deployed)
			require.Equal(t, 200, deployed.StatusCode(), string(deployed.Body))
			require.NotNil(t, deployed.JSON200)
			for _, item := range deployed.JSON200.Deployments {
				if item.Resource == nil {
					continue
				}
				r := *item.Resource
				require.NotEmpty(t, r.ResourceKey)
				require.NotEmpty(t, r.ResourceId)
				t.Cleanup(func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					deleted, err := client.generated.DeleteResourceOpWithResponse(ctx, r.ResourceKey, camundav810.DeleteResourceOpJSONRequestBody{})
					require.NoError(t, err)
					require.NotNil(t, deleted)
					require.Equal(t, 200, deleted.StatusCode(), string(deleted.Body))
					require.Eventually(t, func() bool {
						found, err := client.generated.GetResourceWithResponse(ctx, r.ResourceKey)
						return err == nil && found != nil && found.StatusCode() == 404
					}, 20*time.Second, time.Second)
					writeJSON(t, "resource-cleanup-"+profile.Name+".json", map[string]any{"key": r.ResourceKey, "removed": true})
				})
				writeJSON(t, "resource-setup-"+profile.Name+".json", map[string]any{"setup": "native generic-resource deployment", "resource": r, "deploymentKey": deployed.JSON200.DeploymentKey})
				require.NotEqual(t, deployed.JSON200.DeploymentKey, r.ResourceKey)
				// Deployment acceptance can precede exporter visibility. Establish
				// the fixture independently before testing a single CLI lookup.
				require.Eventually(t, func() bool {
					found, err := client.generated.GetResourceWithResponse(ctx, r.ResourceKey)
					return err == nil && found != nil && found.JSON200 != nil && found.JSON200.ResourceKey == r.ResourceKey && found.JSON200.ResourceName == name
				}, 20*time.Second, time.Second)
				writeJSON(t, "resource-ready-"+profile.Name+".json", map[string]any{"key": r.ResourceKey, "indexed": true})
				human := runC8VoltForProfile(t, profile.Name, "resource-real-key-human", "get", "resource", "--id", r.ResourceKey)
				require.NoError(t, human.Err, human.Stderr)
				require.Contains(t, human.Stdout, r.ResourceKey)
				require.Contains(t, human.Stdout, name)
				result := runC8VoltForProfile(t, profile.Name, "resource-real-key-json", "--json", "get", "resource", "--id", r.ResourceKey)
				require.NoError(t, result.Err, result.Stderr)
				var got resource.Resource
				require.NoError(t, decodeCommandPayload(result.Stdout, &got))
				require.Equal(t, r.ResourceKey, got.Key)
				require.Equal(t, r.ResourceId, got.ID)
				require.Equal(t, name, got.Name)
				keys := runC8VoltForProfile(t, profile.Name, "resource-logical-id-keys-only", "--keys-only", "get", "resource", "--id", r.ResourceKey)
				require.NoError(t, keys.Err, keys.Stderr)
				require.Equal(t, r.ResourceId+"\n", keys.Stdout)
				return
			}
			t.Fatal("generic-resource deployment returned no resource metadata")
		})
	}
}
