// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatValidationError(t *testing.T) {
	err := errors.Join(
		errors.Join(
			errors.New("auth.oauth2.token_url: token_url is required"),
			errors.New("auth.oauth2.client_id: client_id is required"),
		),
		errors.New("log.level: invalid value"),
	)

	got := FormatValidationError("configuration is invalid", err)

	require.EqualError(t, got, "configuration is invalid:\n- auth.oauth2.token_url: token_url is required\n- auth.oauth2.client_id: client_id is required\n- log.level: invalid value")
}
