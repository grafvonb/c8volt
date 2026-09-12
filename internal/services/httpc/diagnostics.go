// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"sort"
	"sync"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
)

// diagnosticInvocationContext is the private immutable redaction seed for one invocation.
type diagnosticInvocationContext struct {
	profile string
	tenant  string
	secrets []string
}

// diagnosticCollector owns invocation-local sequencing, logger routing and redaction context.
type diagnosticCollector struct {
	log      *slog.Logger
	verbose  bool
	sequence diagnosticSequence
	context  diagnosticInvocationContext
}

// diagnosticExchange protects mutable observations until one terminal snapshot is frozen.
type diagnosticExchange struct {
	collector    *diagnosticCollector
	sanitizer    *diagnosticSanitizer
	mu           sync.Mutex
	frozen       bool
	record       diagnosticRecord
	once         sync.Once
	started      time.Time
	headersAt    time.Time
	phaseSeq     uint64
	dnsOpen      []diagnosticPhaseStart
	tlsOpen      []diagnosticPhaseStart
	connectOpen  map[string][]diagnosticPhaseStart
	dnsSamples   []diagnosticTimedSample
	tcpSamples   []diagnosticTimedSample
	tlsSamples   []diagnosticTimedSample
	failurePhase diagnosticPhase
}

type diagnosticPhaseStart struct {
	at    time.Time
	order uint64
}

type diagnosticTimedSample struct {
	started  time.Time
	duration time.Duration
	order    uint64
}

// DiagnosticsTransport observes one RoundTrip boundary without changing request semantics.
type DiagnosticsTransport struct {
	base      http.RoundTripper
	collector *diagnosticCollector
	now       func() time.Time
}

// newDiagnosticCollector avoids installing observation when verbose INFO output is unavailable.
func newDiagnosticCollector(cfg *config.Config, log *slog.Logger, verbose bool) *diagnosticCollector {
	if !verbose || log == nil || !log.Enabled(context.Background(), slog.LevelInfo) {
		return nil
	}
	return &diagnosticCollector{
		log:     log,
		verbose: verbose,
		context: newDiagnosticInvocationContext(cfg),
	}
}

// newDiagnosticInvocationContext copies only the configured context and secrets diagnostics need.
func newDiagnosticInvocationContext(cfg *config.Config) diagnosticInvocationContext {
	if cfg == nil {
		return diagnosticInvocationContext{}
	}
	invocation := diagnosticInvocationContext{
		profile: cfg.ActiveProfile,
		tenant:  cfg.App.Tenant,
	}
	for _, secret := range []string{cfg.Auth.OAuth2.ClientSecret, cfg.Auth.Cookie.Username, cfg.Auth.Cookie.Password} {
		if secret != "" {
			invocation.secrets = append(invocation.secrets, secret)
		}
	}
	return invocation
}

// start allocates one positive sequence and prepares an exchange-local sanitizer and record.
func (collector *diagnosticCollector) start(req *http.Request) *diagnosticExchange {
	sanitizer := newDiagnosticSanitizer(nil)
	sanitizer.profile = collector.context.profile
	sanitizer.tenant = collector.context.tenant
	for _, secret := range collector.context.secrets {
		sanitizer.addSecret(secret)
	}
	request := sanitizer.sanitizeRequest(req)
	return &diagnosticExchange{
		collector: collector,
		sanitizer: sanitizer,
		record: diagnosticRecord{
			sequence:            collector.sequence.next(),
			method:              request.method,
			target:              request.target,
			host:                request.host,
			profile:             request.profile,
			tenant:              request.tenant,
			clientRequestID:     request.clientRequestID,
			clientCorrelationID: request.clientCorrelationID,
		},
		connectOpen: make(map[string][]diagnosticPhaseStart),
	}
}

func (transport *DiagnosticsTransport) rt() http.RoundTripper {
	if transport != nil && transport.base != nil {
		return transport.base
	}
	return http.DefaultTransport
}

func (transport *DiagnosticsTransport) clock() func() time.Time {
	if transport != nil && transport.now != nil {
		return transport.now
	}
	return time.Now
}

// RoundTrip attaches transparent byte/lifecycle observation below retry and logging transports.
func (transport *DiagnosticsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if transport == nil || transport.collector == nil {
		return transport.rt().RoundTrip(req)
	}
	now := transport.clock()
	exchange := transport.collector.start(req)
	exchange.started = now()

	observed := req.Clone(httptrace.WithClientTrace(req.Context(), exchange.clientTrace(now)))
	if req.Body == nil || req.Body == http.NoBody {
		exchange.update(func(record *diagnosticRecord) {
			record.requestBytes = diagnosticInt64Pointer(0)
			record.requestComplete = diagnosticBoolPointer(true)
		})
	} else {
		observed.Body = newDiagnosticRequestBody(req.Body, exchange)
	}

	response, err := transport.rt().RoundTrip(observed)
	headersAt := now()
	if err != nil {
		exchange.finish(func(record *diagnosticRecord) {
			record.total = elapsedDiagnostic(exchange.started, headersAt)
			classifiedError := err
			if contextError := observed.Context().Err(); contextError != nil {
				classifiedError = contextError
			}
			record.failure, record.reason = classifyDiagnosticError(classifiedError, exchange.failurePhase, false)
			record.phase = exchange.failurePhase
			exchange.copyTraceEvidence(record)
		})
		return response, err
	}

	exchange.headersAt = headersAt
	metadata := exchange.sanitizer.sanitizeResponse(response)
	exchange.update(func(record *diagnosticRecord) {
		status := response.StatusCode
		record.status = &status
		headers := elapsedDiagnostic(exchange.started, headersAt)
		record.headers = &headers
		record.requestID = metadata.requestID
		record.correlationID = metadata.correlationID
		record.retryAfter = metadata.retryAfter
		record.serverTiming = metadata.serverTiming
	})

	if diagnosticKnownBodyless(req, response) {
		exchange.finish(func(record *diagnosticRecord) {
			zero := time.Duration(0)
			record.body = &zero
			record.total = elapsedDiagnostic(exchange.started, headersAt)
			record.responseBytes = diagnosticInt64Pointer(0)
			record.responseComplete = diagnosticBoolPointer(true)
			exchange.copyTraceEvidence(record)
		})
		return response, nil
	}
	response.Body = newDiagnosticResponseBody(response.Body, exchange, now)
	return response, nil
}

func diagnosticKnownBodyless(req *http.Request, response *http.Response) bool {
	return req.Method == http.MethodHead || response.Body == nil || response.Body == http.NoBody ||
		response.StatusCode == http.StatusNoContent || response.StatusCode == http.StatusNotModified ||
		(response.StatusCode >= 100 && response.StatusCode < 200)
}

func elapsedDiagnostic(start, end time.Time) time.Duration {
	duration := end.Sub(start)
	if duration < 0 {
		return 0
	}
	return duration
}

func (exchange *diagnosticExchange) clientTrace(now func() time.Time) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart:          func(httptrace.DNSStartInfo) { exchange.startSimplePhase(diagnosticPhaseDNS, now()) },
		DNSDone:           func(info httptrace.DNSDoneInfo) { exchange.finishSimplePhase(diagnosticPhaseDNS, now(), info.Err) },
		ConnectStart:      func(network, address string) { exchange.startConnect(network, address, now()) },
		ConnectDone:       func(network, address string, err error) { exchange.finishConnect(network, address, now(), err) },
		TLSHandshakeStart: func() { exchange.startSimplePhase(diagnosticPhaseTLS, now()) },
		TLSHandshakeDone:  func(_ tls.ConnectionState, err error) { exchange.finishSimplePhase(diagnosticPhaseTLS, now(), err) },
		GotConn: func(info httptrace.GotConnInfo) {
			exchange.update(func(record *diagnosticRecord) {
				if info.Reused {
					record.connection = diagnosticConnectionReused
				} else {
					record.connection = diagnosticConnectionNew
				}
			})
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err != nil {
				exchange.setFailurePhase(diagnosticPhaseRequestWrite)
			}
		},
	}
}

func (exchange *diagnosticExchange) nextPhaseStart(at time.Time) diagnosticPhaseStart {
	exchange.phaseSeq++
	return diagnosticPhaseStart{at: at, order: exchange.phaseSeq}
}

func (exchange *diagnosticExchange) startSimplePhase(phase diagnosticPhase, at time.Time) {
	exchange.mu.Lock()
	defer exchange.mu.Unlock()
	if exchange.frozen {
		return
	}
	start := exchange.nextPhaseStart(at)
	if phase == diagnosticPhaseDNS {
		exchange.dnsOpen = append(exchange.dnsOpen, start)
	} else {
		exchange.tlsOpen = append(exchange.tlsOpen, start)
	}
}

func (exchange *diagnosticExchange) finishSimplePhase(phase diagnosticPhase, at time.Time, err error) {
	exchange.mu.Lock()
	defer exchange.mu.Unlock()
	if exchange.frozen {
		return
	}
	var open *[]diagnosticPhaseStart
	var samples *[]diagnosticTimedSample
	if phase == diagnosticPhaseDNS {
		open, samples = &exchange.dnsOpen, &exchange.dnsSamples
	} else {
		open, samples = &exchange.tlsOpen, &exchange.tlsSamples
	}
	if len(*open) > 0 {
		start := (*open)[0]
		*open = (*open)[1:]
		*samples = append(*samples, diagnosticTimedSample{started: start.at, duration: elapsedDiagnostic(start.at, at), order: start.order})
	}
	if err != nil {
		exchange.failurePhase = phase
	}
}

func (exchange *diagnosticExchange) startConnect(network, address string, at time.Time) {
	exchange.mu.Lock()
	defer exchange.mu.Unlock()
	if exchange.frozen {
		return
	}
	key := network + "\x00" + address
	exchange.connectOpen[key] = append(exchange.connectOpen[key], exchange.nextPhaseStart(at))
}

func (exchange *diagnosticExchange) finishConnect(network, address string, at time.Time, err error) {
	exchange.mu.Lock()
	defer exchange.mu.Unlock()
	if exchange.frozen {
		return
	}
	key := network + "\x00" + address
	open := exchange.connectOpen[key]
	if len(open) > 0 {
		start := open[0]
		exchange.connectOpen[key] = open[1:]
		if tcpNetwork(network) {
			exchange.tcpSamples = append(exchange.tcpSamples, diagnosticTimedSample{started: start.at, duration: elapsedDiagnostic(start.at, at), order: start.order})
		}
	}
	if err != nil {
		exchange.failurePhase = diagnosticPhaseConnect
	}
}

func tcpNetwork(network string) bool {
	return network == "tcp" || network == "tcp4" || network == "tcp6"
}

func (exchange *diagnosticExchange) setFailurePhase(phase diagnosticPhase) {
	exchange.mu.Lock()
	defer exchange.mu.Unlock()
	if !exchange.frozen {
		exchange.failurePhase = phase
	}
}

func (exchange *diagnosticExchange) copyTraceEvidence(record *diagnosticRecord) {
	record.dns = diagnosticSampleDurations(exchange.dnsSamples)
	record.tcp = diagnosticSampleDurations(exchange.tcpSamples)
	record.tls = diagnosticSampleDurations(exchange.tlsSamples)
}

func diagnosticSampleDurations(samples []diagnosticTimedSample) []time.Duration {
	ordered := append([]diagnosticTimedSample(nil), samples...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].started.Equal(ordered[j].started) {
			return ordered[i].order < ordered[j].order
		}
		return ordered[i].started.Before(ordered[j].started)
	})
	result := make([]time.Duration, len(ordered))
	for index := range ordered {
		result[index] = ordered[index].duration
	}
	return result
}

// update applies an observation only while the exchange remains mutable.
func (exchange *diagnosticExchange) update(apply func(*diagnosticRecord)) bool {
	if exchange == nil || apply == nil {
		return false
	}
	exchange.mu.Lock()
	defer exchange.mu.Unlock()
	if exchange.frozen {
		return false
	}
	apply(&exchange.record)
	return true
}

// finish freezes one immutable snapshot under lock and emits it only after releasing that lock.
func (exchange *diagnosticExchange) finish(apply func(*diagnosticRecord)) {
	if exchange == nil || exchange.collector == nil {
		return
	}
	var snapshot diagnosticRecord
	frozenNow := false
	exchange.once.Do(func() {
		exchange.mu.Lock()
		if apply != nil {
			apply(&exchange.record)
		}
		exchange.frozen = true
		snapshot = cloneDiagnosticRecord(exchange.record)
		exchange.mu.Unlock()
		frozenNow = true
	})
	if !frozenNow {
		return
	}
	exchange.collector.emit(snapshot)
}

// cloneDiagnosticRecord prevents later pointer or slice mutation from changing a frozen snapshot.
func cloneDiagnosticRecord(record diagnosticRecord) diagnosticRecord {
	record.status = cloneDiagnosticPointer(record.status)
	record.headers = cloneDiagnosticPointer(record.headers)
	record.body = cloneDiagnosticPointer(record.body)
	record.requestBytes = cloneDiagnosticPointer(record.requestBytes)
	record.requestComplete = cloneDiagnosticPointer(record.requestComplete)
	record.responseBytes = cloneDiagnosticPointer(record.responseBytes)
	record.responseComplete = cloneDiagnosticPointer(record.responseComplete)
	record.dns = append([]time.Duration(nil), record.dns...)
	record.tcp = append([]time.Duration(nil), record.tcp...)
	record.tls = append([]time.Duration(nil), record.tls...)
	return record
}

// cloneDiagnosticPointer copies optional scalar evidence into snapshot-owned storage.
func cloneDiagnosticPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// emit formats safe fields and delegates one complete line to the existing verbose logger.
func (collector *diagnosticCollector) emit(record diagnosticRecord) {
	message, err := record.format()
	if err != nil {
		return
	}
	logging.InfoIfVerbose(message, collector.log, collector.verbose)
}
