// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ProcessInstanceVariableUpdateStatus string

const (
	ProcessInstanceVariableUpdateStatusSubmitted          ProcessInstanceVariableUpdateStatus = "submitted"
	ProcessInstanceVariableUpdateStatusConfirmed          ProcessInstanceVariableUpdateStatus = "confirmed"
	ProcessInstanceVariableUpdateStatusMutationFailed     ProcessInstanceVariableUpdateStatus = "mutation_failed"
	ProcessInstanceVariableUpdateStatusConfirmationFailed ProcessInstanceVariableUpdateStatus = "confirmation_failed"
)

type ProcessInstanceVariableUpdateResult struct {
	Key                string
	Status             ProcessInstanceVariableUpdateStatus
	MutationAccepted   bool
	ConfirmationStatus string
	StatusCode         int
	Message            string
	Error              string
	Variables          map[string]any
}

type ProcessInstanceVariableUpdateResults struct {
	Items []ProcessInstanceVariableUpdateResult
}
