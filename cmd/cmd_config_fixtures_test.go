// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func writeTestConfig(t *testing.T, baseURL string) string {
	t.Helper()
	return testx.WriteTestConfig(t, baseURL)
}

func writeTestConfigForVersion(t *testing.T, baseURL string, camundaVersion string) string {
	t.Helper()
	return testx.WriteTestConfigForVersion(t, baseURL, camundaVersion)
}

// writeRawTestConfig preserves malformed or partial YAML exactly enough for
// config validation tests while still trimming the leading newline used in
// readable fixture literals.
func writeRawTestConfig(t *testing.T, content string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(strings.TrimLeft(content, "\n")), 0o600))
	return cfgPath
}
