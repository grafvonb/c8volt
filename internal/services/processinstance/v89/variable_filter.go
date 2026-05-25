// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"fmt"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

// newVariableValueFiltersPtr builds native v8.9 variable filters from
// version-neutral clauses while preserving the command's all-clauses order.
func newVariableValueFiltersPtr(filters d.ProcessInstanceVariableFilterSet) (*[]camundav89.VariableValueFilterProperty, error) {
	if len(filters.Clauses) == 0 {
		return nil, nil
	}
	if err := filters.Validate(); err != nil {
		return nil, err
	}
	out := make([]camundav89.VariableValueFilterProperty, 0, len(filters.Clauses))
	for _, clause := range filters.Clauses {
		valueFilter, err := newVariableStringFilter(clause)
		if err != nil {
			return nil, err
		}
		out = append(out, camundav89.VariableValueFilterProperty{
			Name:  clause.Name,
			Value: valueFilter,
		})
	}
	return &out, nil
}

// newVariableStringFilter maps one validated clause to the generated string
// filter union used by native process-instance variable search.
func newVariableStringFilter(clause d.ProcessInstanceVariableFilterClause) (camundav89.StringFilterProperty, error) {
	var valueFilter camundav89.StringFilterProperty
	switch clause.Operator {
	case d.ProcessInstanceVariableFilterOperatorExists:
		if err := valueFilter.FromAdvancedStringFilter(camundav89.AdvancedStringFilter{Exists: clause.Exists}); err != nil {
			return camundav89.StringFilterProperty{}, err
		}
		return valueFilter, nil
	default:
		return camundav89.StringFilterProperty{}, fmt.Errorf("unsupported variable filter operator %q", clause.Operator)
	}
}
