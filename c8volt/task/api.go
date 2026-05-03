// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	types "github.com/grafvonb/c8volt/typex"
)

type API interface {
	ResolveProcessInstanceKeyFromUserTask(ctx context.Context, taskKey string, opts ...options.FacadeOption) (string, error)
	ResolveProcessInstanceKeysFromUserTasks(ctx context.Context, taskKeys types.Keys, opts ...options.FacadeOption) (types.Keys, error)
}

var _ API = (*client)(nil)
