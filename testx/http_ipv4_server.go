// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// NewIPv4Server starts an HTTP test server on IPv4 loopback for command tests that cannot use bracketed IPv6 URLs.
func NewIPv4Server(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()

	listener := newIPv4Listener(t)
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)

	return srv
}

// NewIPv4TLSServer starts a TLS test server on IPv4 loopback for client paths that need stable localhost URLs.
func NewIPv4TLSServer(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()

	listener := newIPv4Listener(t)
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = listener
	srv.StartTLS()
	t.Cleanup(srv.Close)

	return srv
}

// newIPv4Listener isolates environments without IPv4 loopback by skipping the dependent integration-style tests.
func newIPv4Listener(t testing.TB) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local listener unavailable in this environment: %v", err)
	}
	return listener
}
