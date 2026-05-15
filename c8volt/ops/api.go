// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
)

type API interface {
	PurgeOrphanProcessInstances(ctx context.Context, request OrphanPurgeRequest, opts ...options.FacadeOption) (OrphanPurgeResult, error)
	ExecuteRetentionPolicy(ctx context.Context, request RetentionPolicyRequest, opts ...options.FacadeOption) (RetentionPolicyResult, error)
}

var _ API = (*client)(nil)
