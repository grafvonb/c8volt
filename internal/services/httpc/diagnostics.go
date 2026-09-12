// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"log/slog"
	"net/http"
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
	collector *diagnosticCollector
	sanitizer *diagnosticSanitizer
	mu        sync.Mutex
	frozen    bool
	record    diagnosticRecord
	once      sync.Once
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
	for _, secret := range []string{cfg.Auth.OAuth2.ClientSecret, cfg.Auth.Cookie.Password} {
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
	}
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
