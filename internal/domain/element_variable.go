// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strings"
)

// ScopeVariableQuery binds a local variable read to independent owner and scope
// identities. ExpectedTenantId is optional observed evidence, not a discovery filter.
type ScopeVariableQuery struct {
	ProcessInstanceKey string
	ScopeKey           string
	ExpectedTenantId   *string
}

// Validate rejects incomplete identities before any variable request is made.
func (q ScopeVariableQuery) Validate() error {
	if strings.TrimSpace(q.ProcessInstanceKey) == "" {
		return fmt.Errorf("%w: scope variable process-instance key must not be blank", ErrValidation)
	}
	if strings.TrimSpace(q.ScopeKey) == "" {
		return fmt.Errorf("%w: scope variable scope key must not be blank", ErrValidation)
	}
	return nil
}
