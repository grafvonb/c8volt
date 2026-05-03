// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package c8volt

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/cluster"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/c8volt/tenant"
)

type API interface {
	Capabilities(ctx context.Context) (Capabilities, error)
	process.API
	task.API
	cluster.API
	resource.API
	tenant.API
}

type Capabilities struct {
	CamundaVersion string
	Features       map[Feature]bool
}
type Feature string

func (c Capabilities) Has(f Feature) bool { return c.Features[f] }
