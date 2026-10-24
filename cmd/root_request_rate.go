// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

// currentHTTPServiceCleanup holds ownership only; request counters remain service-local.
// Like the singleton command tree, this handle is not for concurrent invocations.
var currentHTTPServiceCleanup func()

// init registers one finalizer, including for returned pre-run and execution failures.
func init() { cobra.OnFinalize(closeInvocationHTTPService) }

// closeInvocationHTTPService consumes ownership before closing, making cleanup idempotent.
func closeInvocationHTTPService() {
	closeService := currentHTTPServiceCleanup
	currentHTTPServiceCleanup = nil
	if closeService != nil {
		closeService()
	}
}

// ownInvocationHTTPService replaces a stale invocation without accumulating finalizers.
func ownInvocationHTTPService(closeService func()) {
	closeInvocationHTTPService()
	currentHTTPServiceCleanup = closeService
}
