// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

// ResolveProcessInstanceSelection mechanically maps service selection and preserves partial evidence with converted errors.
func (c *client) ResolveProcessInstanceSelection(ctx context.Context, request ProcessInstanceSelectionRequest, opts ...options.FacadeOption) (ProcessInstanceSelection, error) {
	result, err := pisvc.ResolveProcessInstanceSelection(ctx, c.piApi, c.incApi, d.ProcessInstanceSelectionRequest{SearchRequest: toDomainProcessInstanceSearchRequest(request.SearchRequest), Cardinality: d.ProcessInstanceSelectionCardinality(request.Cardinality)}, options.MapFacadeOptionsToCallOptions(opts)...)
	return fromDomainProcessInstanceSelection(result), ferrors.FromDomain(err)
}
