// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	v87 "github.com/grafvonb/c8volt/internal/services/usertask/v87"
	"github.com/stretchr/testify/require"
)

// TestService_SearchUserTaskEffectiveVariablesPage_ReturnsUnsupportedWithoutRequest proves V87 rejects effective-variable reads before transport use.
func TestService_SearchUserTaskEffectiveVariablesPage_ReturnsUnsupportedWithoutRequest(t *testing.T) {
	var requests atomic.Int32
	client := &http.Client{Transport: variableRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, nil
	})}
	svc, err := v87.New(&config.Config{APIs: config.APIs{Camunda: config.API{BaseURL: "https://camunda.local/v2"}}}, client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	page, err := svc.SearchUserTaskEffectiveVariablesPage(context.Background(), "2251799815391233", d.UserTaskVariablePageRequest{Size: 10})

	require.Empty(t, page)
	require.ErrorIs(t, err, d.ErrUnsupported)
	require.Contains(t, err.Error(), "effective variables are unsupported in Camunda 8.7")
	require.Zero(t, requests.Load())
}

// variableRoundTripperFunc gives the V87 variable test an observable transport.
type variableRoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper for the request-counting test transport.
func (f variableRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
