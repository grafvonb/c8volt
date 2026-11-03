// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	types "github.com/grafvonb/c8volt/typex"
)

// DeleteProcessDefinitions maps service-owned ordered reports without reselecting keys.
func (c *client) DeleteProcessDefinitions(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DeleteReports, error) {
	responses, err := pdsvc.DeleteProcessDefinitionReports(ctx, c.api, c.pdApi, c.piApi, c.log, keys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromResourceDeleteResponses(responses), ferr.FromDomain(err)
}
