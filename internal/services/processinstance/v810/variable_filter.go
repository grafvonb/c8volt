// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"encoding/json"
	"fmt"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

// newVariableValueFiltersPtr builds native v8.10 variable filters from
// version-neutral clauses while preserving the command's all-clauses order.
func newVariableValueFiltersPtr(filters d.ProcessInstanceVariableFilterSet) (*[]camundav810.VariableValueFilterProperty, error) {
	if len(filters.Clauses) == 0 {
		return nil, nil
	}
	if err := filters.Validate(); err != nil {
		return nil, err
	}
	out := make([]camundav810.VariableValueFilterProperty, 0, len(filters.Clauses))
	for _, clause := range filters.Clauses {
		valueFilter, err := newVariableStringFilter(clause)
		if err != nil {
			return nil, err
		}
		out = append(out, camundav810.VariableValueFilterProperty{
			Name:  clause.Name,
			Value: valueFilter,
		})
	}
	return &out, nil
}

// newVariableStringFilter maps one validated clause to the generated string
// filter union used by native process-instance variable search.
func newVariableStringFilter(clause d.ProcessInstanceVariableFilterClause) (camundav810.StringFilterProperty, error) {
	var valueFilter camundav810.StringFilterProperty
	switch clause.Operator {
	case d.ProcessInstanceVariableFilterOperatorEq:
		if err := valueFilter.FromAdvancedStringFilter(camundav810.AdvancedStringFilter{Eq: &clause.Value}); err != nil {
			return camundav810.StringFilterProperty{}, err
		}
		return valueFilter, nil
	case d.ProcessInstanceVariableFilterOperatorNeq:
		if err := valueFilter.FromAdvancedStringFilter(camundav810.AdvancedStringFilter{Neq: &clause.Value}); err != nil {
			return camundav810.StringFilterProperty{}, err
		}
		return valueFilter, nil
	case d.ProcessInstanceVariableFilterOperatorIn:
		values, err := variableStringArrayValues(clause)
		if err != nil {
			return camundav810.StringFilterProperty{}, err
		}
		if err := valueFilter.FromAdvancedStringFilter(camundav810.AdvancedStringFilter{In: &values}); err != nil {
			return camundav810.StringFilterProperty{}, err
		}
		return valueFilter, nil
	case d.ProcessInstanceVariableFilterOperatorNotIn:
		values, err := variableStringArrayValues(clause)
		if err != nil {
			return camundav810.StringFilterProperty{}, err
		}
		if err := valueFilter.FromAdvancedStringFilter(camundav810.AdvancedStringFilter{NotIn: &values}); err != nil {
			return camundav810.StringFilterProperty{}, err
		}
		return valueFilter, nil
	case d.ProcessInstanceVariableFilterOperatorLike:
		if err := valueFilter.FromAdvancedStringFilter(camundav810.AdvancedStringFilter{Like: &clause.Value}); err != nil {
			return camundav810.StringFilterProperty{}, err
		}
		return valueFilter, nil
	case d.ProcessInstanceVariableFilterOperatorExists:
		if err := valueFilter.FromAdvancedStringFilter(camundav810.AdvancedStringFilter{Exists: clause.Exists}); err != nil {
			return camundav810.StringFilterProperty{}, err
		}
		return valueFilter, nil
	default:
		return camundav810.StringFilterProperty{}, fmt.Errorf("unsupported variable filter operator %q", clause.Operator)
	}
}

// variableStringArrayValues parses validated domain array text into the string
// slice shape required by the generated Camunda string filter.
func variableStringArrayValues(clause d.ProcessInstanceVariableFilterClause) ([]string, error) {
	var values []string
	if err := json.Unmarshal([]byte(clause.Value), &values); err != nil {
		return nil, fmt.Errorf("%s requires a JSON string array: %w", clause.Operator, err)
	}
	return values, nil
}
