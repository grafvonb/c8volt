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

func (c *client) ResolveProcessInstanceKeyFromUserTask(ctx context.Context, taskKey string, opts ...options.FacadeOption) (string, error) {
	task, err := c.utApi.GetUserTask(ctx, taskKey, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return "", ferr.FromDomain(err)
	}
	return task.ProcessInstanceKey, nil
}
