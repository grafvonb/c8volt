// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

// invocationHTTPService exposes lifecycle controls to the existing singleton CLI owner.
type invocationHTTPService interface {
	Close()
	PauseRequestRateReporting() func()
}

// currentHTTPService holds ownership only; request counters remain service-local.
// Like the singleton command tree, this handle is not for concurrent invocations.
var currentHTTPService invocationHTTPService

// init registers one finalizer, including for returned pre-run and execution failures.
func init() { cobra.OnFinalize(closeInvocationHTTPService) }

// closeInvocationHTTPService consumes ownership before closing, making cleanup idempotent.
func closeInvocationHTTPService() {
	service := currentHTTPService
	currentHTTPService = nil
	if service != nil {
		service.Close()
	}
}

// ownInvocationHTTPService replaces a stale invocation without accumulating finalizers.
func ownInvocationHTTPService(service invocationHTTPService) {
	closeInvocationHTTPService()
	currentHTTPService = service
}

// pauseInvocationRequestRateReporting protects both shared manual-input policies
// without changing each command's prompt writer, eligibility, or call signature.
func pauseInvocationRequestRateReporting() func() {
	if currentHTTPService == nil {
		return func() {}
	}
	return currentHTTPService.PauseRequestRateReporting()
}
