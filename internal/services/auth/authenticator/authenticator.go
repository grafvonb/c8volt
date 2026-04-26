// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package authenticator

import (
	"context"
	"errors"
	"net/http"
)

var (
	ErrNoAuthServiceInContext      = errors.New("no auth service in context")
	ErrInvalidAuthServiceInContext = errors.New("invalid auth service in context")
)

type RequestEditor func(ctx context.Context, req *http.Request) error

type Authenticator interface {
	Init(ctx context.Context) error
	Editor() RequestEditor
	ClearCache()
	Name() string
}

type BearerProvider interface {
	Token(ctx context.Context, target string) (string, error)
}

type ctxKey struct{}

func ToContext(ctx context.Context, a Authenticator) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

func FromContext(ctx context.Context) (Authenticator, error) {
	v := ctx.Value(ctxKey{})
	if v == nil {
		return nil, ErrNoAuthServiceInContext
	}
	s, ok := v.(Authenticator)
	if !ok || s == nil {
		return nil, ErrInvalidAuthServiceInContext
	}
	return s, nil
}
