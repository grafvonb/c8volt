// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package batchoperation

import (
	"context"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	batchsvc "github.com/grafvonb/c8volt/internal/services/batchoperation"
)

type client struct {
	api batchsvc.API
	log *slog.Logger
}

func New(api batchsvc.API, log *slog.Logger) API {
	return &client{api: api, log: log}
}

func (c *client) CheckBatchOperationReadAccess(ctx context.Context, opts ...options.FacadeOption) error {
	if err := c.api.CheckReadAccess(ctx, options.MapFacadeOptionsToCallOptions(opts)...); err != nil {
		return ferr.FromDomain(err)
	}
	return nil
}

func (c *client) CancelProcessInstancesBatch(ctx context.Context, filter process.ProcessInstanceFilter, opts ...options.FacadeOption) (BatchOperation, error) {
	op, err := c.api.CancelProcessInstances(ctx, toDomainProcessInstanceFilter(filter), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return BatchOperation{}, ferr.FromDomain(err)
	}
	return fromDomainBatchOperation(op), nil
}

func (c *client) WaitBatchOperation(ctx context.Context, key string, opts ...options.FacadeOption) (BatchOperation, error) {
	op, err := c.api.WaitForCompletion(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return fromDomainBatchOperation(op), ferr.FromDomain(err)
	}
	return fromDomainBatchOperation(op), nil
}

func toDomainProcessInstanceFilter(filter process.ProcessInstanceFilter) d.ProcessInstanceFilter {
	return d.ProcessInstanceFilter{
		Key:                  filter.Key,
		BpmnProcessId:        filter.BpmnProcessId,
		ProcessVersion:       filter.ProcessVersion,
		ProcessVersionTag:    filter.ProcessVersionTag,
		ProcessDefinitionKey: filter.ProcessDefinitionKey,
		StartDateAfter:       filter.StartDateAfter,
		StartDateBefore:      filter.StartDateBefore,
		EndDateAfter:         filter.EndDateAfter,
		EndDateBefore:        filter.EndDateBefore,
		State:                d.State(filter.State),
		ParentKey:            filter.ParentKey,
		HasParent:            filter.HasParent,
		HasIncident:          filter.HasIncident,
	}
}

func fromDomainBatchOperation(op d.BatchOperation) BatchOperation {
	return BatchOperation{
		Key:        op.Key,
		Type:       op.Type,
		State:      op.State,
		StatusCode: op.StatusCode,
		Status:     op.Status,
	}
}
