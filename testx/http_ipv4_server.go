// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func NewIPv4Server(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()

	listener := newIPv4Listener(t)
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)

	return srv
}

func NewIPv4TLSServer(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()

	listener := newIPv4Listener(t)
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = listener
	srv.StartTLS()
	t.Cleanup(srv.Close)

	return srv
}

func newIPv4Listener(t testing.TB) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local listener unavailable in this environment: %v", err)
	}
	return listener
}
