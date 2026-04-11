package main

import (
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/cmd"
)

func TestFormatDocsBuildInfo(t *testing.T) {
	info := cmd.BuildInfo{
		Version:                  "v2.1.0",
		Commit:                   "abcdef123456",
		Date:                     "2026-04-11T09:10:11Z",
		SupportedCamundaVersions: "8.7, 8.8",
	}

	got := formatDocsBuildInfo(info)

	for _, want := range []string{
		"Generated with",
		"c8volt v2.1.0 (abcdef123456, 2026-04-11T09:10:11Z) | camunda: 8.7, 8.8",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected build info to contain %q, got %q", want, got)
		}
	}
}
