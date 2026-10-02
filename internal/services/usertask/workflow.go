// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"

	"github.com/grafvonb/c8volt/internal/services"
	types "github.com/grafvonb/c8volt/typex"
)

// ResolveProcessInstanceKeys resolves each task to its process instance in input order and stops
// at the first lookup error.
func ResolveProcessInstanceKeys(ctx context.Context, api API, taskKeys types.Keys, opts ...services.CallOption) (types.Keys, error) {
	processInstanceKeys := make(types.Keys, 0, len(taskKeys))
	for _, taskKey := range taskKeys {
		task, err := api.GetUserTask(ctx, taskKey, opts...)
		if err != nil {
			return nil, err
		}
		processInstanceKeys = append(processInstanceKeys, task.ProcessInstanceKey)
	}
	return processInstanceKeys, nil
}
