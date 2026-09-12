// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsSanitizesAdversarialQueryFamilies verifies encoded credential families and payloads fail closed.
func TestAPIDiagnosticsSanitizesAdversarialQueryFamilies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawQuery string
		want     string
	}{
		{
			name:     "aws signed url",
			rawQuery: "safe=visible&X-Amz-Credential=credential&X-Amz-Signature=signature&X-Amz-Security-Token=token",
			want:     "/v2/items?safe=visible",
		},
		{
			name:     "google signed url",
			rawQuery: "X-Goog-Credential=credential&X-Goog-Signature=signature&resource-key=2251799813685249",
			want:     "/v2/items?resource-key=2251799813685249",
		},
		{
			name:     "multiply encoded names and values",
			rawQuery: "access%255Ftoken=drop&safe=token%253Dnested-secret&safe=retained",
			want:     "/v2/items?safe=retained",
		},
		{
			name:     "repeated safe and credential values",
			rawQuery: "tenant=customer-a&tenant=Bearer%20nested-secret&signature=drop&payload=%7B%22business%22%3Atrue%7D",
			want:     "/v2/items?tenant=customer-a",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			metadata := newDiagnosticSanitizer(nil).sanitizeRequest(&http.Request{
				Method: http.MethodGet,
				URL:    &url.URL{Scheme: "https", Host: "example.test", Path: "/v2/items", RawQuery: test.rawQuery},
			})
			require.Equal(t, test.want, metadata.target)
			require.NotContains(t, metadata.target, "credential")
			require.NotContains(t, metadata.target, "signature")
			require.NotContains(t, metadata.target, "nested-secret")
			require.NotContains(t, metadata.target, "business")
		})
	}
}

// TestAPIDiagnosticsRedactsKnownSecretsAcrossMetadata verifies private seeds are applied before any allowed field is retained.
func TestAPIDiagnosticsRedactsKnownSecretsAcrossMetadata(t *testing.T) {
	t.Parallel()

	const (
		clientSecret = "client/secret + value"
		cookieSecret = "cookie-response-secret"
	)
	cfg := config.New()
	cfg.ActiveProfile = "profile-" + url.QueryEscape(clientSecret)
	cfg.App.Tenant = "tenant-" + clientSecret
	cfg.Auth.OAuth2.ClientSecret = clientSecret

	sanitizer := newDiagnosticSanitizer(cfg)
	request := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Path: "/v2/items/" + url.PathEscape(clientSecret), RawQuery: "key=2251799813685249&echo=" + url.QueryEscape(clientSecret)},
		Header: make(http.Header),
		Body:   &panicReadCloser{},
	}
	request.Header.Add("X-Request-ID", url.QueryEscape(clientSecret))
	request.Header.Add("X-Request-ID", "safe-client-request")
	request.Header.Set("X-Correlation-ID", "safe-client-correlation")
	requestMetadata := sanitizer.sanitizeRequest(request)

	response := &http.Response{Header: make(http.Header), Body: &panicReadCloser{}}
	response.Header.Add("Set-Cookie", "SESSION="+cookieSecret+"; Path=/; HttpOnly")
	response.Header.Add("X-Request-ID", cookieSecret)
	response.Header.Add("X-Request-ID", "safe-response-request")
	response.Header.Set("X-Correlation-ID", "safe-response-correlation")
	response.Header.Set("X-Api-Key", "response-api-key")
	response.Header.Set("Server-Timing", `db;desc="response-api-key";dur=2`)
	responseMetadata := sanitizer.sanitizeResponse(response)

	serialized := strings.Join([]string{
		requestMetadata.target,
		requestMetadata.profile,
		requestMetadata.tenant,
		requestMetadata.clientRequestID,
		requestMetadata.clientCorrelationID,
		responseMetadata.requestID,
		responseMetadata.correlationID,
		responseMetadata.serverTiming,
	}, " ")
	for _, secret := range []string{clientSecret, url.QueryEscape(clientSecret), cookieSecret, "response-api-key"} {
		require.NotContains(t, serialized, secret)
	}
	require.Contains(t, requestMetadata.target, "key=2251799813685249")
	require.Equal(t, "safe-client-request", requestMetadata.clientRequestID)
	require.Equal(t, "safe-client-correlation", requestMetadata.clientCorrelationID)
	require.Equal(t, "safe-response-request", responseMetadata.requestID)
	require.Equal(t, "safe-response-correlation", responseMetadata.correlationID)
	require.Equal(t, "db;dur=2", responseMetadata.serverTiming)
}

// TestAPIDiagnosticsRejectsControlInjection verifies malformed identity and allowed metadata cannot forge another record.
func TestAPIDiagnosticsRejectsControlInjection(t *testing.T) {
	t.Parallel()

	request := &http.Request{
		Method: "GET\napi",
		URL:    &url.URL{Host: "example.test\rforged", Path: "/v2/items\napi #999", RawQuery: "safe=ok%0Aforged"},
		Header: http.Header{"X-Request-ID": {"safe\rforged"}},
	}
	metadata := newDiagnosticSanitizer(nil).sanitizeRequest(request)
	require.Empty(t, metadata.method)
	require.Empty(t, metadata.host)
	require.Empty(t, metadata.clientRequestID)
	require.NotContains(t, metadata.target, "\n")
	require.NotContains(t, metadata.target, "\r")
}

// FuzzAPIDiagnosticsSanitizer exercises arbitrary malformed metadata while guarding the no-body and one-line contracts.
func FuzzAPIDiagnosticsSanitizer(f *testing.F) {
	for _, seed := range []struct {
		query  string
		header string
	}{
		{query: "safe=value", header: "request-123"},
		{query: "X-Amz-Signature=secret&key=42", header: "safe\napi #999"},
		{query: "broken=%zz", header: strings.Repeat("a", diagnosticCorrelationIDLimit+1)},
		{query: "safe=token%253Dnested", header: "\x00\x1b[31m"},
	} {
		f.Add(seed.query, seed.header)
	}

	f.Fuzz(func(t *testing.T, rawQuery, headerValue string) {
		const knownSecret = "fuzz-known-secret"
		cfg := config.New()
		cfg.Auth.OAuth2.ClientSecret = knownSecret
		sanitizer := newDiagnosticSanitizer(cfg)
		request := &http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Host: "example.test", Path: "/v2/items", RawQuery: rawQuery},
			Header: http.Header{"Authorization": {"Bearer " + knownSecret}, "X-Request-ID": {headerValue}},
			Body:   &panicReadCloser{},
		}
		requestMetadata := sanitizer.sanitizeRequest(request)

		response := &http.Response{
			Header: http.Header{"Set-Cookie": {"SESSION=" + knownSecret + "; Path=/"}, "X-Correlation-ID": {headerValue}},
			Body:   &panicReadCloser{},
		}
		responseMetadata := sanitizer.sanitizeResponse(response)
		serialized := strings.Join([]string{
			requestMetadata.method, requestMetadata.target, requestMetadata.host, requestMetadata.profile,
			requestMetadata.tenant, requestMetadata.clientRequestID, requestMetadata.clientCorrelationID,
			responseMetadata.requestID, responseMetadata.correlationID, responseMetadata.retryAfter, responseMetadata.serverTiming,
		}, " ")
		require.NotContains(t, serialized, knownSecret)
		require.NotContains(t, serialized, "\n")
		require.NotContains(t, serialized, "\r")
	})
}

// TestAPIDiagnosticsSanitizesRequestIdentity verifies safe URL context survives while credentials and payloads do not.
func TestAPIDiagnosticsSanitizesRequestIdentity(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cfg.ActiveProfile = "prod secret-value"
	cfg.App.Tenant = "tenant-secret-value"
	cfg.Auth.OAuth2.ClientSecret = "secret-value"
	cfg.Auth.Cookie.Password = "cookie-password"

	reqURL, err := url.Parse("https://user:url-password@camunda.example.test:8443/v2/items/secret-value?tenant=customer-a&TOKEN=drop-me&access%5Ftoken=drop-too&safe=one&safe=two&variables=%7B%22password%22%3A%22payload%22%7D#fragment")
	require.NoError(t, err)
	req := &http.Request{Method: http.MethodGet, URL: reqURL, Header: make(http.Header), Body: &panicReadCloser{}}
	req.Header.Set("Authorization", "Bearer reflected-token")
	req.Header.Set("Cookie", "SESSION=session-secret")
	req.Header.Set("X-Request-ID", "client-request")
	req.Header.Set("X-Correlation-ID", "session-secret")

	sanitizer := newDiagnosticSanitizer(cfg)
	metadata := sanitizer.sanitizeRequest(req)

	require.Equal(t, http.MethodGet, metadata.method)
	require.Equal(t, "camunda.example.test:8443", metadata.host)
	require.Equal(t, "/v2/items/[REDACTED]?safe=one&safe=two&tenant=customer-a", metadata.target)
	require.Equal(t, "prod [REDACTED]", metadata.profile)
	require.Equal(t, "tenant-[REDACTED]", metadata.tenant)
	require.Equal(t, "client-request", metadata.clientRequestID)
	require.Empty(t, metadata.clientCorrelationID)
	serialized := strings.Join([]string{
		metadata.method, metadata.host, metadata.target, metadata.profile, metadata.tenant,
		metadata.clientRequestID, metadata.clientCorrelationID,
	}, " ")
	for _, secret := range []string{"user", "url-password", "secret-value", "cookie-password", "drop-me", "drop-too", "payload", "fragment", "reflected-token", "session-secret"} {
		require.NotContains(t, serialized, secret)
	}
}

// TestAPIDiagnosticsRejectsMalformedQuery verifies malformed query data is omitted instead of serialized raw.
func TestAPIDiagnosticsRejectsMalformedQuery(t *testing.T) {
	t.Parallel()

	req := &http.Request{Method: http.MethodGet, URL: &url.URL{Scheme: "https", Host: "example.test", Path: "/v2/items", RawQuery: "safe=visible&broken=%zz"}}
	metadata := newDiagnosticSanitizer(nil).sanitizeRequest(req)
	require.Equal(t, "/v2/items", metadata.target)
	require.NotContains(t, metadata.target, "visible")
	require.NotContains(t, metadata.target, "%zz")
}

// TestAPIDiagnosticsCollectsResponseSecretsBeforeAllowedHeaders verifies reflected cookies never survive correlation formatting.
func TestAPIDiagnosticsCollectsResponseSecretsBeforeAllowedHeaders(t *testing.T) {
	t.Parallel()

	sanitizer := newDiagnosticSanitizer(nil)
	req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/"}, Header: make(http.Header)}
	sanitizer.sanitizeRequest(req)
	response := &http.Response{Header: make(http.Header), Body: &panicReadCloser{}}
	response.Header.Add("Set-Cookie", "SESSION=response-secret; Path=/; HttpOnly")
	response.Header.Add("X-Request-ID", "response-secret")
	response.Header.Add("Request-ID", "fallback-request")
	response.Header.Add("X-Correlation-ID", "safe-correlation")
	response.Header.Add("X-Correlation-ID", "ignored-second")
	response.Header.Set("Retry-After", "120")
	response.Header.Set("Server-Timing", `cache;desc="response-secret";dur=2.5, db;dur=18;other=unsafe`)

	metadata := sanitizer.sanitizeResponse(response)
	require.Empty(t, metadata.requestID, "X-Request-ID takes precedence and must not fall back when unsafe")
	require.Equal(t, "safe-correlation", metadata.correlationID)
	require.Equal(t, "120", metadata.retryAfter)
	require.Equal(t, "cache;dur=2.5,db;dur=18", metadata.serverTiming)
	require.NotContains(t, metadata.serverTiming, "response-secret")
}

// TestAPIDiagnosticsValidatesAllowedHeaders verifies bounded identifiers and syntactic metadata parsing.
func TestAPIDiagnosticsValidatesAllowedHeaders(t *testing.T) {
	t.Parallel()

	response := &http.Response{Header: make(http.Header)}
	response.Header.Set("X-Request-ID", strings.Repeat("a", 257))
	response.Header.Set("X-Correlation-ID", "forged\napi #999")
	response.Header.Set("Retry-After", "-1")
	response.Header.Set("Server-Timing", "ok;dur=1.25, bad name;dur=3, nope;dur=payload")

	metadata := newDiagnosticSanitizer(nil).sanitizeResponse(response)
	require.Empty(t, metadata.requestID)
	require.Empty(t, metadata.correlationID)
	require.Empty(t, metadata.retryAfter)
	require.Equal(t, "ok;dur=1.25", metadata.serverTiming)

	response.Header.Set("Retry-After", "Sun, 06 Nov 1994 08:49:37 GMT")
	metadata = newDiagnosticSanitizer(nil).sanitizeResponse(response)
	require.Equal(t, "Sun, 06 Nov 1994 08:49:37 GMT", metadata.retryAfter)
}

// TestAPIDiagnosticsClassifiesTypedErrors verifies raw error strings are reduced to bounded outcomes and reasons.
func TestAPIDiagnosticsClassifiesTypedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		phase   diagnosticPhase
		closing bool
		failure diagnosticFailure
		reason  diagnosticReason
	}{
		{name: "canceled", err: context.Canceled, failure: diagnosticFailureCanceled},
		{name: "deadline", err: context.DeadlineExceeded, phase: diagnosticPhaseResponseHeaders, failure: diagnosticFailureTimeout},
		{name: "dns", err: &net.DNSError{Err: "seeded-secret", Name: "secret.example"}, failure: diagnosticFailureDNS},
		{name: "refused", err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, phase: diagnosticPhaseConnect, failure: diagnosticFailureConnect, reason: diagnosticReasonConnectionRefused},
		{name: "reset", err: syscall.ECONNRESET, phase: diagnosticPhaseBody, failure: diagnosticFailureBody, reason: diagnosticReasonConnectionReset},
		{name: "certificate", err: x509.UnknownAuthorityError{}, phase: diagnosticPhaseTLS, failure: diagnosticFailureTLS, reason: diagnosticReasonCertificateInvalid},
		{name: "unexpected eof", err: io.ErrUnexpectedEOF, phase: diagnosticPhaseBody, failure: diagnosticFailureBody, reason: diagnosticReasonUnexpectedEOF},
		{name: "read", err: errors.New("body contains secret"), phase: diagnosticPhaseBody, failure: diagnosticFailureBody, reason: diagnosticReasonReadFailed},
		{name: "close", err: errors.New("body contains secret"), phase: diagnosticPhaseBody, closing: true, failure: diagnosticFailureBody, reason: diagnosticReasonCloseFailed},
		{name: "transport", err: errors.New("authorization=secret"), failure: diagnosticFailureTransport},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			failure, reason := classifyDiagnosticError(test.err, test.phase, test.closing)
			require.Equal(t, test.failure, failure)
			require.Equal(t, test.reason, reason)
			require.NotContains(t, string(failure)+string(reason), "secret")
		})
	}
}

// TestAPIDiagnosticsSanitizerNeverReadsBodies guards the redaction boundary against payload inspection.
func TestAPIDiagnosticsSanitizerNeverReadsBodies(t *testing.T) {
	t.Parallel()

	req := &http.Request{Method: http.MethodPost, URL: &url.URL{Path: "/v2/items"}, Header: make(http.Header), Body: &panicReadCloser{}}
	response := &http.Response{Header: make(http.Header), Body: &panicReadCloser{}}
	sanitizer := newDiagnosticSanitizer(nil)
	require.NotPanics(t, func() { sanitizer.sanitizeRequest(req) })
	require.NotPanics(t, func() { sanitizer.sanitizeResponse(response) })
}

// panicReadCloser fails the test process if a sanitizer attempts payload access.
type panicReadCloser struct{}

// Read panics because request and response payloads are outside the diagnostic contract.
func (*panicReadCloser) Read([]byte) (int, error) { panic("diagnostic sanitizer read a body") }

// Close panics because sanitization must not alter body ownership.
func (*panicReadCloser) Close() error { panic("diagnostic sanitizer closed a body") }
