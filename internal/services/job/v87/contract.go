// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	LookupJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error)
	UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error)
}

var _ API = (*Service)(nil)
