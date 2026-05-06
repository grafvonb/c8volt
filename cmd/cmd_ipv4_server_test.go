// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafvonb/c8volt/testx"
)

// newIPv4Server creates an IPv4-only test server for command tests that must avoid IPv6 listeners.
func newIPv4Server(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, handler)
}
