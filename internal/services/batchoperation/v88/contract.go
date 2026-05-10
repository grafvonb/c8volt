// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	CheckReadAccess(ctx context.Context, opts ...services.CallOption) error
	CancelProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, opts ...services.CallOption) (d.BatchOperation, error)
	WaitForCompletion(ctx context.Context, batchOperationKey string, opts ...services.CallOption) (d.BatchOperation, error)
}

type GenBatchOperationClientCamunda interface {
	SearchBatchOperationsWithResponse(ctx context.Context, body camundav88.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchBatchOperationsResponse, error)
	GetBatchOperationWithResponse(ctx context.Context, batchOperationKey camundav88.BatchOperationKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetBatchOperationResponse, error)
	CancelProcessInstancesBatchOperationWithResponse(ctx context.Context, body camundav88.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CancelProcessInstancesBatchOperationResponse, error)
}

var _ API = (*Service)(nil)
var _ GenBatchOperationClientCamunda = (*camundav88.ClientWithResponses)(nil)
