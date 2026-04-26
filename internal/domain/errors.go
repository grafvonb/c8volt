// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
)

var (
	ErrBadRequest        = errors.New("bad request")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrPrecondition      = errors.New("precondition failed")
	ErrUnsupported       = errors.New("unsupported media type")
	ErrValidation        = errors.New("validation failed")
	ErrRateLimited       = errors.New("rate limited")
	ErrGatewayTimeout    = errors.New("gateway timeout")
	ErrUnavailable       = errors.New("service unavailable")
	ErrUpstream          = errors.New("upstream error")
	ErrInternal          = errors.New("internal error")
	ErrMalformedResponse = errors.New("malformed response")
)
