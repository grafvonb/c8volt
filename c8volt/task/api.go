// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
)

type API interface {
	ResolveProcessInstanceKeyFromUserTask(ctx context.Context, taskKey string, opts ...options.FacadeOption) (string, error)
}

var _ API = (*client)(nil)
