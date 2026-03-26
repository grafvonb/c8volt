//go:build integration

package integration87_test

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
	require.Contains(t, b.Version, "8.7.")

	deployTestProcessDefinitions(t, ctx, api, cfg, log)
}

func TestIT_GetClusterLicense(t *testing.T) {
	ctx, api, _, _ := newITClient(t)

	license, err := api.GetClusterLicense(ctx)
	require.NoError(t, err)

	require.Equal(t, "SaaS", license.LicenseType)
	require.True(t, license.ValidLicense)
	require.Nil(t, license.ExpiresAt)
	require.Nil(t, license.IsCommercial)
}
