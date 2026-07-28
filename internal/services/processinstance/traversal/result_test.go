// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package traversal

import (
	"context"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type stubResultTraversalAPI struct{}

// Ancestry returns a fixed root path for progress-oriented result builder tests.
func (stubResultTraversalAPI) Ancestry(context.Context, string, ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error) {
	return "root", []string{"child", "root"}, map[string]d.ProcessInstance{"root": {Key: "root"}, "child": {Key: "child"}}, nil
}

// Descendants returns a fixed family for progress-oriented result builder tests.
func (stubResultTraversalAPI) Descendants(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	return []string{"root", "child"}, map[string][]string{"root": []string{"child"}}, map[string]d.ProcessInstance{"root": {Key: "root"}, "child": {Key: "child"}}, nil
}

// Family is unused by the structured result builders and exists to satisfy LegacyAPI.
func (stubResultTraversalAPI) Family(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	return nil, nil, nil, nil
}

// TestBuildFamilyResultEmitsFrozenProgress verifies keyed walk reports exact progress after the immutable family scope is known.
func TestBuildFamilyResultEmitsFrozenProgress(t *testing.T) {
	var events []d.OpsProgressEvent
	got, err := BuildFamilyResult(context.Background(), stubResultTraversalAPI{}, "child", services.WithProgress(func(event d.OpsProgressEvent) {
		events = append(events, event)
	}))

	require.NoError(t, err)
	require.Equal(t, []string{"root", "child"}, got.Keys)
	require.Len(t, events, 3)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "walking process-instance ancestry", CoreResource: "process instance(s)", Done: 2, Total: 2}, *events[0].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "walking process-instance descendants", CoreResource: "process instance(s)", Done: 2, Total: 2}, *events[1].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "walking process-instance family", CoreResource: "process instance(s)", Done: 2, Total: 2}, *events[2].FrozenScope)
}
