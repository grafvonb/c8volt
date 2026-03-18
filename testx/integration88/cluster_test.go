//go:build integration

package integration88_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIT_GetClusterTopology(t *testing.T) {
	ctx, api, cfg, log := newITClient(t)

	topology, err := api.GetClusterTopology(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, topology.ClusterSize, int32(1))
	require.GreaterOrEqual(t, topology.PartitionsCount, int32(1))
	require.NotEmpty(t, topology.Brokers)

	b := topology.Brokers[0]
	require.NotZero(t, b.Port)
	require.NotEmpty(t, b.Host)
	require.Contains(t, b.Version, "8.8.")

	deployTestProcessDefinitions(t, ctx, api, cfg, log)
}

func TestIT_GetClusterLicense(t *testing.T) {
	ctx, api, _, _ := newITClient(t)

	license, err := api.GetClusterLicense(ctx)
	require.NoError(t, err)

	require.Equal(t, "Self-Managed Enterprise", license.LicenseType)
	require.True(t, license.ValidLicense)
	require.NotNil(t, license.ExpiresAt)
	require.NotNil(t, license.IsCommercial)
}
