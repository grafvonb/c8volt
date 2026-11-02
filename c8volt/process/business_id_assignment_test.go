// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	v88 "github.com/grafvonb/c8volt/internal/services/processinstance/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/processinstance/v89"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// facadeAssignmentAPI explicitly models the new methods rather than relying on embedded nil behavior.
type facadeAssignmentAPI struct {
	pisvc.API
	read   func(context.Context, string) (d.ProcessInstance, error)
	submit func(context.Context, string, *string, ...services.CallOption) (d.BusinessIDAssignmentResult, error)
}

// ValidateBusinessIDAssignment checks required direct intent in the isolated facade dependency.
func (a facadeAssignmentAPI) ValidateBusinessIDAssignment(value *string, opts ...services.CallOption) error {
	if value == nil {
		return d.ErrValidation
	}
	return d.ValidateBusinessIDAssignment(value)
}

// GetProcessInstance returns owned preview facts.
func (a facadeAssignmentAPI) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	return a.read(ctx, key)
}

// AssignProcessInstanceBusinessID exposes submitted arguments and returns partial evidence.
func (a facadeAssignmentAPI) AssignProcessInstanceBusinessID(ctx context.Context, key string, value *string, opts ...services.CallOption) (d.BusinessIDAssignmentResult, error) {
	return a.submit(ctx, key, value, opts...)
}

// TestBusinessIDAssignmentFacade preserves options, independent pointer storage and partial results with converted errors.
func TestBusinessIDAssignmentFacade(t *testing.T) {
	t.Parallel()
	keys := typex.Keys{"2251799813685249", "2251799813685250"}
	value := "  Order,001  "
	prior := "prior"
	later := "later"
	code := 503
	var plan BusinessIDAssignmentPlan
	calls := 0
	api := facadeAssignmentAPI{read: func(ctx context.Context, key string) (d.ProcessInstance, error) {
		return d.ProcessInstance{Key: key, BusinessId: &prior, State: d.StateActive}, nil
	}, submit: func(ctx context.Context, key string, bid *string, opts ...services.CallOption) (d.BusinessIDAssignmentResult, error) {
		calls++
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.NoWait)
		require.True(t, cfg.NoWorkerLimit)
		require.True(t, cfg.Verbose)
		require.Equal(t, "  Order,001  ", *bid)
		// Caller mutation during execution must not change the already copied payload or membership.
		*plan.RequestedBusinessID = "changed plan"
		plan.RequestedKeys[1] = "invalid"
		plan.Items[1].Key = "invalid"
		*plan.Items[1].PreviouslyObservedBusinessID = "changed prior"
		if calls == 1 {
			return d.BusinessIDAssignmentResult{Key: key, Status: "unknown", Confirmation: "skipped", HTTPStatus: &code, ObservedBusinessID: &later, HasObservation: true, Error: "machine diagnostic"}, d.ErrUnavailable
		}
		require.Equal(t, "2251799813685250", key)
		return d.BusinessIDAssignmentResult{Key: key, Status: "accepted", Confirmation: "skipped"}, nil
	}}
	facade := New(nil, api, nil, nil)
	var err error
	plan, err = facade.PlanProcessInstanceBusinessIDAssignments(context.Background(), keys, &value)
	require.NoError(t, err)
	value = "changed input"
	keys[0] = "invalid"
	prior = "changed source"
	require.Equal(t, "  Order,001  ", *plan.RequestedBusinessID)
	require.Equal(t, "prior", *plan.Items[0].PreviouslyObservedBusinessID)
	out, err := facade.ExecuteProcessInstanceBusinessIDAssignments(context.Background(), plan, 1, options.WithNoWait(), options.WithNoWorkerLimit(), options.WithVerbose())
	require.ErrorIs(t, err, ferr.ErrUnavailable)
	require.Equal(t, 2, calls)
	require.Len(t, out.Items, 2)
	require.Equal(t, "unknown", out.Items[0].Status)
	require.Equal(t, "machine diagnostic", out.Items[0].Error)
	require.Equal(t, "prior", *out.Items[1].PreviouslyObservedBusinessID)
	later = "changed"
	code = 0
	require.Equal(t, "later", *out.Items[0].ObservedBusinessID)
	require.Equal(t, 503, *out.Items[0].HTTPStatus)
	require.Equal(t, "  Order,001  ", *out.Items[1].RequestedBusinessID)
}

// TestBusinessIDAssignmentFacadeGuards exercises public validation, empty scopes and both unsupported adapters.
func TestBusinessIDAssignmentFacadeGuards(t *testing.T) {
	t.Parallel()
	api := facadeAssignmentAPI{read: func(context.Context, string) (d.ProcessInstance, error) {
		t.Fatal("unexpected read")
		return d.ProcessInstance{}, nil
	}, submit: func(context.Context, string, *string, ...services.CallOption) (d.BusinessIDAssignmentResult, error) {
		t.Fatal("unexpected write")
		return d.BusinessIDAssignmentResult{}, nil
	}}
	facade := New(nil, api, nil, nil)
	for _, value := range []*string{nil, new(""), new("  ")} {
		_, err := facade.PlanProcessInstanceBusinessIDAssignments(context.Background(), nil, value)
		require.ErrorIs(t, err, ferr.ErrInvalidInput)
	}
	plan, err := facade.PlanProcessInstanceBusinessIDAssignments(context.Background(), nil, new("x"))
	require.NoError(t, err)
	out, err := facade.ExecuteProcessInstanceBusinessIDAssignments(context.Background(), plan, 1)
	require.NoError(t, err)
	require.NotNil(t, out.Items)
	require.Empty(t, out.Items)
	for _, old := range []pisvc.API{&v88.Service{}, &v89.Service{}} {
		facade := New(nil, old, nil, nil)
		_, err := facade.PlanProcessInstanceBusinessIDAssignments(context.Background(), nil, new("x"))
		require.ErrorIs(t, err, ferr.ErrUnsupported)
		_, err = facade.ExecuteProcessInstanceBusinessIDAssignments(context.Background(), plan, 1)
		require.ErrorIs(t, err, ferr.ErrUnsupported)
	}
}
