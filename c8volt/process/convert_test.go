// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"encoding/json"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestConvertProcessInstanceSuspendedDate verifies both facade directions and
// nested payloads preserve only the metadata present in the current snapshot.
func TestConvertProcessInstanceSuspendedDate(t *testing.T) {
	t.Parallel()
	for _, date := range []string{"2026-10-17T08:30:00.123456789Z", ""} {
		source := d.ProcessInstance{Key: "123", State: d.StateSuspended, SuspendedDate: date}
		public := fromDomainProcessInstance(source)
		require.Equal(t, date, public.SuspendedDate)
		require.Equal(t, source, toDomainProcessInstance(public))
		payloads := map[string]any{
			"instance":   public,
			"collection": fromDomainProcessInstances([]d.ProcessInstance{source}),
			"incident":   fromDomainIncidentEnrichedProcessInstance(d.IncidentEnrichedProcessInstance{Item: source}),
			"variable":   fromDomainVariableEnrichedProcessInstance(d.VariableEnrichedProcessInstance{Item: source}),
			"element":    fromDomainElementEnrichedProcessInstance(d.ElementEnrichedProcessInstance{Item: source}),
		}
		for name, payload := range payloads {
			t.Run(name+"/"+date, func(t *testing.T) {
				raw, err := json.Marshal(payload)
				require.NoError(t, err)
				var decoded map[string]any
				require.NoError(t, json.Unmarshal(raw, &decoded))
				item := decoded
				if name == "collection" {
					item = decoded["items"].([]any)[0].(map[string]any)
				} else if name != "instance" {
					item = decoded["item"].(map[string]any)
				}
				require.Equal(t, "SUSPENDED", item["state"])
				if date == "" {
					require.NotContains(t, item, "suspendedDate")
				} else {
					require.Equal(t, date, item["suspendedDate"])
				}
			})
		}
	}
}
