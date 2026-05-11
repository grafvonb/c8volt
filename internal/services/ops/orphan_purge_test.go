// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewCreatesOrphanPurgeServiceBoundary(t *testing.T) {
	t.Parallel()

	api := New(nil)

	require.NotNil(t, api)
	require.Implements(t, (*API)(nil), api)
}

func TestPurgeOrphanProcessInstancesPreservesRequestUntilWorkflowImplementation(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	request := d.OrphanPurgeRequest{
		CommandName: "ops purge orphan-process-instances",
		DryRun:      true,
		StartedAt:   started,
		Selection:   d.ProcessInstanceFilter{BpmnProcessId: "invoice"},
	}

	got, err := New(nil).PurgeOrphanProcessInstances(context.Background(), request)

	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrUnsupported))
	require.Equal(t, request, got.Request)
	require.Equal(t, request.CommandName, got.Report.CommandName)
	require.Equal(t, request.Selection, got.Report.SelectionFilters)
	require.Equal(t, d.OrphanPurgeOutcomeFailed, got.Outcome)
}
