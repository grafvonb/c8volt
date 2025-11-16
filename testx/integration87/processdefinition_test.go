//go:build integration

package integration87_test

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

const itBpmnProcessID = "C87_SimpleParentProcess"

func TestIT_SearchProcessDefinitionsLatest_and_GetProcessDefinition(t *testing.T) {
	ctx, api, _, _ := newITClient(t)

	filter := process.ProcessDefinitionFilter{
		BpmnProcessId: itBpmnProcessID,
	}
	latest, err := api.SearchProcessDefinitionsLatest(ctx, filter)
	require.NoError(t, err)
	require.GreaterOrEqual(t, latest.Total, int32(1))
	require.NotEmpty(t, latest.Items)

	// latest should be a single item for that BPMN ID
	pd := latest.Items[0]
	require.Equal(t, itBpmnProcessID, pd.BpmnProcessId)
	require.NotEmpty(t, pd.Key)
	require.Greater(t, pd.ProcessVersion, int32(0))

	// get by key and compare
	got, err := api.GetProcessDefinition(ctx, pd.Key)
	require.NoError(t, err)

	require.Equal(t, pd.Key, got.Key)
	require.Equal(t, pd.BpmnProcessId, got.BpmnProcessId)
	require.Equal(t, pd.ProcessVersion, got.ProcessVersion)
	require.Equal(t, pd.TenantId, got.TenantId)
}

func TestIT_SearchProcessDefinitions_allVersions(t *testing.T) {
	ctx, api, _, _ := newITClient(t)

	filter := process.ProcessDefinitionFilter{
		BpmnProcessId: itBpmnProcessID,
	}

	all, err := api.SearchProcessDefinitions(ctx, filter)
	require.NoError(t, err)
	require.GreaterOrEqual(t, all.Total, int32(1))
	require.NotEmpty(t, all.Items)

	versions := make([]int32, 0, len(all.Items))
	for _, it := range all.Items {
		require.Equal(t, itBpmnProcessID, it.BpmnProcessId)
		require.Greater(t, it.ProcessVersion, int32(0))
		versions = append(versions, it.ProcessVersion)
	}
}
