// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func requireVolumeJSONReportFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read JSON report %s: %v", path, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode JSON report %s: %v", path, err)
	}
	return payload
}

func requireVolumeMarkdownReportSections(t *testing.T, content string, sections ...string) {
	t.Helper()
	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Fatalf("markdown report missing section %q\n%s", section, content)
		}
	}
}
