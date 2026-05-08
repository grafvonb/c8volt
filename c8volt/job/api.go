// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
)

type API interface {
	GetJob(ctx context.Context, key string, opts ...options.FacadeOption) (Job, error)
	UpdateJob(ctx context.Context, request UpdateRequest, opts ...options.FacadeOption) (UpdateResult, error)
}

var _ API = (*client)(nil)
