// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API is the version-neutral batch-operation contract implemented by V810.
type API interface {
	CheckReadAccess(ctx context.Context, opts ...services.CallOption) error
	CancelProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, opts ...services.CallOption) (d.BatchOperation, error)
	WaitForCompletion(ctx context.Context, batchOperationKey string, opts ...services.CallOption) (d.BatchOperation, error)
}

// GenBatchOperationClientCamunda is the V810 unified generated-client subset used by this adapter.
type GenBatchOperationClientCamunda interface {
	SearchBatchOperationsWithResponse(ctx context.Context, body camundav810.SearchBatchOperationsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchBatchOperationsResponse, error)
	GetBatchOperationWithResponse(ctx context.Context, batchOperationKey camundav810.BatchOperationKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetBatchOperationResponse, error)
	CancelProcessInstancesBatchOperationWithResponse(ctx context.Context, body camundav810.CancelProcessInstancesBatchOperationJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstancesBatchOperationResponse, error)
}

var _ API = (*Service)(nil)
var _ GenBatchOperationClientCamunda = (*camundav810.ClientWithResponses)(nil)
