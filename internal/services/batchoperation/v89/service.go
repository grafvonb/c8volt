// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
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
	c, err := camundav89.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav89.WithHTTPClient(deps.HTTPClient),
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
	resp, err := s.c.CancelProcessInstancesBatchOperationWithResponse(ctx, camundav89.CancelProcessInstancesBatchOperationJSONRequestBody{
		Filter: bodyFilter,
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
		result = d.BatchOperation{
			Key:        payload.BatchOperationKey,
			Type:       string(payload.BatchOperationType),
			State:      string(payload.State),
			StatusCode: resp.StatusCode(),
			Status:     resp.Status(),
		}
		switch payload.State {
		case camundav89.BatchOperationStateEnumCOMPLETED:
			return poller.JobPollStatus{Success: true, Message: fmt.Sprintf("batch operation %s completed", batchOperationKey)}, nil
		case camundav89.BatchOperationStateEnumFAILED, camundav89.BatchOperationStateEnumCANCELED, camundav89.BatchOperationStateEnumPARTIALLYCOMPLETED:
			return poller.JobPollStatus{}, fmt.Errorf("batch operation %s finished with state %s", batchOperationKey, payload.State)
		default:
			return poller.JobPollStatus{Success: false, Message: fmt.Sprintf("batch operation %s state %s", batchOperationKey, payload.State)}, nil
		}
	}
	if err := poller.WaitForCompletion(ctx, s.log, poller.DefaultCompletionTimeout, true, poll); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) processInstanceFilter(filter d.ProcessInstanceFilter) (camundav89.ProcessInstanceFilter, error) {
	tenantFilter, err := newStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return camundav89.ProcessInstanceFilter{}, fmt.Errorf("building tenant filter: %w", err)
	}
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(filter.ProcessDefinitionKey)
	if err != nil {
		return camundav89.ProcessInstanceFilter{}, fmt.Errorf("building process-definition-key filter: %w", err)
	}
	stateFilter, err := newProcessInstanceStateEqFilterPtr(string(filter.State))
	if err != nil {
		return camundav89.ProcessInstanceFilter{}, fmt.Errorf("building state filter: %w", err)
	}
	return camundav89.ProcessInstanceFilter{
		TenantId:             tenantFilter,
		ProcessDefinitionKey: processDefinitionKeyFilter,
		State:                stateFilter,
	}, nil
}

func newStringEqFilterPtr(v string) (*camundav89.StringFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	return newFilterPtr(v, (*camundav89.StringFilterProperty).FromStringFilterProperty0)
}

func newProcessDefinitionKeyEqFilterPtr(v string) (*camundav89.ProcessDefinitionKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	return newFilterPtr(v, (*camundav89.ProcessDefinitionKeyFilterProperty).FromProcessDefinitionKeyFilterProperty0)
}

func newProcessInstanceStateEqFilterPtr(v string) (*camundav89.ProcessInstanceStateFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	return newFilterPtr(v, func(f *camundav89.ProcessInstanceStateFilterProperty, s string) error {
		return f.FromProcessInstanceStateFilterProperty0(camundav89.ProcessInstanceStateEnum(s))
	})
}

func newFilterPtr[T any, D any](v D, init func(*T, D) error) (*T, error) {
	var f T
	if err := init(&f, v); err != nil {
		return nil, err
	}
	return &f, nil
}

func batchOperationReadProbe() camundav89.SearchBatchOperationsJSONRequestBody {
	from := int32(0)
	limit := int32(1)
	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return camundav89.SearchBatchOperationsJSONRequestBody{Page: &page}
}
