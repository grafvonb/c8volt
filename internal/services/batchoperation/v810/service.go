// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/poller"
)

// Service implements batch-operation workflows using the Camunda 8.10 unified client.
type Service struct {
	c   GenBatchOperationClientCamunda
	cfg *config.Config
	log *slog.Logger
}

// Option customizes a V810 batch-operation service for tests and callers.
type Option func(*Service)

// WithClient replaces the generated client while preserving a previously configured client for nil inputs.
func WithClient(c GenBatchOperationClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

// New creates a V810 batch-operation service with the unified Camunda client.
func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	c, err := camundav810.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav810.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{c: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.c)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

// CheckReadAccess uses a minimal search request as a non-mutating batch-operation access probe.
func (s *Service) CheckReadAccess(ctx context.Context, opts ...services.CallOption) error {
	_ = services.ApplyCallOptions(opts)

	resp, err := s.c.SearchBatchOperationsWithResponse(ctx, batchOperationReadProbe())
	if err != nil {
		return err
	}
	return httpc.HttpStatusErr(resp.HTTPResponse, resp.Body)
}

// CancelProcessInstances starts a V810 batch cancellation using the version-neutral process-instance filter.
func (s *Service) CancelProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, opts ...services.CallOption) (d.BatchOperation, error) {
	_ = services.ApplyCallOptions(opts)

	bodyFilter, err := s.processInstanceFilter(filter)
	if err != nil {
		return d.BatchOperation{}, err
	}
	body := camundav810.CancelProcessInstancesBatchOperationJSONRequestBody{
		Filter: bodyFilter,
	}
	resp, err := services.RetryCamundaMutation(ctx, s.log, "cancel pi batch", func(ctx context.Context) (*camundav810.CancelProcessInstancesBatchOperationResponse, *http.Response, []byte, error) {
		resp, err := s.c.CancelProcessInstancesBatchOperationWithResponse(ctx, body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		return d.BatchOperation{}, err
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.BatchOperation{}, err
	}
	if resp.JSON200 == nil {
		return d.BatchOperation{}, d.ErrMalformedResponse
	}
	return d.BatchOperation{
		Key:        resp.JSON200.BatchOperationKey,
		Type:       string(resp.JSON200.BatchOperationType),
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}, nil
}

// WaitForCompletion polls the V810 batch operation until the existing completion contract is satisfied.
func (s *Service) WaitForCompletion(ctx context.Context, batchOperationKey string, opts ...services.CallOption) (d.BatchOperation, error) {
	_ = services.ApplyCallOptions(opts)

	var result d.BatchOperation
	poll := func(ctx context.Context) (poller.JobPollStatus, error) {
		resp, err := s.c.GetBatchOperationWithResponse(ctx, batchOperationKey)
		if err != nil {
			return poller.JobPollStatus{}, err
		}
		payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
		if err != nil {
			if errors.Is(err, d.ErrNotFound) {
				return poller.JobPollStatus{Success: false, Message: fmt.Sprintf("batch operation %s not visible yet", batchOperationKey)}, nil
			}
			return poller.JobPollStatus{}, err
		}
		result = batchOperationFromPayload(payload, resp.StatusCode(), resp.Status())
		switch payload.State {
		case camundav810.BatchOperationStateEnumCOMPLETED:
			if result.OperationsFailedCount > 0 {
				return poller.JobPollStatus{}, batchOperationFailedError(result)
			}
			return poller.JobPollStatus{Success: true, Message: fmt.Sprintf("batch operation %s completed (%d/%d completed, %d failed)", batchOperationKey, result.OperationsCompletedCount, result.OperationsTotalCount, result.OperationsFailedCount)}, nil
		case camundav810.BatchOperationStateEnumFAILED, camundav810.BatchOperationStateEnumCANCELED, camundav810.BatchOperationStateEnumPARTIALLYCOMPLETED:
			if result.OperationsFailedCount > 0 {
				return poller.JobPollStatus{}, batchOperationFailedError(result)
			}
			return poller.JobPollStatus{}, fmt.Errorf("batch operation %s finished with state %s (%d/%d completed, %d failed)", batchOperationKey, payload.State, result.OperationsCompletedCount, result.OperationsTotalCount, result.OperationsFailedCount)
		default:
			return poller.JobPollStatus{Success: false, Message: fmt.Sprintf("batch operation %s state %s (%d/%d completed, %d failed)", batchOperationKey, payload.State, result.OperationsCompletedCount, result.OperationsTotalCount, result.OperationsFailedCount)}, nil
		}
	}
	if err := poller.WaitForCompletion(ctx, s.log, poller.DefaultCompletionTimeout, true, poll); err != nil {
		return result, err
	}
	return result, nil
}

// batchOperationFromPayload maps V810 generated batch-operation payloads into the version-neutral domain model.
func batchOperationFromPayload(payload *camundav810.BatchOperationResponse, statusCode int, status string) d.BatchOperation {
	if payload == nil {
		return d.BatchOperation{}
	}
	return d.BatchOperation{
		Key:                      payload.BatchOperationKey,
		Type:                     string(payload.BatchOperationType),
		State:                    string(payload.State),
		OperationsTotalCount:     payload.OperationsTotalCount,
		OperationsCompletedCount: payload.OperationsCompletedCount,
		OperationsFailedCount:    payload.OperationsFailedCount,
		Errors:                   batchOperationErrorMessages(payload.Errors),
		StatusCode:               statusCode,
		Status:                   status,
	}
}

// batchOperationErrorMessages formats generated error details without leaking generated types across the service boundary.
func batchOperationErrorMessages(errors []camundav810.BatchOperationError) []string {
	out := make([]string, 0, len(errors))
	for _, item := range errors {
		parts := make([]string, 0, 3)
		if item.PartitionId != 0 {
			parts = append(parts, fmt.Sprintf("partition %d", item.PartitionId))
		}
		if item.Type != "" {
			parts = append(parts, string(item.Type))
		}
		if item.Message != "" {
			parts = append(parts, item.Message)
		}
		if len(parts) > 0 {
			out = append(out, strings.Join(parts, ": "))
		}
	}
	return out
}

// batchOperationFailedError preserves the existing failed-item summary for completed batch operations.
func batchOperationFailedError(op d.BatchOperation) error {
	msg := fmt.Sprintf("batch operation %s completed with %d/%d failed item(s) (%d completed)", op.Key, op.OperationsFailedCount, op.OperationsTotalCount, op.OperationsCompletedCount)
	if len(op.Errors) > 0 {
		msg += ": " + strings.Join(op.Errors, "; ")
	}
	return errors.New(msg)
}

// processInstanceFilter converts the shared process-instance filter into V810 generated search-filter fields.
func (s *Service) processInstanceFilter(filter d.ProcessInstanceFilter) (camundav810.ProcessInstanceFilter, error) {
	tenantFilter, err := newStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return camundav810.ProcessInstanceFilter{}, fmt.Errorf("building tenant filter: %w", err)
	}
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(filter.ProcessDefinitionKey)
	if err != nil {
		return camundav810.ProcessInstanceFilter{}, fmt.Errorf("building process-definition-key filter: %w", err)
	}
	stateFilter, err := newProcessInstanceStateEqFilterPtr(string(filter.State))
	if err != nil {
		return camundav810.ProcessInstanceFilter{}, fmt.Errorf("building state filter: %w", err)
	}
	return camundav810.ProcessInstanceFilter{
		TenantId:             tenantFilter,
		ProcessDefinitionKey: processDefinitionKeyFilter,
		State:                stateFilter,
	}, nil
}

// newStringEqFilterPtr creates a generated equality filter only when a value is present.
func newStringEqFilterPtr(v string) (*camundav810.StringFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	return newFilterPtr(v, (*camundav810.StringFilterProperty).FromStringFilterProperty0)
}

// newProcessDefinitionKeyEqFilterPtr creates a generated process-definition-key equality filter.
func newProcessDefinitionKeyEqFilterPtr(v string) (*camundav810.ProcessDefinitionKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	return newFilterPtr(v, (*camundav810.ProcessDefinitionKeyFilterProperty).FromProcessDefinitionKeyFilterProperty0)
}

// newProcessInstanceStateEqFilterPtr creates a generated state equality filter from the domain state string.
func newProcessInstanceStateEqFilterPtr(v string) (*camundav810.ProcessInstanceStateFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	return newFilterPtr(v, func(f *camundav810.ProcessInstanceStateFilterProperty, s string) error {
		return f.FromProcessInstanceStateFilterProperty0(camundav810.ProcessInstanceStateEnum(s))
	})
}

// newFilterPtr centralizes oapi-codegen union initialization for simple generated filters.
func newFilterPtr[T any, D any](v D, init func(*T, D) error) (*T, error) {
	var f T
	if err := init(&f, v); err != nil {
		return nil, err
	}
	return &f, nil
}

// batchOperationReadProbe builds the smallest V810 batch-operation search request used for read checks.
func batchOperationReadProbe() camundav810.SearchBatchOperationsJSONRequestBody {
	from := int32(0)
	limit := int32(1)
	page := camundav810.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav810.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return camundav810.SearchBatchOperationsJSONRequestBody{Page: &page}
}
