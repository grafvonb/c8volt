// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import "testing"

// TestIntegrationPackageNoBuildTagHarmless keeps `go test ./integration/cli`
// useful as a cheap guard even though real CLI scenarios require
// `-tags=integration`.
func TestIntegrationPackageNoBuildTagHarmless(t *testing.T) {
	t.Helper()
}
