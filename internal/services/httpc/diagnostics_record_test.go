// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsRecordFormatting verifies the complete stable message contract.
func TestAPIDiagnosticsRecordFormatting(t *testing.T) {
	t.Parallel()

	status := 200
	requestBytes := int64(12)
	requestComplete := true
	responseBytes := int64(34)
	responseComplete := false
	record := diagnosticRecord{
		sequence:            45,
		method:              "GET",
		target:              "/v2/process-instances/search?tenant=customer-a",
		status:              &status,
		failure:             diagnosticFailureTimeout,
		total:               2240 * time.Millisecond,
		headers:             durationPointer(2100 * time.Millisecond),
		body:                durationPointer(140 * time.Millisecond),
		phase:               diagnosticPhaseBody,
		reason:              diagnosticReasonReadFailed,
		connection:          diagnosticConnectionReused,
		dns:                 []time.Duration{8 * time.Millisecond},
		tcp:                 []time.Duration{12 * time.Millisecond, 24 * time.Millisecond},
		tls:                 []time.Duration{61 * time.Millisecond},
		host:                "camunda.example.com:443",
		profile:             "production",
		tenant:              "customer-a",
		requestBytes:        &requestBytes,
		requestComplete:     &requestComplete,
		responseBytes:       &responseBytes,
		responseComplete:    &responseComplete,
		requestID:           "request-123",
		correlationID:       "correlation-123",
		clientRequestID:     "client-request-123",
		clientCorrelationID: "client-correlation-123",
		retryAfter:          "120",
		serverTiming:        "cache;dur=2.5,db;dur=18",
	}

	got, err := record.format()
	require.NoError(t, err)
	require.Equal(t, "api #45 GET /v2/process-instances/search?tenant=customer-a: status=200 error=TIMEOUT total=2.24s headers=2.1s body=140ms phase=body reason=read-failed conn=reused dns=8ms tcp=[12ms,24ms] tls=61ms host=camunda.example.com:443 profile=production tenant=customer-a request-bytes=12 request-complete=true response-bytes=34 response-complete=false request-id=request-123 correlation-id=correlation-123 client-request-id=client-request-123 client-correlation-id=client-correlation-123 retry-after=120 server-timing=cache;dur=2.5,db;dur=18", got)
}

// TestAPIDiagnosticsRecordSequence verifies identifiers stay positive and unique within a concurrent invocation.
func TestAPIDiagnosticsRecordSequence(t *testing.T) {
	t.Parallel()

	const exchanges = 100
	var sequence diagnosticSequence
	values := make(chan uint64, exchanges)
	var workers sync.WaitGroup
	workers.Add(exchanges)
	for range exchanges {
		go func() {
			defer workers.Done()
			values <- sequence.next()
		}()
	}
	workers.Wait()
	close(values)

	seen := make(map[uint64]struct{}, exchanges)
	for value := range values {
		require.Positive(t, value)
		_, duplicate := seen[value]
		require.False(t, duplicate)
		seen[value] = struct{}{}
	}
	require.Len(t, seen, exchanges)
	for expected := uint64(1); expected <= exchanges; expected++ {
		require.Contains(t, seen, expected)
	}
}

// TestAPIDiagnosticsRecordOmissionAndUnknownIdentity verifies absent evidence is not fabricated.
func TestAPIDiagnosticsRecordOmissionAndUnknownIdentity(t *testing.T) {
	t.Parallel()

	record := diagnosticRecord{sequence: 1}
	got, err := record.format()
	require.NoError(t, err)
	require.Equal(t, "api #1 ? ?: total=0s", got)
	for _, absent := range []string{
		"status=", "error=", "headers=", "body=", "phase=", "reason=", "conn=",
		"dns=", "tcp=", "tls=", "host=", "profile=", "tenant=", "request-bytes=",
		"request-complete=", "response-bytes=", "response-complete=", "request-id=",
		"correlation-id=", "client-request-id=", "client-correlation-id=", "retry-after=",
		"server-timing=",
	} {
		require.NotContains(t, got, absent)
	}
}

// TestAPIDiagnosticsRecordEscaping verifies unsafe tokens remain on one physical line.
func TestAPIDiagnosticsRecordEscaping(t *testing.T) {
	t.Parallel()

	record := diagnosticRecord{
		sequence: 2,
		method:   "PO ST",
		target:   "/v2/items/\"quoted\"\nnext\\part",
		total:    250 * time.Microsecond,
		profile:  "prod\tblue",
	}
	got, err := record.format()
	require.NoError(t, err)
	require.Equal(t, "api #2 \"PO ST\" \"/v2/items/\\\"quoted\\\"\\nnext\\\\part\": total=250us profile=\"prod\\tblue\"", got)
	require.Equal(t, 1, strings.Count(got, "api #"))
	require.NotContains(t, got, "\n")
	require.NotContains(t, got, "µ")
}

// TestAPIDiagnosticsRecordRejectsInvalidNumbers verifies negative evidence and zero sequences cannot be serialized.
func TestAPIDiagnosticsRecordRejectsInvalidNumbers(t *testing.T) {
	t.Parallel()

	negativeCount := int64(-1)
	tests := map[string]diagnosticRecord{
		"zero sequence":     {total: time.Second},
		"negative total":    {sequence: 1, total: -time.Nanosecond},
		"negative headers":  {sequence: 1, headers: durationPointer(-time.Nanosecond)},
		"negative body":     {sequence: 1, body: durationPointer(-time.Nanosecond)},
		"negative dns":      {sequence: 1, dns: []time.Duration{-time.Nanosecond}},
		"negative tcp":      {sequence: 1, tcp: []time.Duration{-time.Nanosecond}},
		"negative tls":      {sequence: 1, tls: []time.Duration{-time.Nanosecond}},
		"negative request":  {sequence: 1, requestBytes: &negativeCount},
		"negative response": {sequence: 1, responseBytes: &negativeCount},
	}
	for name, record := range tests {
		record := record
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := record.format()
			require.Error(t, err)
		})
	}
}

// TestAPIDiagnosticsRecordFailureStatusEvidence verifies transport failures do not invent status and body failures retain it.
func TestAPIDiagnosticsRecordFailureStatusEvidence(t *testing.T) {
	t.Parallel()

	status := 502
	responseComplete := false
	bodyFailure := diagnosticRecord{
		sequence:         8,
		method:           "GET",
		target:           "/v2/topology",
		status:           &status,
		failure:          diagnosticFailureBody,
		total:            2 * time.Second,
		headers:          durationPointer(100 * time.Millisecond),
		body:             durationPointer(1900 * time.Millisecond),
		phase:            diagnosticPhaseBody,
		responseComplete: &responseComplete,
	}
	got, err := bodyFailure.format()
	require.NoError(t, err)
	require.Equal(t, "api #8 GET /v2/topology: status=502 error=BODY_ERROR total=2s headers=100ms body=1.9s phase=body response-complete=false", got)

	transportFailure := diagnosticRecord{
		sequence: 9,
		method:   "GET",
		target:   "/v2/topology",
		failure:  diagnosticFailureConnect,
		total:    12 * time.Millisecond,
		phase:    diagnosticPhaseConnect,
		reason:   diagnosticReasonConnectionRefused,
	}
	got, err = transportFailure.format()
	require.NoError(t, err)
	require.Equal(t, "api #9 GET /v2/topology: error=CONNECT_ERROR total=12ms phase=connect reason=connection-refused", got)
	require.NotContains(t, got, "status=")
}

// durationPointer keeps optional duration setup concise without hiding zero values.
func durationPointer(value time.Duration) *time.Duration {
	return &value
}
