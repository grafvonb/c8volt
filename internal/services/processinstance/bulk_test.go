// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"errors"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type stubProcessInstanceCreator struct {
	create func(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error)
}

// CreateProcessInstance delegates creation to the configured test callback.
func (s stubProcessInstanceCreator) CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
	if s.create == nil {
		return d.ProcessInstanceCreation{}, errors.New("unexpected process-instance creation")
	}
	return s.create(ctx, data, opts...)
}

// TestCreateProcessInstancesPreservesOrderAndOptions verifies the service-owned create-many workflow remains sequential and option-aware.
func TestCreateProcessInstancesPreservesOrderAndOptions(t *testing.T) {
	seen := []string{}
	got, err := CreateProcessInstances(context.Background(), stubProcessInstanceCreator{
		create: func(_ context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			seen = append(seen, data.BpmnProcessId)
			return d.ProcessInstanceCreation{Key: "created-" + data.BpmnProcessId, BpmnProcessId: data.BpmnProcessId}, nil
		},
	}, []d.ProcessInstanceData{
		{BpmnProcessId: "alpha"},
		{BpmnProcessId: "beta"},
	}, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "beta"}, seen)
	require.Equal(t, []d.ProcessInstanceCreation{
		{Key: "created-alpha", BpmnProcessId: "alpha"},
		{Key: "created-beta", BpmnProcessId: "beta"},
	}, got)
}

// TestCreateProcessInstancesStopsOnFirstError verifies the previous fail-on-first-error behavior stays intact.
func TestCreateProcessInstancesStopsOnFirstError(t *testing.T) {
	seen := []string{}
	wantErr := errors.New("create failed")
	got, err := CreateProcessInstances(context.Background(), stubProcessInstanceCreator{
		create: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			seen = append(seen, data.BpmnProcessId)
			if data.BpmnProcessId == "beta" {
				return d.ProcessInstanceCreation{}, wantErr
			}
			return d.ProcessInstanceCreation{Key: "created-" + data.BpmnProcessId}, nil
		},
	}, []d.ProcessInstanceData{
		{BpmnProcessId: "alpha"},
		{BpmnProcessId: "beta"},
		{BpmnProcessId: "gamma"},
	})

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
	require.Equal(t, []string{"alpha", "beta"}, seen)
}
