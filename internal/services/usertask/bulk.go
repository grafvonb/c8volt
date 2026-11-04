// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"errors"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// GetUserTasks reads stable-unique task keys concurrently while preserving
// first-input order and treating any individual read failure as a bulk failure.
func GetUserTasks(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.UserTask, error) {
	uniqueKeys := keys.Unique()
	if len(uniqueKeys) == 0 {
		return make([]d.UserTask, 0), nil
	}

	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(uniqueKeys), wantedWorkers, cfg.NoWorkerLimit)
	tasks, err := pool.ExecuteSlice[string, d.UserTask](ctx, uniqueKeys, workers, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.UserTask, error) {
		return api.GetNativeUserTask(ctx, key, opts...)
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil && !errors.Is(err, ctxErr) {
			err = errors.Join(err, ctxErr)
		}
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}
