// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unicode"

	"github.com/grafvonb/c8volt/config"
)

const diagnosticCorrelationIDLimit = 256

// diagnosticRequestMetadata contains only request fields approved for a diagnostic record.
type diagnosticRequestMetadata struct {
	method              string
	target              string
	host                string
	profile             string
	tenant              string
	clientRequestID     string
	clientCorrelationID string
}

// diagnosticResponseMetadata contains only response fields approved for a diagnostic record.
type diagnosticResponseMetadata struct {
	requestID     string
	correlationID string
	retryAfter    string
	serverTiming  string
}

// diagnosticSanitizer owns the private known-secret set used by one exchange.
type diagnosticSanitizer struct {
	mu      sync.Mutex
	secrets map[string]struct{}
	profile string
	tenant  string
}

// newDiagnosticSanitizer seeds configured credentials without retaining the configuration object.
func newDiagnosticSanitizer(cfg *config.Config) *diagnosticSanitizer {
	sanitizer := &diagnosticSanitizer{secrets: make(map[string]struct{})}
	if cfg == nil {
		return sanitizer
	}
	sanitizer.profile = cfg.ActiveProfile
	sanitizer.tenant = cfg.App.Tenant
	sanitizer.addSecret(cfg.Auth.OAuth2.ClientSecret)
	sanitizer.addSecret(cfg.Auth.Cookie.Password)
	return sanitizer
}

// sanitizeRequest collects request secrets before producing any retained request field.
func (sanitizer *diagnosticSanitizer) sanitizeRequest(req *http.Request) diagnosticRequestMetadata {
	if sanitizer == nil {
		sanitizer = newDiagnosticSanitizer(nil)
	}
	sanitizer.mu.Lock()
	defer sanitizer.mu.Unlock()

	if req == nil {
		return diagnosticRequestMetadata{
			profile: sanitizer.redactKnownSecrets(sanitizer.profile),
			tenant:  sanitizer.redactKnownSecrets(sanitizer.tenant),
		}
	}
	sanitizer.collectURLSecrets(req.URL)
	sanitizer.collectSensitiveHeaders(req.Header, false)

	metadata := diagnosticRequestMetadata{
		method:  sanitizeDiagnosticMethod(req.Method),
		profile: sanitizer.redactKnownSecrets(sanitizer.profile),
		tenant:  sanitizer.redactKnownSecrets(sanitizer.tenant),
	}
	metadata.target, metadata.host = sanitizer.sanitizeURL(req.URL)
	metadata.clientRequestID = sanitizer.firstSafeHeader(req.Header.Values("X-Request-ID"))
	metadata.clientCorrelationID = sanitizer.firstSafeHeader(req.Header.Values("X-Correlation-ID"))
	return metadata
}

// sanitizeResponse collects all response secrets before formatting any allowed response header.
func (sanitizer *diagnosticSanitizer) sanitizeResponse(resp *http.Response) diagnosticResponseMetadata {
	if sanitizer == nil {
		sanitizer = newDiagnosticSanitizer(nil)
	}
	sanitizer.mu.Lock()
	defer sanitizer.mu.Unlock()

	if resp == nil {
		return diagnosticResponseMetadata{}
	}
	sanitizer.collectSensitiveHeaders(resp.Header, true)

	requestValues := resp.Header.Values("X-Request-ID")
	if len(requestValues) == 0 {
		requestValues = resp.Header.Values("Request-ID")
	}
	serverTiming := sanitizeDiagnosticServerTiming(resp.Header.Values("Server-Timing"))
	return diagnosticResponseMetadata{
		requestID:     sanitizer.firstSafeHeader(requestValues),
		correlationID: sanitizer.firstSafeHeader(resp.Header.Values("X-Correlation-ID")),
		retryAfter:    sanitizeDiagnosticRetryAfter(resp.Header.Get("Retry-After")),
		serverTiming:  sanitizer.redactKnownSecrets(serverTiming),
	}
}

// collectURLSecrets adds userinfo and values carried by sensitive query parameters.
func (sanitizer *diagnosticSanitizer) collectURLSecrets(value *url.URL) {
	if value == nil {
		return
	}
	if value.User != nil {
		sanitizer.addSecret(value.User.Username())
		if password, ok := value.User.Password(); ok {
			sanitizer.addSecret(password)
		}
	}
	query, err := url.ParseQuery(value.RawQuery)
	if err != nil {
		return
	}
	for name, values := range query {
		if !isSensitiveDiagnosticName(name) {
			continue
		}
		for _, item := range values {
			sanitizer.addSecret(item)
		}
	}
}

// collectSensitiveHeaders adds credentials and cookie values without admitting arbitrary headers.
func (sanitizer *diagnosticSanitizer) collectSensitiveHeaders(header http.Header, response bool) {
	for name, values := range header {
		if !isSensitiveDiagnosticHeader(name) {
			continue
		}
		for _, value := range values {
			sanitizer.addSecret(value)
			for _, credential := range diagnosticCredentialParts(value) {
				sanitizer.addSecret(credential)
			}
		}
	}
	if response {
		response := &http.Response{Header: header}
		for _, cookie := range response.Cookies() {
			sanitizer.addSecret(cookie.Value)
		}
		return
	}
	request := &http.Request{Header: header}
	for _, cookie := range request.Cookies() {
		sanitizer.addSecret(cookie.Value)
	}
}

// diagnosticCredentialParts extracts structured credential values while keeping them private.
func diagnosticCredentialParts(value string) []string {
	parts := strings.Fields(value)
	if len(parts) == 2 && (strings.EqualFold(parts[0], "bearer") || strings.EqualFold(parts[0], "basic")) {
		return []string{parts[1]}
	}
	return nil
}

// sanitizeURL builds a safe identity from components and never serializes URL userinfo or fragments.
func (sanitizer *diagnosticSanitizer) sanitizeURL(value *url.URL) (string, string) {
	if value == nil || value.Opaque != "" {
		return "", ""
	}
	path := value.EscapedPath()
	if path == "" {
		path = "/"
	}
	path = sanitizer.redactKnownSecrets(path)
	host := sanitizer.redactKnownSecrets(value.Host)
	if containsUnsafeDiagnosticText(host) {
		host = ""
	}

	if value.RawQuery == "" {
		return path, host
	}
	query, err := url.ParseQuery(value.RawQuery)
	if err != nil {
		return path, host
	}
	safeQuery := make(url.Values)
	for name, values := range query {
		if isSensitiveDiagnosticName(name) || isPayloadDiagnosticName(name) {
			continue
		}
		if containsUnsafeDiagnosticText(name) || suspiciousDiagnosticValue(name) {
			continue
		}
		for _, item := range values {
			if containsUnsafeDiagnosticText(item) || suspiciousDiagnosticValue(item) || sanitizer.containsKnownSecret(item) {
				continue
			}
			safeQuery.Add(name, sanitizer.redactKnownSecrets(item))
		}
	}
	if encoded := safeQuery.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return path, host
}

// addSecret records decoded and encoded representations for matching before serialization.
func (sanitizer *diagnosticSanitizer) addSecret(secret string) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return
	}
	variants := []string{secret, url.QueryEscape(secret), url.PathEscape(secret)}
	if decoded, err := url.QueryUnescape(secret); err == nil {
		variants = append(variants, decoded)
	}
	for _, variant := range variants {
		if variant != "" {
			sanitizer.secrets[variant] = struct{}{}
		}
	}
}

// containsKnownSecret detects any configured or observed credential representation.
func (sanitizer *diagnosticSanitizer) containsKnownSecret(value string) bool {
	for secret := range sanitizer.secrets {
		if strings.Contains(value, secret) {
			return true
		}
	}
	return false
}

// redactKnownSecrets replaces known values before the result can reach record formatting.
func (sanitizer *diagnosticSanitizer) redactKnownSecrets(value string) string {
	secrets := make([]string, 0, len(sanitizer.secrets))
	for secret := range sanitizer.secrets {
		secrets = append(secrets, secret)
	}
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	for _, secret := range secrets {
		value = strings.ReplaceAll(value, secret, "[REDACTED]")
	}
	return value
}

// firstSafeHeader returns the first bounded identifier that cannot reflect known secrets.
func (sanitizer *diagnosticSanitizer) firstSafeHeader(values []string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !validDiagnosticCorrelationID(value) || sanitizer.containsKnownSecret(value) || suspiciousDiagnosticValue(value) {
			continue
		}
		return value
	}
	return ""
}

// sanitizeDiagnosticMethod admits only a single RFC token and leaves malformed methods unknown.
func sanitizeDiagnosticMethod(method string) string {
	if method == "" || !validDiagnosticToken(method) {
		return ""
	}
	return method
}

// validDiagnosticCorrelationID enforces the bounded, control-free identifier contract.
func validDiagnosticCorrelationID(value string) bool {
	if value == "" || len(value) > diagnosticCorrelationIDLimit {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// containsUnsafeDiagnosticText rejects control characters that could forge a log line.
func containsUnsafeDiagnosticText(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// isSensitiveDiagnosticHeader recognizes credential-bearing metadata case-insensitively.
func isSensitiveDiagnosticHeader(name string) bool {
	normalized := normalizeDiagnosticName(name)
	return normalized == "authorization" || normalized == "proxyauthorization" ||
		normalized == "cookie" || normalized == "setcookie" ||
		strings.Contains(normalized, "token") || strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "clientsecret") || strings.Contains(normalized, "password")
}

// isSensitiveDiagnosticName recognizes direct and signed-URL credential parameter families.
func isSensitiveDiagnosticName(name string) bool {
	normalized := normalizeDiagnosticName(name)
	if strings.HasPrefix(normalized, "xamz") || strings.HasPrefix(normalized, "xgoog") {
		return true
	}
	switch normalized {
	case "authorization", "auth", "token", "accesstoken", "refreshtoken", "idtoken",
		"password", "passwd", "clientsecret", "cookie", "apikey", "signature", "sig", "securitytoken", "credential":
		return true
	default:
		return false
	}
}

// isPayloadDiagnosticName omits query fields commonly used to carry business payloads.
func isPayloadDiagnosticName(name string) bool {
	switch normalizeDiagnosticName(name) {
	case "body", "data", "payload", "variable", "variables", "processvariables", "businesspayload":
		return true
	default:
		return false
	}
}

// normalizeDiagnosticName compares decoded names after removing conventional separators.
func normalizeDiagnosticName(name string) string {
	for range 3 {
		decoded, err := url.QueryUnescape(name)
		if err != nil || decoded == name {
			break
		}
		name = decoded
	}
	name = strings.ToLower(name)
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || r == '.' || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, name)
}

// suspiciousDiagnosticValue rejects nested credentials, payload objects and credential-shaped identifiers.
func suspiciousDiagnosticValue(value string) bool {
	decoded := value
	for range 3 {
		next, err := url.QueryUnescape(decoded)
		if err != nil || next == decoded {
			break
		}
		decoded = next
	}
	lower := strings.ToLower(decoded)
	for _, marker := range []string{
		"authorization=", "token=", "access_token", "refresh_token", "id_token", "password=", "passwd=",
		"client_secret", "api_key", "apikey=", "signature=", "x-amz-", "x-goog-", `"password"`, `"variables"`,
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return strings.HasPrefix(lower, "bearer ") || strings.HasPrefix(lower, "basic ")
}

// sanitizeDiagnosticRetryAfter retains canonical nonnegative seconds or an HTTP date.
func sanitizeDiagnosticRetryAfter(value string) string {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseUint(value, 10, 64); err == nil {
		return strconv.FormatUint(seconds, 10)
	}
	parsed, err := http.ParseTime(value)
	if err != nil {
		return ""
	}
	return parsed.UTC().Format(http.TimeFormat)
}

// sanitizeDiagnosticServerTiming retains metric tokens and finite nonnegative dur parameters only.
func sanitizeDiagnosticServerTiming(values []string) string {
	var safe []string
	for _, value := range values {
		for _, entry := range splitDiagnosticHeaderList(value) {
			parts := strings.Split(entry, ";")
			metric := strings.TrimSpace(parts[0])
			if !validDiagnosticToken(metric) {
				continue
			}
			duration := ""
			invalidDuration := false
			for _, parameter := range parts[1:] {
				name, raw, found := strings.Cut(strings.TrimSpace(parameter), "=")
				if !found || !strings.EqualFold(strings.TrimSpace(name), "dur") {
					continue
				}
				number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
				if err != nil || number < 0 || math.IsInf(number, 0) || math.IsNaN(number) {
					invalidDuration = true
					break
				}
				duration = strconv.FormatFloat(number, 'f', -1, 64)
			}
			if invalidDuration {
				continue
			}
			if duration != "" {
				metric += ";dur=" + duration
			}
			safe = append(safe, metric)
		}
	}
	return strings.Join(safe, ",")
}

// splitDiagnosticHeaderList separates comma-delimited values without trusting quoted descriptions.
func splitDiagnosticHeaderList(value string) []string {
	var entries []string
	start := 0
	quoted := false
	escaped := false
	for index, r := range value {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quoted {
			escaped = true
			continue
		}
		if r == '"' {
			quoted = !quoted
			continue
		}
		if r == ',' && !quoted {
			entries = append(entries, strings.TrimSpace(value[start:index]))
			start = index + 1
		}
	}
	entries = append(entries, strings.TrimSpace(value[start:]))
	return entries
}

// validDiagnosticToken applies the RFC token character set used by methods and metric names.
func validDiagnosticToken(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r > unicode.MaxASCII || !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("!#$%&'*+-.^_`|~", r)) {
			return false
		}
	}
	return true
}

// classifyDiagnosticError maps typed failure evidence to bounded fields without serializing error text.
func classifyDiagnosticError(err error, phase diagnosticPhase, closing bool) (diagnosticFailure, diagnosticReason) {
	if err == nil {
		return "", ""
	}
	if errors.Is(err, context.Canceled) {
		return diagnosticFailureCanceled, ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return diagnosticFailureTimeout, ""
	}
	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		return diagnosticFailureTimeout, ""
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return diagnosticFailureDNS, ""
	}
	if isDiagnosticCertificateError(err) {
		return diagnosticFailureTLS, diagnosticReasonCertificateInvalid
	}

	reason := diagnosticReason("")
	switch {
	case errors.Is(err, syscall.ECONNREFUSED):
		reason = diagnosticReasonConnectionRefused
	case errors.Is(err, syscall.ECONNRESET):
		reason = diagnosticReasonConnectionReset
	case errors.Is(err, io.ErrUnexpectedEOF):
		reason = diagnosticReasonUnexpectedEOF
	case phase == diagnosticPhaseBody && closing:
		reason = diagnosticReasonCloseFailed
	case phase == diagnosticPhaseBody:
		reason = diagnosticReasonReadFailed
	}

	switch phase {
	case diagnosticPhaseDNS:
		return diagnosticFailureDNS, reason
	case diagnosticPhaseConnect:
		return diagnosticFailureConnect, reason
	case diagnosticPhaseTLS:
		return diagnosticFailureTLS, reason
	case diagnosticPhaseBody:
		return diagnosticFailureBody, reason
	default:
		return diagnosticFailureTransport, reason
	}
}

// isDiagnosticCertificateError recognizes standard certificate validation error types.
func isDiagnosticCertificateError(err error) bool {
	var unknownAuthority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	return errors.As(err, &unknownAuthority) || errors.As(err, &hostname) || errors.As(err, &invalid)
}
