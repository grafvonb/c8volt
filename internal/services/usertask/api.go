// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v810 "github.com/grafvonb/c8volt/internal/services/usertask/v810"
	v87 "github.com/grafvonb/c8volt/internal/services/usertask/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/usertask/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/usertask/v89"
)

// API exposes legacy resolver lookup plus native direct and paged user-task reads.
type API interface {
	GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
	GetNativeUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
	SearchUserTasksPage(ctx context.Context, query d.UserTaskSearchQuery, page d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
var _ API = (*v89.Service)(nil)
var _ API = (*v810.Service)(nil)
var _ API = (v87.API)(nil)
var _ API = (v88.API)(nil)
var _ API = (v89.API)(nil)
var _ API = (v810.API)(nil)
