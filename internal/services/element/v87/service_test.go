// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	return svc
}

func TestService_GetElement_Unsupported(t *testing.T) {
	svc := newTestService(t)

	element, err := svc.GetElement(context.Background(), "2251799813689002")

	require.ErrorIs(t, err, domain.ErrUnsupported)
	require.Empty(t, element)
	require.Contains(t, err.Error(), "element lookup")
	require.Contains(t, err.Error(), "Camunda 8.8")
}
