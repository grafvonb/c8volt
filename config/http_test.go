// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHTTPRequestRateValidation accepts fractional allowances and rejects non-finite or negative rates.
func TestHTTPRequestRateValidation(t *testing.T) {
	t.Parallel()
	for _, value := range []float64{0, 0.5, 10, math.SmallestNonzeroFloat64, math.MaxFloat64} {
		h := HTTP{Timeout: "30s", RequestRate: value}
		require.NoError(t, h.Validate())
	}
	for _, value := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		h := HTTP{Timeout: "30s", RequestRate: value}
		require.ErrorContains(t, h.Validate(), "http.request_rate")
	}
	h := HTTP{}
	require.NoError(t, h.Normalize())
	require.Zero(t, h.RequestRate)
}
