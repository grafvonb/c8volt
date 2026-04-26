// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFilterPtr_ReturnsInitError(t *testing.T) {
	expectedErr := errors.New("boom")

	filter, err := newFilterPtr("value", func(_ *string, _ string) error {
		return expectedErr
	})

	require.Nil(t, filter)
	require.ErrorIs(t, err, expectedErr)
}
