// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	envITVolumeCount         = "C8VOLT_IT_VOLUME_COUNT"
	defaultITVolumeCount     = 12
	minimumITVolumeCount     = 3
	volumeOutcomePass        = "pass"
	volumeOutcomeFail        = "fail"
	volumeFailureProduct     = "product"
	volumeFailureHarness     = "harness_setup"
	volumeFailureEnvironment = "environment_availability"
	volumeDataSeeded         = "seeded"
	volumeDataPreexisting    = "preexisting"
)

type volumeTarget struct {
	Name                   string   `json:"name"`
	Family                 string   `json:"family"`
	TestPattern            string   `json:"testPattern"`
	DefaultDatasetCount    int      `json:"defaultDatasetCount"`
	Destructive            bool     `json:"destructive"`
	Profiles               []string `json:"profiles,omitempty"`
	ReservedNotImplemented bool     `json:"reservedNotImplemented,omitempty"`
}

func volumeDatasetCount(t *testing.T) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(envITVolumeCount))
	if raw == "" {
		return defaultITVolumeCount
	}
	count, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("%s must be a positive integer, got %q", envITVolumeCount, raw)
	}
	if count < minimumITVolumeCount {
		t.Fatalf("%s=%d is too small for volume coverage; minimum is %d", envITVolumeCount, count, minimumITVolumeCount)
	}
	return count
}

func volumeTargetEvidencePath(family string, name string) string {
	cleanFamily := sanitizeEvidenceName(family)
	cleanName := filepath.Clean(name)
	if cleanName == "." || cleanName == "" {
		return "volume-" + cleanFamily + ".json"
	}
	return filepath.Join("data", "volume-"+cleanFamily, cleanName)
}

func allVolumeTargets() []volumeTarget {
	return []volumeTarget{
		{Name: "integration-cli-get-volume", Family: "get", TestPattern: "TestVolumeGetFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true},
		{Name: "integration-cli-walk-volume", Family: "walk", TestPattern: "TestVolumeWalkFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true},
		{Name: "integration-cli-update-volume", Family: "update", TestPattern: "TestVolumeUpdateFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true},
		{Name: "integration-cli-cancel-volume", Family: "cancel", TestPattern: "TestVolumeCancelFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true},
		{Name: "integration-cli-delete-volume", Family: "delete", TestPattern: "TestVolumeDeleteFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true},
		{Name: "integration-cli-expect-resolve-volume", Family: "expect-resolve", TestPattern: "TestVolumeExpectResolveFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true},
		{Name: "integration-cli-deploy-embed-run-volume", Family: "deploy-embed-run", TestPattern: "TestVolumeDeployEmbedRunFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true},
		{Name: "integration-cli-ops-analyse-volume", Family: "ops-analyse", TestPattern: "TestVolumeOpsAnalyseFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true},
		{Name: "integration-cli-ops-execute-volume", Family: "ops-execute", TestPattern: "TestVolumeOpsExecuteFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true, ReservedNotImplemented: true},
		{Name: "integration-cli-ops-purge-volume", Family: "ops-purge", TestPattern: "TestVolumeOpsPurgeFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true, ReservedNotImplemented: true},
		{Name: "integration-cli-ops-repair-volume", Family: "ops-repair", TestPattern: "TestVolumeOpsRepairFamily", DefaultDatasetCount: defaultITVolumeCount, Destructive: true, ReservedNotImplemented: true},
	}
}

func TestVolumeTargetCatalog(t *testing.T) {
	targets := allVolumeTargets()
	if len(targets) != 11 {
		t.Fatalf("volume target count = %d, want 11", len(targets))
	}
	seen := map[string]struct{}{}
	for _, target := range targets {
		if !strings.HasPrefix(target.Name, "integration-cli-") || !strings.HasSuffix(target.Name, "-volume") {
			t.Fatalf("volume target %q does not use required naming", target.Name)
		}
		if target.Family == "" || target.TestPattern == "" {
			t.Fatalf("volume target has empty family or test pattern: %+v", target)
		}
		if target.DefaultDatasetCount < minimumITVolumeCount {
			t.Fatalf("volume target %q default count %d below minimum %d", target.Name, target.DefaultDatasetCount, minimumITVolumeCount)
		}
		if _, ok := seen[target.Name]; ok {
			t.Fatalf("duplicate volume target %q", target.Name)
		}
		seen[target.Name] = struct{}{}
	}
}
