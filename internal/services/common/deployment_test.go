// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"io"
	"mime"
	"mime/multipart"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestBuildDeploymentBody_IncludesTenantAndResources verifies the exact multipart contract shared by
// all versioned resource services: tenant goes into the optional tenantId field, every deployment unit
// is sent as a resources file part, and filenames/data survive round-tripping through a multipart parser.
func TestBuildDeploymentBody_IncludesTenantAndResources(t *testing.T) {
	contentType, body, err := BuildDeploymentBody("tenant-a", []d.DeploymentUnitData{
		{Name: "one.bpmn", Data: []byte("one")},
		{Name: "two.bpmn", Data: []byte("two")},
	})
	require.NoError(t, err)

	mediaType, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mediaType)

	reader := multipart.NewReader(body, params["boundary"])
	form, err := reader.ReadForm(1024)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, form.RemoveAll())
	})

	require.Equal(t, []string{"tenant-a"}, form.Value["tenantId"])
	files := form.File["resources"]
	require.Len(t, files, 2)
	require.Equal(t, "one.bpmn", files[0].Filename)
	require.Equal(t, "two.bpmn", files[1].Filename)

	first, err := files[0].Open()
	require.NoError(t, err)
	defer first.Close()
	firstBody, err := io.ReadAll(first)
	require.NoError(t, err)
	require.Equal(t, "one", string(firstBody))
}
