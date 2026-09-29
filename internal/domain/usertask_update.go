// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// ScopeVariableUpdateResponse records whether a scope-local variable write was accepted.
// Confirmation belongs to the user-task update workflow and is intentionally absent here.
type ScopeVariableUpdateResponse struct {
	ScopeKey   string
	Accepted   bool
	StatusCode int
	Status     string
}
