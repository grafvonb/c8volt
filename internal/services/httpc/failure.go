// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"encoding/json"
	"errors"
)

// statusError retains HTTP status evidence without changing existing error text.
type statusError struct {
	error
	status int
}

func (e *statusError) Unwrap() error { return e.error }

// WithHTTPStatus annotates an existing failure without exposing response details.
func WithHTTPStatus(err error, status int) error {
	if err == nil {
		return nil
	}
	return &statusError{error: err, status: status}
}

type authenticationError struct{ error }

func (e *authenticationError) Unwrap() error { return e.error }

// AuthenticationFailure identifies failures before an authenticated API request.
func AuthenticationFailure(err error) error {
	if err == nil {
		return nil
	}
	return &authenticationError{error: err}
}

// HasFailureDetails identifies annotations that facade conversion must retain.
func HasFailureDetails(err error) bool {
	var status *statusError
	var authentication *authenticationError
	return errors.As(err, &status) || errors.As(err, &authentication)
}

// FailureDetails returns only bounded facts derived from error types, never text.
func FailureDetails(err error) (failure, reason string, status int, authentication bool) {
	var statusFailure *statusError
	if errors.As(err, &statusFailure) {
		status = statusFailure.status
	}
	var authFailure *authenticationError
	authentication = errors.As(err, &authFailure)
	var syntax *json.SyntaxError
	var valueType *json.UnmarshalTypeError
	if errors.As(err, &syntax) || errors.As(err, &valueType) {
		return "INVALID_RESPONSE", "", status, authentication
	}
	f, r := classifyDiagnosticError(err, typedDiagnosticFailurePhase(err), false)
	return string(f), string(r), status, authentication
}
