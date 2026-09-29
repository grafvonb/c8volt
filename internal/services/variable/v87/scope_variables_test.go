// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"testing"

	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

type rejectingScopeVariableClient struct{}

// SearchVariablesForProcessInstancesWithResponse panics if unsupported scope writes touch transport.
func (rejectingScopeVariableClient) SearchVariablesForProcessInstancesWithResponse(context.Context, operatev87.SearchVariablesForProcessInstancesJSONRequestBody, ...operatev87.RequestEditorFn) (*operatev87.SearchVariablesForProcessInstancesResponse, error) {
	panic("unexpected transport call")
}

// TestUpdateScopeVariablesUnsupported verifies v8.7 rejects scope writes without transport access.
func TestUpdateScopeVariablesUnsupported(t *testing.T) {
	svc := &Service{co: rejectingScopeVariableClient{}}

	got, err := svc.UpdateScopeVariables(context.Background(), "456", map[string]any{"foo": "bar"})

	require.ErrorIs(t, err, d.ErrUnsupported)
	require.Equal(t, d.ScopeVariableUpdateResponse{ScopeKey: "456"}, got)
}
