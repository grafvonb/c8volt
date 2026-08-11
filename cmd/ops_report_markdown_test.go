// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

// TestWriteMarkdownReportFieldUsesDashForEmptyValues verifies shared Markdown report fields stay compact and placeholder-safe.
func TestWriteMarkdownReportFieldUsesDashForEmptyValues(t *testing.T) {
	var out strings.Builder

	writeMarkdownReportField(&out, "Schema Version", "ops.example.v1")
	writeMarkdownReportField(&out, "Profile", "")

	require.Equal(t, "- Schema Version: ops.example.v1\n- Profile: -\n", out.String())
}

// TestWriteMarkdownReportListUsesNestedBullets verifies shared Markdown report lists keep empty and populated values stable.
func TestWriteMarkdownReportListUsesNestedBullets(t *testing.T) {
	var out strings.Builder

	writeMarkdownReportList(&out, "Errors", nil)
	writeMarkdownReportList(&out, "Keys", []string{"2251799813685249", "2251799813685250"})

	require.Equal(t, "- Errors: -\n- Keys:\n  - 2251799813685249\n  - 2251799813685250\n", out.String())
}

// TestFormatOpsPurgeReportTimeUsesUTCAndConfiguredOffset verifies Markdown report timestamps follow the shared report time policy.
func TestFormatOpsPurgeReportTimeUsesUTCAndConfiguredOffset(t *testing.T) {
	when := time.Date(2026, time.August, 10, 12, 34, 56, 0, time.FixedZone("CEST", 2*60*60))

	require.Empty(t, formatOpsPurgeReportTime(time.Time{}, nil))
	require.Equal(t, "2026-08-10T10:34:56.000", formatOpsPurgeReportTime(when, nil))
	require.Equal(t, "2026-08-10T10:34:56.000", formatOpsPurgeReportTime(when, &config.Config{}))
	require.Equal(t, "2026-08-10T10:34:56.000+00:00", formatOpsPurgeReportTime(when, &config.Config{App: config.App{ShowTimezoneOffset: true}}))
}
