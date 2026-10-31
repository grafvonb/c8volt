// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"context"
	"net/http"
	"testing"

	native "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestUserTaskBusinessIDNativeRead retains backend absence/empty values and copies pointers without parent reads.
func TestUserTaskBusinessIDNativeRead(t *testing.T) {
	t.Parallel()
	for _, id := range []*string{nil, toolx.Ptr(""), toolx.Ptr("old,\"ID\"\n")} {
		original := toolx.CopyPtr(id)
		calls := 0
		svc := newTestService(t, &mockUserTaskClient{getUserTaskWithResponse: func(context.Context, native.UserTaskKey, ...native.RequestEditorFn) (*native.GetUserTaskResponse, error) {
			calls++
			return &native.GetUserTaskResponse{HTTPResponse: newHTTPResponse(http.MethodGet, "https://fixture.invalid/task", 200, "200 OK"), JSON200: &native.UserTaskResult{UserTaskKey: "1", State: "CREATED", ProcessInstanceKey: "2", BusinessId: id}}, nil
		}}, "")
		got, err := svc.GetNativeUserTask(context.Background(), "1")
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Equal(t, original, got.BusinessId)
		if id != nil {
			*id = "changed"
			require.Equal(t, original, got.BusinessId)
		}
	}
}
