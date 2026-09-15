// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// SearchUserTaskEffectiveVariablesPage rejects native effective-variable reads because Camunda 8.7 does not expose the required API contract.
func (s *Service) SearchUserTaskEffectiveVariablesPage(context.Context, string, d.UserTaskVariablePageRequest, ...services.CallOption) (d.UserTaskVariablePage, error) {
	return d.UserTaskVariablePage{}, fmt.Errorf("%w: user-task effective variables are unsupported in Camunda 8.7; requires Camunda 8.8 or newer", d.ErrUnsupported)
}
