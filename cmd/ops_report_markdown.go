// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
)

// writeMarkdownReportField renders one shared Markdown report field with a placeholder for empty values.
func writeMarkdownReportField(out *strings.Builder, name string, value string) {
	if value == "" {
		value = "-"
	}
	out.WriteString(fmt.Sprintf("- %s: %s\n", name, value))
}

// writeMarkdownReportList renders a shared Markdown report list while preserving the empty-list placeholder.
func writeMarkdownReportList(out *strings.Builder, name string, values []string) {
	if len(values) == 0 {
		out.WriteString(fmt.Sprintf("- %s: -\n", name))
		return
	}
	out.WriteString(fmt.Sprintf("- %s:\n", name))
	for _, value := range values {
		out.WriteString(fmt.Sprintf("  - %s\n", value))
	}
}

// formatOpsPurgeReportTime normalizes shared ops Markdown report timestamps to UTC.
func formatOpsPurgeReportTime(t time.Time, cfg *config.Config) string {
	if t.IsZero() {
		return ""
	}
	showTimezoneOffset := false
	if cfg != nil {
		showTimezoneOffset = cfg.App.ShowTimezoneOffset
	}
	return toolx.FormatTime(t.UTC(), showTimezoneOffset)
}
