// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	v87 "github.com/grafvonb/c8volt/internal/services/usertask/v87"
	"github.com/stretchr/testify/require"
)

func TestService_GetUserTask_ReturnsExplicitUnsupportedError(t *testing.T) {
	svc, err := v87.New(
		&config.Config{
			APIs: config.APIs{
				Camunda: config.API{
					BaseURL: "https://camunda.local/v2",
				},
			},
		},
		&http.Client{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	require.NoError(t, err)

	task, err := svc.GetUserTask(context.Background(), "2251799815391233")

	require.Empty(t, task)
	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrUnsupported))
	require.Contains(t, err.Error(), "has-user-tasks lookup is unsupported in Camunda 8.7")
	require.Contains(t, err.Error(), "requires Camunda 8.8 or 8.9")
}
