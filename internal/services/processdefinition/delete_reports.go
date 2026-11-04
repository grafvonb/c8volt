// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"context"
	"log/slog"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/typex"
)

// DeleteProcessDefinitionReports retains a keyed report for every returned slot,
// including failed or unattempted work. The lower-level deletion API keeps its
// original response semantics for internal callers that inspect submission evidence.
func DeleteProcessDefinitionReports(ctx context.Context, api ResourceDeleteAPI, pdAPI API, piAPI pisvc.API, log *slog.Logger, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ResourceDeleteResponse, error) {
	selected := keys.Unique()
	responses, err := DeleteProcessDefinitions(ctx, api, pdAPI, piAPI, log, selected, wantedWorkers, opts...)
	for i := range responses {
		responses[i].Key = selected[i]
	}
	return responses, err
}
