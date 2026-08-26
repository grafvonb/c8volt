// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incidentfilter

import (
	"strings"

	"github.com/grafvonb/c8volt/toolx"
)

var validErrorTypes = []string{
	"AD_HOC_SUB_PROCESS_NO_RETRIES",
	"CALLED_DECISION_ERROR",
	"CALLED_ELEMENT_ERROR",
	"CONDITION_ERROR",
	"DECISION_EVALUATION_ERROR",
	"EXECUTION_LISTENER_NO_RETRIES",
	"EXTRACT_VALUE_ERROR",
	"FORM_NOT_FOUND",
	"IO_MAPPING_ERROR",
	"JOB_NO_RETRIES",
	"MESSAGE_SIZE_EXCEEDED",
	"RESOURCE_NOT_FOUND",
	"SECRET_RESOLUTION_ERROR",
	"TASK_LISTENER_NO_RETRIES",
	"UNHANDLED_ERROR_EVENT",
	"UNKNOWN",
	"UNSPECIFIED",
}

var validStates = []string{
	"active",
	"pending",
	"resolved",
	"migrated",
	"unknown",
	"all",
}

// ValidErrorTypes returns the version-neutral canonical incident error types.
func ValidErrorTypes() []string {
	out := make([]string, len(validErrorTypes))
	copy(out, validErrorTypes)
	return out
}

// ValidErrorTypesString renders valid incident error types for validation errors.
func ValidErrorTypesString() string {
	return strings.Join(validErrorTypes, ", ")
}

// ValidStatesString renders valid incident states for validation errors.
func ValidStatesString() string {
	return strings.Join(validStates, ", ")
}

// NormalizeState returns the canonical incident state or false for unknown values.
func NormalizeState(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", true
	}
	return toolx.CanonicalEnumString(value, validStates)
}

// NormalizeErrorType returns the canonical incident error type or false for unknown values.
func NormalizeErrorType(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", true
	}
	return toolx.CanonicalEnumString(value, validErrorTypes)
}

// ErrorTypeMatches compares a wanted filter value to a backend incident error type.
func ErrorTypeMatches(want string, got string) bool {
	normalized, ok := NormalizeErrorType(want)
	if !ok || normalized == "" {
		return ok
	}
	return strings.EqualFold(strings.TrimSpace(got), normalized)
}

// ErrorMessageContains reports whether got contains a case-insensitive message fragment.
func ErrorMessageContains(want string, got string) bool {
	needle := strings.TrimSpace(want)
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(got), strings.ToLower(needle))
}
