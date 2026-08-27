// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incidentfilter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidErrorTypesReturnsVersionNeutralCanonicalValues proves incident
// filtering owns its accepted error-type set without generated enum imports.
func TestValidErrorTypesReturnsVersionNeutralCanonicalValues(t *testing.T) {
	got := ValidErrorTypes()

	require.Equal(t, []string{
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
	}, got)
}

// TestValidErrorTypesReturnsCopy verifies callers cannot mutate the package
// canonical error-type list used by later validations.
func TestValidErrorTypesReturnsCopy(t *testing.T) {
	got := ValidErrorTypes()
	got[0] = "BROKEN"

	next := ValidErrorTypes()
	require.Equal(t, "AD_HOC_SUB_PROCESS_NO_RETRIES", next[0])
}

// TestNormalizeStateUsesVersionNeutralCanonicalValues verifies state matching
// remains case-insensitive while preserving the lower-case CLI contract.
func TestNormalizeStateAcceptsCanonicalValueCaseInsensitively(t *testing.T) {
	got, ok := NormalizeState(" RESOLVED ")

	require.True(t, ok)
	require.Equal(t, "resolved", got)
}

// TestNormalizeErrorTypeUsesVersionNeutralCanonicalValues verifies error-type
// matching is independent from generated enum packages.
func TestNormalizeErrorTypeAcceptsCanonicalValueCaseInsensitively(t *testing.T) {
	got, ok := NormalizeErrorType(" secret_resolution_error ")

	require.True(t, ok)
	require.Equal(t, "SECRET_RESOLUTION_ERROR", got)
}

// TestNormalizeIncidentFiltersRejectUnknownValues keeps invalid CLI and service
// filters from being silently passed through as generated enum strings.
func TestNormalizeIncidentFiltersRejectUnknownValues(t *testing.T) {
	state, stateOK := NormalizeState("paused")
	errorType, errorTypeOK := NormalizeErrorType("BROKEN_ERROR")

	require.False(t, stateOK)
	require.Empty(t, state)
	require.False(t, errorTypeOK)
	require.Empty(t, errorType)
}
