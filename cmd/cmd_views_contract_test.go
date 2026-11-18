// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/stretchr/testify/require"
)

// TestBoundedCommandFailureReasons locks typed recognition, source precedence,
// class/exit stability and generic advice without trusting transport error text.
func TestBoundedCommandFailureReasons(t *testing.T) {
	unsafe := errors.New("POST https://private.invalid body-secret resource-key\nforged log")
	type failureCase struct {
		name string
		err  error
		want string
	}
	tests := []failureCase{
		{"refused", syscall.ECONNREFUSED, "gateway connection refused; check the configured host and port and that the service is running"},
		{"reset", syscall.ECONNRESET, "gateway connection interrupted; check the service and proxy"},
		{"unexpected EOF", io.ErrUnexpectedEOF, "gateway connection interrupted; check the service and proxy"},
		{"deadline", context.DeadlineExceeded, "gateway timed out; check service availability, network access and configured timeout"},
		{"network timeout", &net.DNSError{IsTimeout: true}, "gateway timed out; check service availability, network access and configured timeout"},
		{"DNS", &net.DNSError{Name: "private.invalid"}, "cannot resolve gateway host; check the configured hostname and DNS"},
		{"untrusted certificate", x509.UnknownAuthorityError{}, "gateway TLS certificate validation failed; check the certificate trust, hostname and expiry"},
		{"certificate hostname", x509.HostnameError{Certificate: &x509.Certificate{}, Host: "private.invalid"}, "gateway TLS certificate validation failed; check the certificate trust, hostname and expiry"},
		{"certificate expiry", x509.CertificateInvalidError{Cert: &x509.Certificate{}}, "gateway TLS certificate validation failed; check the certificate trust, hostname and expiry"},
		{"syntax", &json.SyntaxError{}, "invalid gateway response; check the configured URL and proxy routing"},
		{"response type", &json.UnmarshalTypeError{Type: reflect.TypeOf("")}, "invalid gateway response; check the configured URL and proxy routing"},
		{"auth refused normalized", ferrors.Normalize(httpc.AuthenticationFailure(syscall.ECONNREFUSED)), "authentication service connection refused; check the configured host and port and that the service is running"},
		{"auth only", httpc.AuthenticationFailure(unsafe), "authentication failed; check the authentication configuration; use --debug for diagnostic details"},
		{"canceled", context.Canceled, "local precondition"},
		{"canceled auth status", httpc.AuthenticationFailure(httpc.WithHTTPStatus(context.Canceled, 401)), "local precondition"},
		{"unknown", unsafe, "internal"},
		{"misleading text", errors.New("connection refused HTTP 401 timeout"), "internal"},
		{"class timeout", ferrors.ErrTimeout, "timeout"},
		{"class malformed", ferrors.ErrMalformedResponse, "malformed response"},
		{"class invalid", ferrors.ErrInvalidInput, "invalid input"},
		{"discarded bare refusal", ferrors.Normalize(syscall.ECONNREFUSED), "internal"},
		{"zero status", httpc.WithHTTPStatus(unsafe, 0), "internal"},
		{"malformed precedes cancellation", errors.Join(context.Canceled, &json.SyntaxError{}), "invalid gateway response; check the configured URL and proxy routing"},
		{"malformed precedes status", httpc.WithHTTPStatus(&json.SyntaxError{}, 503), "invalid gateway response; check the configured URL and proxy routing"},
		{"joined independent auth and refusal", errors.Join(syscall.ECONNREFUSED, httpc.AuthenticationFailure(unsafe)), "authentication service connection refused; check the configured host and port and that the service is running"},
		{"joined first status", errors.Join(httpc.WithHTTPStatus(unsafe, 403), httpc.WithHTTPStatus(unsafe, 401)), "gateway HTTP 403: access denied; check credentials, roles and API permissions"},
	}
	for _, tc := range tests {
		if strings.HasPrefix(tc.want, "gateway ") || strings.HasPrefix(tc.want, "cannot resolve gateway") || strings.HasPrefix(tc.want, "invalid gateway") {
			tests = append(tests, failureCase{tc.name + " auth normalized", ferrors.Normalize(httpc.AuthenticationFailure(tc.err)), strings.ReplaceAll(tc.want, "gateway", "authentication service")})
		}
	}
	for _, status := range []struct {
		code int
		hint string
	}{
		{400, "request rejected; check the selected profile and service configuration"},
		{401, "authentication rejected; check the selected profile and credentials"},
		{403, "access denied; check credentials, roles and API permissions"},
		{404, "endpoint not found; check the Camunda API base URL and proxy routing"},
		{429, "rate limit reached; retry later or reduce request rate"},
		{500, "service error; check service health and proxy logs"},
		{502, "service error; check service health and proxy logs"},
		{503, "service error; check service health and proxy logs"},
		{504, "service error; check service health and proxy logs"},
		{418, "request rejected; check the selected profile and service configuration"},
	} {
		for _, auth := range []bool{false, true} {
			target, hint := "gateway", status.hint
			err := httpc.WithHTTPStatus(unsafe, status.code)
			if auth {
				target = "authentication service"
				err = httpc.AuthenticationFailure(err)
				if status.code == 400 {
					hint = "authentication request rejected; check client credentials and scope configuration"
				}
				if status.code == 404 {
					hint = "authentication endpoint not found; check the authentication URL and proxy routing"
				}
			}
			tests = append(tests, failureCase{fmt.Sprintf("%s %d normalized", target, status.code), ferrors.Normalize(err), fmt.Sprintf("%s HTTP %d: %s", target, status.code, hint)})
		}
	}
	// A canceled admission creates real typed local evidence without dispatching I/O.
	svc, err := httpc.New(&config.Config{HTTP: config.HTTP{Timeout: "1s", RequestRate: 1, RequestRateReportInterval: "0s"}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unsent.invalid", nil)
	require.NoError(t, err)
	_, admission := svc.Client().Transport.RoundTrip(req)
	require.True(t, httpc.IsRequestRateWaitError(admission))
	tests = append(tests, failureCase{"admission precedes connection", errors.Join(admission, httpc.AuthenticationFailure(syscall.ECONNREFUSED)), "local precondition; request not sent: request-rate wait (other attempts may have been submitted)"})
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, err := range []error{tc.err, fmt.Errorf("operation: %w", tc.err), errors.Join(tc.err, unsafe)} {
				class, exit := ferrors.Classify(err), ferrors.ExitCode(err)
				require.Equal(t, tc.want, boundedCommandFailureReason(err))
				require.Equal(t, class, ferrors.Classify(err))
				require.Equal(t, exit, ferrors.ExitCode(err))
			}
		})
	}
}
