// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
)

// diagnosticFailure is a bounded transport outcome safe for diagnostic output.
type diagnosticFailure string

const (
	diagnosticFailureCanceled  diagnosticFailure = "CANCELED"
	diagnosticFailureTimeout   diagnosticFailure = "TIMEOUT"
	diagnosticFailureDNS       diagnosticFailure = "DNS_ERROR"
	diagnosticFailureConnect   diagnosticFailure = "CONNECT_ERROR"
	diagnosticFailureTLS       diagnosticFailure = "TLS_ERROR"
	diagnosticFailureBody      diagnosticFailure = "BODY_ERROR"
	diagnosticFailureTransport diagnosticFailure = "TRANSPORT_ERROR"
)

// diagnosticPhase identifies a failure phase only when observation supports it.
type diagnosticPhase string

const (
	diagnosticPhaseDNS             diagnosticPhase = "dns"
	diagnosticPhaseConnect         diagnosticPhase = "connect"
	diagnosticPhaseTLS             diagnosticPhase = "tls"
	diagnosticPhaseRequestWrite    diagnosticPhase = "request-write"
	diagnosticPhaseResponseHeaders diagnosticPhase = "response-headers"
	diagnosticPhaseBody            diagnosticPhase = "body"
)

// diagnosticReason is a bounded failure detail derived from typed evidence.
type diagnosticReason string

const (
	diagnosticReasonConnectionRefused  diagnosticReason = "connection-refused"
	diagnosticReasonConnectionReset    diagnosticReason = "connection-reset"
	diagnosticReasonUnexpectedEOF      diagnosticReason = "unexpected-eof"
	diagnosticReasonCertificateInvalid diagnosticReason = "certificate-invalid"
	diagnosticReasonReadFailed         diagnosticReason = "read-failed"
	diagnosticReasonCloseFailed        diagnosticReason = "close-failed"
)

// diagnosticConnection records whether a connection was newly established or reused.
type diagnosticConnection string

const (
	diagnosticConnectionNew    diagnosticConnection = "new"
	diagnosticConnectionReused diagnosticConnection = "reused"
)

// diagnosticSequence allocates positive exchange identifiers within one invocation.
type diagnosticSequence struct {
	value atomic.Uint64
}

// next returns an invocation-local identifier starting at one.
func (sequence *diagnosticSequence) next() uint64 {
	return sequence.value.Add(1)
}

// diagnosticRecord is the immutable, already-sanitized snapshot of one exchange.
type diagnosticRecord struct {
	sequence            uint64
	method              string
	target              string
	status              *int
	failure             diagnosticFailure
	total               time.Duration
	headers             *time.Duration
	body                *time.Duration
	phase               diagnosticPhase
	reason              diagnosticReason
	connection          diagnosticConnection
	dns                 []time.Duration
	tcp                 []time.Duration
	tls                 []time.Duration
	host                string
	profile             string
	tenant              string
	requestBytes        *int64
	requestComplete     *bool
	responseBytes       *int64
	responseComplete    *bool
	requestID           string
	correlationID       string
	clientRequestID     string
	clientCorrelationID string
	retryAfter          string
	serverTiming        string
}

// format validates and serializes the snapshot using the stable one-line field order.
func (record diagnosticRecord) format() (string, error) {
	if err := record.validate(); err != nil {
		return "", err
	}

	method := record.method
	if method == "" {
		method = "?"
	}
	target := record.target
	if target == "" {
		target = "?"
	}

	fields := make([]string, 0, 24)
	if record.status != nil {
		fields = append(fields, "status="+strconv.Itoa(*record.status))
	}
	if record.failure != "" {
		fields = append(fields, "error="+string(record.failure))
	}
	fields = append(fields, "total="+formatDiagnosticDuration(record.total))
	fields = appendOptionalDiagnosticDuration(fields, "headers", record.headers)
	fields = appendOptionalDiagnosticDuration(fields, "body", record.body)
	fields = appendDiagnosticText(fields, "phase", string(record.phase))
	fields = appendDiagnosticText(fields, "reason", string(record.reason))
	fields = appendDiagnosticText(fields, "conn", string(record.connection))
	fields = appendDiagnosticDurations(fields, "dns", record.dns)
	fields = appendDiagnosticDurations(fields, "tcp", record.tcp)
	fields = appendDiagnosticDurations(fields, "tls", record.tls)
	fields = appendDiagnosticText(fields, "host", record.host)
	fields = appendDiagnosticText(fields, "profile", record.profile)
	fields = appendDiagnosticText(fields, "tenant", record.tenant)
	fields = appendOptionalDiagnosticInt64(fields, "request-bytes", record.requestBytes)
	fields = appendOptionalDiagnosticBool(fields, "request-complete", record.requestComplete)
	fields = appendOptionalDiagnosticInt64(fields, "response-bytes", record.responseBytes)
	fields = appendOptionalDiagnosticBool(fields, "response-complete", record.responseComplete)
	fields = appendDiagnosticText(fields, "request-id", record.requestID)
	fields = appendDiagnosticText(fields, "correlation-id", record.correlationID)
	fields = appendDiagnosticText(fields, "client-request-id", record.clientRequestID)
	fields = appendDiagnosticText(fields, "client-correlation-id", record.clientCorrelationID)
	fields = appendDiagnosticText(fields, "retry-after", record.retryAfter)
	fields = appendDiagnosticText(fields, "server-timing", record.serverTiming)

	return fmt.Sprintf("api #%d %s %s: %s", record.sequence, quoteDiagnosticToken(method), quoteDiagnosticToken(target), strings.Join(fields, " ")), nil
}

// validate prevents impossible or fabricated numeric and enum evidence from reaching output.
func (record diagnosticRecord) validate() error {
	if record.sequence == 0 {
		return errors.New("diagnostic sequence must be positive")
	}
	if record.total < 0 {
		return errors.New("diagnostic total duration must be nonnegative")
	}
	if record.status != nil && *record.status <= 0 {
		return errors.New("diagnostic status must be positive")
	}
	for name, duration := range map[string]*time.Duration{"headers": record.headers, "body": record.body} {
		if duration != nil && *duration < 0 {
			return fmt.Errorf("diagnostic %s duration must be nonnegative", name)
		}
	}
	for name, durations := range map[string][]time.Duration{"dns": record.dns, "tcp": record.tcp, "tls": record.tls} {
		for _, duration := range durations {
			if duration < 0 {
				return fmt.Errorf("diagnostic %s duration must be nonnegative", name)
			}
		}
	}
	for name, count := range map[string]*int64{"request": record.requestBytes, "response": record.responseBytes} {
		if count != nil && *count < 0 {
			return fmt.Errorf("diagnostic %s byte count must be nonnegative", name)
		}
	}
	if !validDiagnosticFailure(record.failure) {
		return errors.New("invalid diagnostic failure")
	}
	if !validDiagnosticPhase(record.phase) {
		return errors.New("invalid diagnostic phase")
	}
	if !validDiagnosticReason(record.reason) {
		return errors.New("invalid diagnostic reason")
	}
	if !validDiagnosticConnection(record.connection) {
		return errors.New("invalid diagnostic connection")
	}
	return nil
}

// validDiagnosticFailure restricts failures to the contract's safe categories.
func validDiagnosticFailure(value diagnosticFailure) bool {
	switch value {
	case "", diagnosticFailureCanceled, diagnosticFailureTimeout, diagnosticFailureDNS,
		diagnosticFailureConnect, diagnosticFailureTLS, diagnosticFailureBody, diagnosticFailureTransport:
		return true
	default:
		return false
	}
}

// validDiagnosticPhase restricts phases to evidence locations defined by the contract.
func validDiagnosticPhase(value diagnosticPhase) bool {
	switch value {
	case "", diagnosticPhaseDNS, diagnosticPhaseConnect, diagnosticPhaseTLS, diagnosticPhaseRequestWrite,
		diagnosticPhaseResponseHeaders, diagnosticPhaseBody:
		return true
	default:
		return false
	}
}

// validDiagnosticReason restricts reasons to typed, bounded diagnostic details.
func validDiagnosticReason(value diagnosticReason) bool {
	switch value {
	case "", diagnosticReasonConnectionRefused, diagnosticReasonConnectionReset, diagnosticReasonUnexpectedEOF,
		diagnosticReasonCertificateInvalid, diagnosticReasonReadFailed, diagnosticReasonCloseFailed:
		return true
	default:
		return false
	}
}

// validDiagnosticConnection restricts reuse evidence to the two observable states.
func validDiagnosticConnection(value diagnosticConnection) bool {
	return value == "" || value == diagnosticConnectionNew || value == diagnosticConnectionReused
}

// appendOptionalDiagnosticDuration retains a duration only when observation supplied it.
func appendOptionalDiagnosticDuration(fields []string, name string, value *time.Duration) []string {
	if value == nil {
		return fields
	}
	return append(fields, name+"="+formatDiagnosticDuration(*value))
}

// appendDiagnosticDurations renders one sample as a scalar and multiple samples in start order.
func appendDiagnosticDurations(fields []string, name string, values []time.Duration) []string {
	if len(values) == 0 {
		return fields
	}
	formatted := make([]string, len(values))
	for index, value := range values {
		formatted[index] = formatDiagnosticDuration(value)
	}
	if len(formatted) == 1 {
		return append(fields, name+"="+formatted[0])
	}
	return append(fields, name+"=["+strings.Join(formatted, ",")+"]")
}

// appendDiagnosticText omits unavailable strings and quotes unsafe token content.
func appendDiagnosticText(fields []string, name, value string) []string {
	if value == "" {
		return fields
	}
	return append(fields, name+"="+quoteDiagnosticToken(value))
}

// appendOptionalDiagnosticInt64 distinguishes an observed zero count from no evidence.
func appendOptionalDiagnosticInt64(fields []string, name string, value *int64) []string {
	if value == nil {
		return fields
	}
	return append(fields, name+"="+strconv.FormatInt(*value, 10))
}

// appendOptionalDiagnosticBool distinguishes explicit incomplete evidence from no evidence.
func appendOptionalDiagnosticBool(fields []string, name string, value *bool) []string {
	if value == nil {
		return fields
	}
	return append(fields, name+"="+strconv.FormatBool(*value))
}

// formatDiagnosticDuration keeps Go's compact duration format but replaces its Unicode microsecond unit.
func formatDiagnosticDuration(value time.Duration) string {
	return strings.ReplaceAll(value.String(), "µs", "us")
}

// quoteDiagnosticToken applies JSON-compatible escaping only when the token grammar requires it.
func quoteDiagnosticToken(value string) string {
	if !diagnosticTokenNeedsQuoting(value) {
		return value
	}
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

// diagnosticTokenNeedsQuoting detects characters that could split or forge a record.
func diagnosticTokenNeedsQuoting(value string) bool {
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) || r == '"' || r == '\\' {
			return true
		}
	}
	return false
}
