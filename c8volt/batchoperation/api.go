// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package batchoperation

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
)

type API interface {
	CheckBatchOperationReadAccess(ctx context.Context, opts ...options.FacadeOption) error
	CancelProcessInstancesBatch(ctx context.Context, filter process.ProcessInstanceFilter, opts ...options.FacadeOption) (BatchOperation, error)
	WaitBatchOperation(ctx context.Context, key string, opts ...options.FacadeOption) (BatchOperation, error)
}

var _ API = (*client)(nil)
