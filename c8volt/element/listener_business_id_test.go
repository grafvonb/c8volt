// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"encoding/json"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestListenerBusinessIDConversion preserves optional identity and owns its pointer at the facade boundary.
func TestListenerBusinessIDConversion(t *testing.T) {
	t.Parallel()
	for _, value := range []*string{nil, new(""), new(" old,\"|<tag>\n ")} {
		source := d.RuntimeListenerJob{JobKey: "job", BusinessId: value}
		got := fromDomainRuntimeListenerJob(source)
		require.Equal(t, value, got.BusinessId)
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(raw, &decoded))
		if value == nil {
			require.NotContains(t, decoded, "businessId")
		} else {
			require.Equal(t, *value, decoded["businessId"])
			before := *value
			*value = "mutated"
			require.Equal(t, before, *got.BusinessId)
		}
	}
}

// TestListenerBusinessIDSections preserves unrequested, requested-nil, requested-empty and populated collection shapes.
func TestListenerBusinessIDSections(t *testing.T) {
	t.Parallel()
	require.Nil(t, fromDomainRuntimeListenerJobsPtr(nil))
	var absent []d.RuntimeListenerJob
	empty := []d.RuntimeListenerJob{}
	populated := []d.RuntimeListenerJob{{JobKey: "old", BusinessId: new("old")}, {JobKey: "absent"}, {JobKey: "empty", BusinessId: new("")}}
	for _, source := range []*[]d.RuntimeListenerJob{&absent, &empty, &populated} {
		got := fromDomainRuntimeListenerJobsPtr(source)
		require.NotNil(t, got)
		require.Equal(t, *source == nil, *got == nil)
		require.Len(t, *got, len(*source))
		if len(*source) > 0 {
			*(*source)[0].BusinessId = "changed"
			require.Equal(t, "old", *(*got)[0].BusinessId)
			require.Nil(t, (*got)[1].BusinessId)
			require.Equal(t, "", *(*got)[2].BusinessId)
		}
	}
}
