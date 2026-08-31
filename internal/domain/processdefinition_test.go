// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProcessDefinitionFilterString_RendersOnlyActiveFields verifies debug output only includes configured filters.
func TestProcessDefinitionFilterString_RendersOnlyActiveFields(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessDefinitionFilter{}))

	got := fmt.Sprintf("%+v", ProcessDefinitionFilter{
		Key:               "2251799813685960",
		BpmnProcessId:     "EnquiryProcess",
		TenantId:          "tenant-a",
		ProcessVersion:    2,
		ProcessVersionTag: "2.0.0",
		IsLatestVersion:   true,
	})

	require.Equal(t, `{bpmnProcessId="EnquiryProcess", key="2251799813685960", tenantId="tenant-a", processVersion=2, processVersionTag="2.0.0", isLatestVersion=true}`, got)
}

// TestProcessDefinitionStatisticsFilterString_RendersOnlyActiveFields verifies statistics filters use the shared debug format.
func TestProcessDefinitionStatisticsFilterString_RendersOnlyActiveFields(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessDefinitionStatisticsFilter{}))
	require.Equal(t, `{tenantId="tenant-a"}`, fmt.Sprintf("%+v", ProcessDefinitionStatisticsFilter{TenantId: "tenant-a"}))
}

// TestCompareProcessDefinitionsCanonical_OrdersByContractFields verifies the
// canonical comparator uses exact text fields, numeric version order, and an
// opaque lexical key tie-breaker.
func TestCompareProcessDefinitionsCanonical_OrdersByContractFields(t *testing.T) {
	tests := []struct {
		name string
		a    ProcessDefinition
		b    ProcessDefinition
		want int
	}{
		{
			name: "default tenant uses exact text placement",
			a:    processDefinitionForCanonicalOrder("<default>", "invoice", 1, "1"),
			b:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 1, "1"),
			want: -1,
		},
		{
			name: "tenant comparison is case-sensitive",
			a:    processDefinitionForCanonicalOrder("Tenant-A", "invoice", 1, "1"),
			b:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 1, "1"),
			want: -1,
		},
		{
			name: "bpmn process id comparison is case-sensitive",
			a:    processDefinitionForCanonicalOrder("tenant-a", "Invoice", 1, "1"),
			b:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 1, "1"),
			want: -1,
		},
		{
			name: "version 10 sorts before version 9",
			a:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 10, "1"),
			b:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 9, "1"),
			want: -1,
		},
		{
			name: "opaque key 10 sorts lexically before key 2",
			a:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 10, "10"),
			b:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 10, "2"),
			want: -1,
		},
		{
			name: "equal canonical fields compare equal",
			a:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 10, "2"),
			b:    processDefinitionForCanonicalOrder("tenant-a", "invoice", 10, "2"),
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareProcessDefinitionsCanonical(tt.a, tt.b)
			reverse := CompareProcessDefinitionsCanonical(tt.b, tt.a)
			require.Equal(t, tt.want, got)
			require.Equal(t, -tt.want, reverse)
		})
	}
}

// TestSortProcessDefinitionsCanonical_CoversCollectionEdges verifies empty,
// single-item, already sorted, and shuffled collections keep the exact
// canonical key sequence.
func TestSortProcessDefinitionsCanonical_CoversCollectionEdges(t *testing.T) {
	sorted := []ProcessDefinition{
		processDefinitionForCanonicalOrder("<default>", "Invoice", 1, "1"),
		processDefinitionForCanonicalOrder("<default>", "invoice", 10, "10"),
		processDefinitionForCanonicalOrder("<default>", "invoice", 10, "2"),
		processDefinitionForCanonicalOrder("<default>", "invoice", 9, "1"),
		processDefinitionForCanonicalOrder("Tenant-A", "invoice", 1, "1"),
		processDefinitionForCanonicalOrder("tenant-a", "Payment", 1, "1"),
		processDefinitionForCanonicalOrder("tenant-a", "invoice", 10, "1"),
		processDefinitionForCanonicalOrder("tenant-a", "invoice", 9, "1"),
		processDefinitionForCanonicalOrder("tenant-b", "invoice", 1, "1"),
	}

	tests := []struct {
		name  string
		input []ProcessDefinition
		want  []string
	}{
		{
			name:  "empty collection stays empty",
			input: nil,
			want:  []string{},
		},
		{
			name:  "single collection stays unchanged",
			input: []ProcessDefinition{processDefinitionForCanonicalOrder("tenant-a", "invoice", 1, "only")},
			want:  []string{"only"},
		},
		{
			name:  "already sorted collection stays ordered",
			input: sorted,
			want:  processDefinitionCanonicalKeys(sorted),
		},
		{
			name: "shuffled collection becomes canonical",
			input: []ProcessDefinition{
				sorted[7],
				sorted[2],
				sorted[4],
				sorted[0],
				sorted[8],
				sorted[5],
				sorted[1],
				sorted[6],
				sorted[3],
			},
			want: processDefinitionCanonicalKeys(sorted),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Clone(tt.input)
			SortProcessDefinitionsCanonical(got)
			require.Equal(t, tt.want, processDefinitionCanonicalKeys(got))
		})
	}
}

// processDefinitionForCanonicalOrder keeps comparator fixture fields explicit.
func processDefinitionForCanonicalOrder(tenantID, bpmnProcessID string, version int32, key string) ProcessDefinition {
	return ProcessDefinition{
		TenantId:       tenantID,
		BpmnProcessId:  bpmnProcessID,
		ProcessVersion: version,
		Key:            key,
	}
}

// processDefinitionCanonicalKeys extracts the ordered identity checked by the
// domain comparator tests.
func processDefinitionCanonicalKeys(definitions []ProcessDefinition) []string {
	keys := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		keys = append(keys, definition.Key)
	}
	return keys
}
