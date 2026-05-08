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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	return svc
}

func TestService_LookupJob_Unsupported(t *testing.T) {
	svc := newTestService(t)

	job, err := svc.LookupJob(context.Background(), "2251799813711967")

	require.Error(t, err)
	assert.Empty(t, job)
	assert.ErrorIs(t, err, domain.ErrUnsupported)
	assert.Contains(t, err.Error(), "job lookup")
	assert.Contains(t, err.Error(), "Camunda 8.8")
}

func TestService_UpdateJob_Unsupported(t *testing.T) {
	svc := newTestService(t)

	result, err := svc.UpdateJob(context.Background(), domain.JobUpdateRequest{Key: "2251799813711967"})

	require.Error(t, err)
	assert.Empty(t, result)
	assert.ErrorIs(t, err, domain.ErrUnsupported)
	assert.Contains(t, err.Error(), "job update")
	assert.Contains(t, err.Error(), "Camunda 8.8")
}
