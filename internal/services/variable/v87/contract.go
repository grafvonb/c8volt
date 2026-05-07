// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"

	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
)

// GenVariableClientOperate captures the generated Operate calls used by the v8.7 variable service.
type GenVariableClientOperate interface {
	SearchVariablesForProcessInstancesWithResponse(ctx context.Context, body operatev87.SearchVariablesForProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchVariablesForProcessInstancesResponse, error)
}

var _ GenVariableClientOperate = (*operatev87.ClientWithResponses)(nil)
