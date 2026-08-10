// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/stretchr/testify/require"
)

// TestGetProcessDefinitionWatchSnapshotRequestBroadMode verifies unselected
// watch mode keeps using the broad discovery request shape.
func TestGetProcessDefinitionWatchSnapshotRequestBroadMode(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	request := newGetProcessDefinitionWatchSnapshotRequest(process.ProcessDefinitionFilter{})

	require.True(t, request.WatchAllWhenUnselected)
	require.Empty(t, request.Key)
	require.Equal(t, process.ProcessDefinitionFilter{}, request.Filter)
	require.Equal(t, consts.MaxPISearchSize, request.Page.Size)
	require.False(t, request.Latest)
}

// TestGetProcessDefinitionWatchSnapshotRequestPreservesSelectors verifies
// selector-backed watch refreshes keep their selector and do not broaden scope.
func TestGetProcessDefinitionWatchSnapshotRequestPreservesSelectors(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	flagGetPDLatest = true
	flagGetPDBatchSize = 25
	filter := process.ProcessDefinitionFilter{
		BpmnProcessId:     "invoice",
		ProcessVersion:    3,
		ProcessVersionTag: "stable",
	}

	request := newGetProcessDefinitionWatchSnapshotRequest(filter)

	require.False(t, request.WatchAllWhenUnselected)
	require.Empty(t, request.Key)
	require.Equal(t, filter, request.Filter)
	require.Equal(t, int32(25), request.Page.Size)
	require.True(t, request.Latest)
}
