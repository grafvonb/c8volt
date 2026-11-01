// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

// TestBusinessIDSearchValidation keeps search separate from creation constraints and checks unsupported-before-invalid precedence.
func TestBusinessIDSearchValidation(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810, "future"} {
		require.NoError(t, ValidateBusinessIDSearch(nil, version))
		for _, value := range []string{"", string([]byte{0xff}), "  ", "*,?零", strings.Repeat("x", 1000)} {
			err := ValidateBusinessIDSearch(&value, version)
			switch {
			case version != toolx.V810:
				require.ErrorIs(t, err, d.ErrUnsupported)
			case value == "" || value == string([]byte{0xff}):
				require.ErrorIs(t, err, d.ErrValidation)
			default:
				require.NoError(t, err)
			}
		}
	}
}
