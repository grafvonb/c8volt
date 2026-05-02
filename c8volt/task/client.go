// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	utsvc "github.com/grafvonb/c8volt/internal/services/usertask"
	types "github.com/grafvonb/c8volt/typex"
)

type client struct {
	pdApi pdsvc.API
	piApi pisvc.API
	utApi utsvc.API
	log   *slog.Logger
}

func New(pdApi pdsvc.API, piApi pisvc.API, utApi utsvc.API, log *slog.Logger) API {
	return &client{
		pdApi: pdApi,
		piApi: piApi,
		utApi: utApi,
		log:   log,
	}
}

// ResolveProcessInstanceKeyFromUserTask keeps single task-key lookup aligned with the multi-key path used by the CLI.
func (c *client) ResolveProcessInstanceKeyFromUserTask(ctx context.Context, taskKey string, opts ...options.FacadeOption) (string, error) {
	keys, err := c.ResolveProcessInstanceKeysFromUserTasks(ctx, types.Keys{taskKey}, opts...)
	if err != nil {
		return "", err
	}
	return keys[0], nil
}

// ResolveProcessInstanceKeysFromUserTasks resolves user tasks through the native task API and returns their owning process-instance keys in input order.
func (c *client) ResolveProcessInstanceKeysFromUserTasks(ctx context.Context, taskKeys types.Keys, opts ...options.FacadeOption) (types.Keys, error) {
	processInstanceKeys := make(types.Keys, 0, len(taskKeys))
	callOpts := options.MapFacadeOptionsToCallOptions(opts)
	for _, taskKey := range taskKeys {
		task, err := c.utApi.GetUserTask(ctx, taskKey, callOpts...)
		if err != nil {
			return nil, ferr.FromDomain(err)
		}
		processInstanceKeys = append(processInstanceKeys, task.ProcessInstanceKey)
	}
	return processInstanceKeys, nil
}
