// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	CheckReadAccess(ctx context.Context, opts ...services.CallOption) error
	CancelProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, opts ...services.CallOption) (d.BatchOperation, error)
	WaitForCompletion(ctx context.Context, batchOperationKey string, opts ...services.CallOption) (d.BatchOperation, error)
}

type GenBatchOperationClientCamunda interface {
	SearchBatchOperationsWithResponse(ctx context.Context, body camundav89.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchBatchOperationsResponse, error)
	GetBatchOperationWithResponse(ctx context.Context, batchOperationKey camundav89.BatchOperationKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetBatchOperationResponse, error)
	CancelProcessInstancesBatchOperationWithResponse(ctx context.Context, body camundav89.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstancesBatchOperationResponse, error)
}

var _ API = (*Service)(nil)
var _ GenBatchOperationClientCamunda = (*camundav89.ClientWithResponses)(nil)
