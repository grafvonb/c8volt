// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"encoding/json"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestConvertProcessDefinitionSuspendedStatistics keeps the duplicated ops
// projection consistent with process statistics, including availability and omission.
func TestConvertProcessDefinitionSuspendedStatistics(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		suspended int64
		supported bool
		want      string
	}{
		{"supported", 3, true, `{"canceled":0,"completed":0,"incidents":0,"active":2,"suspended":3,"suspendedCountSupported":true}`},
		{"supported zero", 0, true, `{"canceled":0,"completed":0,"incidents":0,"active":2,"suspendedCountSupported":true}`},
		{"unavailable", 0, false, `{"canceled":0,"completed":0,"incidents":0,"active":2}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unfinished := int64(5)
			source := d.ProcessDefinition{Key: "123", Statistics: &d.ProcessDefinitionStatistics{
				Active: 2, Suspended: tc.suspended, SuspendedCountSupported: tc.supported, Unfinished: &unfinished,
			}}
			converted := fromDomainProcessDefinition(source)
			require.NotNil(t, converted.Statistics)
			require.Equal(t, int64(2), converted.Statistics.Active)
			require.Equal(t, tc.suspended, converted.Statistics.Suspended)
			require.Equal(t, tc.supported, converted.Statistics.SuspendedCountSupported)
			raw, err := json.Marshal(converted.Statistics)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(raw))
		})
	}
	require.Nil(t, fromDomainProcessDefinition(d.ProcessDefinition{Key: "123"}).Statistics)
}
