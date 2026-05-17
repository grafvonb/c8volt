// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/grafvonb/c8volt/config"
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/poller"
)

type Service struct {
	c   GenBatchOperationClientCamunda
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func WithClient(c GenBatchOperationClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	c, err := camundav88.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav88.WithHTTPClient(deps.HTTPClient),
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

func (s *Service) CheckReadAccess(ctx context.Context, opts ...services.CallOption) error {
	_ = services.ApplyCallOptions(opts)

	resp, err := s.c.SearchBatchOperationsWithResponse(ctx, batchOperationReadProbe())
	if err != nil {
		return err
	}
	return httpc.HttpStatusErr(resp.HTTPResponse, resp.Body)
}

func (s *Service) CancelProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, opts ...services.CallOption) (d.BatchOperation, error) {
	_ = services.ApplyCallOptions(opts)

	bodyFilter, err := s.processInstanceFilter(filter)
	if err != nil {
		return d.BatchOperation{}, err
	}
	body := camundav88.CancelProcessInstancesBatchOperationJSONRequestBody{
		Filter: bodyFilter,
	}
	resp, err := services.RetryCamundaMutation(ctx, s.log, "cancel pi batch", func(ctx context.Context) (*camundav88.CancelProcessInstancesBatchOperationResponse, *http.Response, []byte, error) {
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
		case camundav88.BatchOperationStateEnumCOMPLETED:
			if result.OperationsFailedCount > 0 {
				return poller.JobPollStatus{}, batchOperationFailedError(result)
			}
			return poller.JobPollStatus{Success: true, Message: fmt.Sprintf("batch operation %s completed (%d/%d completed, %d failed)", batchOperationKey, result.OperationsCompletedCount, result.OperationsTotalCount, result.OperationsFailedCount)}, nil
		case camundav88.BatchOperationStateEnumFAILED, camundav88.BatchOperationStateEnumCANCELED, camundav88.BatchOperationStateEnumPARTIALLYCOMPLETED:
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

func batchOperationFromPayload(payload *camundav88.BatchOperationResponse, statusCode int, status string) d.BatchOperation {
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

func batchOperationErrorMessages(errors []camundav88.BatchOperationError) []string {
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

func batchOperationFailedError(op d.BatchOperation) error {
	msg := fmt.Sprintf("batch operation %s completed with %d/%d failed item(s) (%d completed)", op.Key, op.OperationsFailedCount, op.OperationsTotalCount, op.OperationsCompletedCount)
	if len(op.Errors) > 0 {
		msg += ": " + strings.Join(op.Errors, "; ")
	}
	return errors.New(msg)
}

func (s *Service) processInstanceFilter(filter d.ProcessInstanceFilter) (camundav88.ProcessInstanceFilter, error) {
	tenantFilter, err := common.NewStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return camundav88.ProcessInstanceFilter{}, fmt.Errorf("building tenant filter: %w", err)
	}
	processDefinitionKeyFilter, err := common.NewProcessDefinitionKeyEqFilterPtr(filter.ProcessDefinitionKey)
	if err != nil {
		return camundav88.ProcessInstanceFilter{}, fmt.Errorf("building process-definition-key filter: %w", err)
	}
	stateFilter, err := common.NewProcessInstanceStateEqFilterPtr(string(filter.State))
	if err != nil {
		return camundav88.ProcessInstanceFilter{}, fmt.Errorf("building state filter: %w", err)
	}
	return camundav88.ProcessInstanceFilter{
		TenantId:             tenantFilter,
		ProcessDefinitionKey: processDefinitionKeyFilter,
		State:                stateFilter,
	}, nil
}

func batchOperationReadProbe() camundav88.SearchBatchOperationsJSONRequestBody {
	from := int32(0)
	limit := int32(1)
	page := camundav88.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return camundav88.SearchBatchOperationsJSONRequestBody{Page: &page}
}
