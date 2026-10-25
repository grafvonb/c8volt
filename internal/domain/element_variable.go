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

// ElementVariablePlannedValue describes one requested or observed
// local variable whose classification needs only one value.
type ElementVariablePlannedValue struct {
	Name         string
	Value        any
	APITruncated bool
}

// ElementVariablePlannedChange preserves the observed and requested
// values for one local variable that requires mutation.
type ElementVariablePlannedChange struct {
	Name         string
	Before       any
	After        any
	APITruncated bool
}

// ElementVariablePlan is the complete element-local classification and
// frozen payload for one selected element.
type ElementVariablePlan struct {
	ElementKey         string
	ProcessInstanceKey string
	ScopeKey           string
	TenantId           string
	Additions          []ElementVariablePlannedValue
	Changes            []ElementVariablePlannedChange
	UnchangedRequested []ElementVariablePlannedValue
	Untouched          []ElementVariablePlannedValue
	MutationVariables  map[string]any
	VerificationValues map[string]any
}

// ElementVariableUpdatePlan is the immutable-by-convention result of
// successful collection-wide discovery and classification.
type ElementVariableUpdatePlan struct {
	RequestedKeys          []string
	RequestedValues        map[string]any
	Elements               []ElementVariablePlan
	RequestedCount         int
	UpdateCount            int
	VariableAddCount       int
	VariableChangeCount    int
	VariableUnchangedCount int
	VariableUntouchedCount int
	MutationSubmitted      bool
	TenantContext          TenantContext
}
