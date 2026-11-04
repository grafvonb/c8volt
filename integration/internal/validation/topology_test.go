// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTopology(t *testing.T) {
	for _, tc := range []struct {
		name    string
		brokers string
		wantErr string
	}{
		{"lowercase health", `[{"version":"8.10.6","partitions":[{"health":"healthy"}]}]`, ""},
		{"uppercase health", `[{"version":"8.10.6","partitions":[{"health":"HEALTHY"}]}]`, ""},
		{"patch difference", `[{"version":"8.10.5","partitions":[{"health":"Healthy"}]}]`, ""},
		{"idle broker", `[{"version":"8.10.6"},{"version":"8.10.6","partitions":[{"health":"healthy"}]}]`, ""},
		{"mixed minor", `[{"version":"8.9.8","partitions":[{"health":"healthy"}]}]`, "differs"},
		{"unknown broker", `[{"version":"unknown","partitions":[{"health":"healthy"}]}]`, "version"},
		{"missing brokers", `[]`, "no brokers"},
		{"missing partitions", `[{"version":"8.10.6"}]`, "no reported partitions"},
		{"unhealthy", `[{"version":"8.10.6","partitions":[{"health":"unhealthy"}]}]`, "health"},
		{"unknown health", `[{"version":"8.10.6","partitions":[{}]}]`, "health"},
		{"second broker", `[{"version":"8.10.6","partitions":[{"health":"healthy"}]},{"version":"8.9.8","partitions":[{"health":"healthy"}]}]`, "broker 1"},
		{"second partition", `[{"version":"8.10.6","partitions":[{"health":"healthy"},{"health":"dead"}]}]`, "partition 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cluster ClusterTopology
			require.NoError(t, json.Unmarshal([]byte(`{"gateway_version":"8.10.6","broker_details":`+tc.brokers+`}`), &cluster))
			err := ValidateTopology(cluster)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
	require.Error(t, ValidateTopology(ClusterTopology{GatewayVersion: "unknown"}))
}
