// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package variable

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API exposes process-root and exact-local variable lookup and update operations.
type API interface {
	// SearchScopeVariablesPage reads one page bound to an independent owner and local scope.
	SearchScopeVariablesPage(ctx context.Context, query d.ScopeVariableQuery, page d.ProcessInstanceVariablePageRequest, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error)
	// SearchScopeVariables returns all local observations or an error without partial results.
	SearchScopeVariables(ctx context.Context, query d.ScopeVariableQuery, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error)
	// SearchProcessInstanceVariablesPage reads one normalized root-variable page.
	SearchProcessInstanceVariablesPage(ctx context.Context, key string, page d.ProcessInstanceVariablePageRequest, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error)
	// SearchProcessInstanceVariables reads the complete root-variable collection.
	SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error)
	UpdateProcessInstanceVariables(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error)
	UpdateScopeVariables(ctx context.Context, scopeKey string, variables map[string]any, opts ...services.CallOption) (d.ScopeVariableUpdateResponse, error)
}
