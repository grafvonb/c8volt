// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
)

const (
	piVariableFilterSourceExists = "--var-exists"
	piVariableFilterSourceVar    = "--var"
	piVariableFilterSourceLike   = "--var-like"
)

// parsePIVariableFilters normalizes all process-instance variable-search flag
// inputs into the public facade filter shape before backend clients are used.
func parsePIVariableFilters() (process.ProcessInstanceVariableFilterSet, error) {
	var clauses []process.ProcessInstanceVariableFilterClause
	for _, raw := range flagGetPIVarExists {
		parsed, err := parsePIVariableExistsFilter(raw)
		if err != nil {
			return process.ProcessInstanceVariableFilterSet{}, err
		}
		clauses = append(clauses, parsed...)
	}
	for _, raw := range flagGetPIVars {
		parsed, err := parsePIVariableValueFilter(raw, piVariableFilterSourceVar, process.ProcessInstanceVariableFilterOperatorEq)
		if err != nil {
			return process.ProcessInstanceVariableFilterSet{}, err
		}
		clauses = append(clauses, parsed...)
	}
	for _, raw := range flagGetPIVarLikes {
		parsed, err := parsePIVariableValueFilter(raw, piVariableFilterSourceLike, process.ProcessInstanceVariableFilterOperatorLike)
		if err != nil {
			return process.ProcessInstanceVariableFilterSet{}, err
		}
		clauses = append(clauses, parsed...)
	}
	return process.ProcessInstanceVariableFilterSet{Clauses: clauses}, nil
}

// hasPIVariableFilterFlags reports whether any future variable-search flag
// carries input. It intentionally keys off globals so parser unit tests can
// exercise the foundational plumbing before Cobra registration lands.
func hasPIVariableFilterFlags() bool {
	return len(flagGetPIVarExists) > 0 || len(flagGetPIVars) > 0 || len(flagGetPIVarLikes) > 0
}

// parsePIVariableExistsFilter expands comma-separated variable names into
// $exists=true clauses while rejecting blank names from doubled delimiters.
func parsePIVariableExistsFilter(raw string) ([]process.ProcessInstanceVariableFilterClause, error) {
	parts, err := splitPIVariableClauses(raw)
	if err != nil {
		return nil, invalidFlagValuef("invalid value for %s: %v", piVariableFilterSourceExists, err)
	}
	clauses := make([]process.ProcessInstanceVariableFilterClause, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, invalidFlagValuef("invalid value for %s: variable name must not be blank", piVariableFilterSourceExists)
		}
		clauses = append(clauses, process.ProcessInstanceVariableFilterClause{
			Name:     name,
			Operator: process.ProcessInstanceVariableFilterOperatorExists,
			Exists:   boolValuePtr(true),
			Source:   piVariableFilterSourceExists,
		})
	}
	return clauses, nil
}

// parsePIVariableValueFilter parses equality, like, and advanced operator
// clauses using the same comma-splitting rules for repeated flag values.
func parsePIVariableValueFilter(raw, source string, defaultOperator process.ProcessInstanceVariableFilterOperator) ([]process.ProcessInstanceVariableFilterClause, error) {
	parts, err := splitPIVariableClauses(raw)
	if err != nil {
		return nil, invalidFlagValuef("invalid value for %s: %v", source, err)
	}
	clauses := make([]process.ProcessInstanceVariableFilterClause, 0, len(parts))
	for _, part := range parts {
		clause, err := parsePIVariableValueClause(part, source, defaultOperator)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause)
	}
	return clauses, nil
}

// parsePIVariableValueClause converts one name=value or name.$operator=value
// expression without interpreting the serialized variable value.
func parsePIVariableValueClause(raw, source string, defaultOperator process.ProcessInstanceVariableFilterOperator) (process.ProcessInstanceVariableFilterClause, error) {
	left, value, ok := splitPIVariableClauseAssignment(raw)
	if !ok {
		return process.ProcessInstanceVariableFilterClause{}, invalidFlagValuef("invalid value for %s: %q must use name=value syntax", source, raw)
	}
	name, operator, err := parsePIVariableClauseTarget(left, defaultOperator)
	if err != nil {
		return process.ProcessInstanceVariableFilterClause{}, invalidFlagValuef("invalid value for %s: %v", source, err)
	}
	clause := process.ProcessInstanceVariableFilterClause{
		Name:     name,
		Operator: operator,
		Value:    strings.TrimSpace(value),
		Source:   source,
	}
	if err := validatePIVariableValueClause(&clause); err != nil {
		return process.ProcessInstanceVariableFilterClause{}, invalidFlagValuef("invalid value for %s: %v", source, err)
	}
	return clause, nil
}

// parsePIVariableClauseTarget separates the variable name from an optional
// advanced native operator suffix and normalizes accepted aliases.
func parsePIVariableClauseTarget(raw string, defaultOperator process.ProcessInstanceVariableFilterOperator) (string, process.ProcessInstanceVariableFilterOperator, error) {
	left := strings.TrimSpace(raw)
	if left == "" {
		return "", "", fmt.Errorf("variable name must not be blank")
	}
	idx := strings.LastIndex(left, ".$")
	if idx < 0 {
		return left, defaultOperator, nil
	}
	name := strings.TrimSpace(left[:idx])
	operator := normalizePIVariableOperator(left[idx+1:])
	if name == "" {
		return "", "", fmt.Errorf("variable name must not be blank")
	}
	if !isSupportedPIVariableOperator(operator) {
		return "", "", fmt.Errorf("unsupported variable operator %q", left[idx+1:])
	}
	return name, operator, nil
}

// validatePIVariableValueClause enforces local shape checks while preserving
// serialized values for the native search request builder.
func validatePIVariableValueClause(clause *process.ProcessInstanceVariableFilterClause) error {
	switch clause.Operator {
	case process.ProcessInstanceVariableFilterOperatorExists:
		switch strings.ToLower(strings.TrimSpace(clause.Value)) {
		case "true":
			clause.Value = ""
			clause.Exists = boolValuePtr(true)
		case "false":
			clause.Value = ""
			clause.Exists = boolValuePtr(false)
		default:
			return fmt.Errorf("%s requires true or false", clause.Operator)
		}
	case process.ProcessInstanceVariableFilterOperatorEq,
		process.ProcessInstanceVariableFilterOperatorNeq,
		process.ProcessInstanceVariableFilterOperatorLike:
		if clause.Value == "" {
			return fmt.Errorf("%s requires a non-empty value", clause.Operator)
		}
	case process.ProcessInstanceVariableFilterOperatorIn,
		process.ProcessInstanceVariableFilterOperatorNotIn:
		if err := validatePIVariableStringArrayValue(clause.Value); err != nil {
			return fmt.Errorf("%s requires an array value: %w", clause.Operator, err)
		}
	default:
		return fmt.Errorf("unsupported variable operator %q", clause.Operator)
	}
	return nil
}

// splitPIVariableClauses splits top-level commas while keeping commas inside
// quoted strings and bracketed JSON-shaped values intact.
func splitPIVariableClauses(raw string) ([]string, error) {
	var parts []string
	var b strings.Builder
	var quote rune
	escaped := false
	bracketDepth := 0
	for _, r := range raw {
		if quote != 0 {
			b.WriteRune(r)
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
			b.WriteRune(r)
		case '[':
			bracketDepth++
			b.WriteRune(r)
		case ']':
			if bracketDepth == 0 {
				return nil, fmt.Errorf("unexpected closing bracket")
			}
			bracketDepth--
			b.WriteRune(r)
		case ',':
			if bracketDepth == 0 {
				parts = append(parts, strings.TrimSpace(b.String()))
				b.Reset()
				continue
			}
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted value")
	}
	if bracketDepth != 0 {
		return nil, fmt.Errorf("unterminated array value")
	}
	parts = append(parts, strings.TrimSpace(b.String()))
	return parts, nil
}

// splitPIVariableClauseAssignment finds the first top-level equals sign so
// serialized values can contain equals signs without being truncated.
func splitPIVariableClauseAssignment(raw string) (string, string, bool) {
	var quote rune
	escaped := false
	bracketDepth := 0
	for i, r := range raw {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '=':
			if bracketDepth == 0 {
				return raw[:i], raw[i+1:], true
			}
		}
	}
	return "", "", false
}

// normalizePIVariableOperator canonicalizes accepted aliases before service-facing mapping.
func normalizePIVariableOperator(raw string) process.ProcessInstanceVariableFilterOperator {
	if raw == "$notin" {
		return process.ProcessInstanceVariableFilterOperatorNotIn
	}
	return process.ProcessInstanceVariableFilterOperator(raw)
}

// isSupportedPIVariableOperator keeps parser validation aligned with the native operator set.
func isSupportedPIVariableOperator(operator process.ProcessInstanceVariableFilterOperator) bool {
	switch operator {
	case process.ProcessInstanceVariableFilterOperatorEq,
		process.ProcessInstanceVariableFilterOperatorNeq,
		process.ProcessInstanceVariableFilterOperatorExists,
		process.ProcessInstanceVariableFilterOperatorIn,
		process.ProcessInstanceVariableFilterOperatorNotIn,
		process.ProcessInstanceVariableFilterOperatorLike:
		return true
	default:
		return false
	}
}

// isPIVariableArrayValue checks only shape so serialized array values are not rewritten by the CLI.
func isPIVariableArrayValue(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) >= 2 && value[0] == '[' && value[len(value)-1] == ']'
}

// validatePIVariableStringArrayValue catches malformed native string-array
// filters locally while leaving the original serialized array text untouched.
func validatePIVariableStringArrayValue(value string) error {
	if !isPIVariableArrayValue(value) {
		return fmt.Errorf("must be a JSON array")
	}
	var values []string
	if err := json.Unmarshal([]byte(value), &values); err != nil {
		return err
	}
	return nil
}

// boolValuePtr creates optional exists values for variable filter clauses.
func boolValuePtr(v bool) *bool {
	return &v
}
