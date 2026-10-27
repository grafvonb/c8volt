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

// TestConvertProcessDefinitionSuspendedStatistics preserves active semantics and
// distinguishes supported zero from unavailable data without leaking union evidence.
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
			stats := d.ProcessDefinitionStatistics{Active: 2, Suspended: tc.suspended, SuspendedCountSupported: tc.supported, Unfinished: &unfinished}
			source := d.ProcessDefinition{Key: "123", Statistics: &stats}
			for _, converted := range []ProcessDefinition{fromDomainProcessDefinition(source), fromDomainProcessDefinitions([]d.ProcessDefinition{source}).Items[0]} {
				require.NotNil(t, converted.Statistics)
				require.Equal(t, int64(2), converted.Statistics.Active)
				require.Equal(t, tc.suspended, converted.Statistics.Suspended)
				require.Equal(t, tc.supported, converted.Statistics.SuspendedCountSupported)
				raw, err := json.Marshal(converted.Statistics)
				require.NoError(t, err)
				require.JSONEq(t, tc.want, string(raw))
			}
			raw, err := json.Marshal(stats)
			require.NoError(t, err)
			require.NotContains(t, string(raw), "unfinished")
			require.NotContains(t, string(raw), "Unfinished")
		})
	}
	require.Nil(t, fromDomainProcessDefinition(d.ProcessDefinition{Key: "123"}).Statistics)
}

// TestConvertProcessDefinitionLifecycle verifies shared conversion preserves fresh
// observations across all public result shapes, including omission after a value.
func TestConvertProcessDefinitionLifecycle(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"ACTIVE", "DRAINING", "DELETED", " future-State ", ""} {
		t.Run(state, func(t *testing.T) {
			source := d.ProcessDefinition{Key: "123", State: state}
			page := d.ProcessDefinitionPage{Items: []d.ProcessDefinition{source}}
			for _, got := range []ProcessDefinition{
				fromDomainProcessDefinition(source),
				fromDomainProcessDefinitions([]d.ProcessDefinition{source}).Items[0],
				fromDomainProcessDefinitionPage(page).Items[0],
				fromDomainProcessDefinitionSearchPageStep(d.ProcessDefinitionSearchPageStep{Page: page}).Page.Items[0],
				fromDomainProcessDefinitionSearchPagesResult(d.ProcessDefinitionSearchPagesResult{Items: page.Items}).Items[0],
				fromDomainProcessDefinitionWatchSnapshot(d.ProcessDefinitionWatchSnapshot{Items: page.Items}).Items[0],
			} {
				require.Equal(t, state, got.State)
				raw, err := json.Marshal(got)
				require.NoError(t, err)
				var object map[string]any
				require.NoError(t, json.Unmarshal(raw, &object))
				if state == "" {
					require.NotContains(t, object, "state")
				} else {
					require.Equal(t, state, object["state"])
				}
			}
		})
	}
}
