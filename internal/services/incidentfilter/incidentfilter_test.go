// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incidentfilter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeStateAcceptsCanonicalValueCaseInsensitively(t *testing.T) {
	got, ok := NormalizeState(" RESOLVED ")

	require.True(t, ok)
	require.Equal(t, "resolved", got)
}

func TestNormalizeErrorTypeAcceptsCanonicalValueCaseInsensitively(t *testing.T) {
	got, ok := NormalizeErrorType(" job_no_retries ")

	require.True(t, ok)
	require.Equal(t, "JOB_NO_RETRIES", got)
}
