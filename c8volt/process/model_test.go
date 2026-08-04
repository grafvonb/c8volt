// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"fmt"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestProcessDefinitionFilterString_RendersOnlyActiveFields(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessDefinitionFilter{}))

	got := fmt.Sprintf("%+v", ProcessDefinitionFilter{
		Key:               "2251799813685960",
		BpmnProcessId:     "EnquiryProcess",
		ProcessVersion:    2,
		ProcessVersionTag: "2.0.0",
	})

	require.Equal(t, `{key="2251799813685960", bpmnProcessId="EnquiryProcess", processVersion=2, processVersionTag="2.0.0"}`, got)
}

func TestProcessInstanceFilterString_RendersOptionalBooleansConsistently(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessInstanceFilter{}))
	require.NotContains(t, fmt.Sprintf("%+v", ProcessInstanceFilter{}), "<nil>")

	hasParent := true
	hasIncident := false
	got := fmt.Sprintf("%+v", ProcessInstanceFilter{
		BpmnProcessId: "order",
		HasParent:     &hasParent,
		HasIncident:   &hasIncident,
	})

	require.Equal(t, `{bpmnProcessId="order", hasParent=true, hasIncident=false}`, got)
}

// TestProcessDefinitionWatchSnapshotRequestConversionPreservesSelectorFields
// verifies watch-specific public selector state reaches the service request.
func TestProcessDefinitionWatchSnapshotRequestConversionPreservesSelectorFields(t *testing.T) {
	got := toDomainProcessDefinitionWatchSnapshotRequest(ProcessDefinitionWatchSnapshotRequest{
		Key: "2251799813685255",
		Filter: ProcessDefinitionFilter{
			BpmnProcessId:     "invoice",
			ProcessVersion:    4,
			ProcessVersionTag: "stable",
		},
		Page:                   ProcessDefinitionPageRequest{From: 5, Size: 10, After: "cursor-2"},
		Latest:                 true,
		WatchAllWhenUnselected: true,
	})

	require.Equal(t, d.ProcessDefinitionWatchSnapshotRequest{
		Key: "2251799813685255",
		Filter: d.ProcessDefinitionFilter{
			BpmnProcessId:     "invoice",
			ProcessVersion:    4,
			ProcessVersionTag: "stable",
		},
		Page:                   d.ProcessDefinitionPageRequest{From: 5, Size: 10, After: "cursor-2"},
		Latest:                 true,
		WatchAllWhenUnselected: true,
	}, got)
}

// TestProcessDefinitionWatchSnapshotConversionPreservesPagingMetadata verifies
// service-collected snapshot counts and reported totals cross the facade boundary.
func TestProcessDefinitionWatchSnapshotConversionPreservesPagingMetadata(t *testing.T) {
	got := fromDomainProcessDefinitionWatchSnapshot(d.ProcessDefinitionWatchSnapshot{
		Items: []d.ProcessDefinition{{
			Key:           "2251799813685255",
			BpmnProcessId: "invoice",
			Statistics: &d.ProcessDefinitionStatistics{
				Active:                 2,
				IncidentCountSupported: true,
			},
		}},
		Total: 1,
		Pages: 2,
		ReportedTotal: &d.ProcessDefinitionReportedTotal{
			Count: 7,
			Kind:  d.ProcessDefinitionReportedTotalKindExact,
		},
		Empty: false,
	})

	require.EqualValues(t, 1, got.Total)
	require.EqualValues(t, 2, got.Pages)
	require.False(t, got.Empty)
	require.NotNil(t, got.ReportedTotal)
	require.Equal(t, ProcessDefinitionReportedTotalKindExact, got.ReportedTotal.Kind)
	require.EqualValues(t, 7, got.ReportedTotal.Count)
	require.Len(t, got.Items, 1)
	require.Equal(t, "invoice", got.Items[0].BpmnProcessId)
	require.NotNil(t, got.Items[0].Statistics)
	require.True(t, got.Items[0].Statistics.IncidentCountSupported)
}
