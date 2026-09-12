// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"io"
	"time"
)

type diagnosticRequestBody struct {
	delegate io.ReadCloser
	exchange *diagnosticExchange
}

func newDiagnosticRequestBody(delegate io.ReadCloser, exchange *diagnosticExchange) io.ReadCloser {
	body := &diagnosticRequestBody{delegate: delegate, exchange: exchange}
	if writerTo, ok := delegate.(io.WriterTo); ok {
		return &diagnosticRequestBodyWriterTo{diagnosticRequestBody: body, writerTo: writerTo}
	}
	return body
}

func (body *diagnosticRequestBody) Read(buffer []byte) (int, error) {
	n, err := body.delegate.Read(buffer)
	body.exchange.update(func(record *diagnosticRecord) {
		if record.requestBytes == nil {
			record.requestBytes = diagnosticInt64Pointer(0)
		}
		*record.requestBytes += int64(n)
		if err == io.EOF {
			record.requestComplete = diagnosticBoolPointer(true)
		}
	})
	return n, err
}

func (body *diagnosticRequestBody) Close() error { return body.delegate.Close() }

type diagnosticRequestBodyWriterTo struct {
	*diagnosticRequestBody
	writerTo io.WriterTo
}

func (body *diagnosticRequestBodyWriterTo) WriteTo(writer io.Writer) (int64, error) {
	n, err := body.writerTo.WriteTo(writer)
	body.exchange.update(func(record *diagnosticRecord) {
		if record.requestBytes == nil {
			record.requestBytes = diagnosticInt64Pointer(0)
		}
		*record.requestBytes += n
		if err == nil {
			record.requestComplete = diagnosticBoolPointer(true)
		}
	})
	return n, err
}

type diagnosticResponseBody struct {
	delegate io.ReadCloser
	exchange *diagnosticExchange
	now      func() time.Time
}

func newDiagnosticResponseBody(delegate io.ReadCloser, exchange *diagnosticExchange, now func() time.Time) io.ReadCloser {
	if delegate == nil {
		delegate = httpNoBody{}
	}
	body := &diagnosticResponseBody{delegate: delegate, exchange: exchange, now: now}
	if writerTo, ok := delegate.(io.WriterTo); ok {
		return &diagnosticResponseBodyWriterTo{diagnosticResponseBody: body, writerTo: writerTo}
	}
	return body
}

func (body *diagnosticResponseBody) Read(buffer []byte) (int, error) {
	n, err := body.delegate.Read(buffer)
	body.exchange.update(func(record *diagnosticRecord) {
		if record.responseBytes == nil {
			record.responseBytes = diagnosticInt64Pointer(0)
		}
		*record.responseBytes += int64(n)
	})
	if err != nil {
		body.finish(err, false, err == io.EOF)
	}
	return n, err
}

func (body *diagnosticResponseBody) Close() error {
	err := body.delegate.Close()
	body.finish(err, true, false)
	return err
}

type diagnosticResponseBodyWriterTo struct {
	*diagnosticResponseBody
	writerTo io.WriterTo
}

func (body *diagnosticResponseBodyWriterTo) WriteTo(writer io.Writer) (int64, error) {
	n, err := body.writerTo.WriteTo(writer)
	body.exchange.update(func(record *diagnosticRecord) {
		if record.responseBytes == nil {
			record.responseBytes = diagnosticInt64Pointer(0)
		}
		*record.responseBytes += n
	})
	body.finish(err, false, err == nil)
	return n, err
}

func (body *diagnosticResponseBody) finish(err error, closing, complete bool) {
	ended := body.now()
	body.exchange.finish(func(record *diagnosticRecord) {
		record.total = elapsedDiagnostic(body.exchange.started, ended)
		bodyDuration := elapsedDiagnostic(body.exchange.headersAt, ended)
		record.body = &bodyDuration
		if record.responseBytes == nil {
			record.responseBytes = diagnosticInt64Pointer(0)
		}
		record.responseComplete = diagnosticBoolPointer(complete)
		if err != nil && err != io.EOF {
			record.failure, record.reason = classifyDiagnosticError(err, diagnosticPhaseBody, closing)
			record.phase = diagnosticPhaseBody
		}
		body.exchange.copyTraceEvidence(record)
	})
}

type httpNoBody struct{}

func (httpNoBody) Read([]byte) (int, error) { return 0, io.EOF }
func (httpNoBody) Close() error             { return nil }

func diagnosticInt64Pointer(value int64) *int64 { return &value }
func diagnosticBoolPointer(value bool) *bool    { return &value }
