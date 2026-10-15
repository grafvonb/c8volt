// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/integration/internal/validation"
)

const (
	realStateOutcomeLiveCovered   = "live-covered"
	realStateOutcomePartialLive   = "partially-live-covered"
	realStateOutcomeDryRunCovered = "dry-run-covered"
	realStateOutcomeSkippedPrereq = "skipped-prerequisite"
	realStateOutcomeUnsupported   = "unsupported"
	realStateOutcomeFailed        = "failed"
)

// realStateTarget describes one independently runnable real-state Make target.
type realStateTarget struct {
	Name          string `json:"name"`
	Topic         string `json:"topic"`
	TestPattern   string `json:"testPattern"`
	Destructive   bool   `json:"destructive"`
	RequiredState string `json:"requiredState"`
}

// allRealStateTargets lists the stable operator entry points reserved by feature 257.
func allRealStateTargets() []realStateTarget {
	return []realStateTarget{
		{Name: "integration-cli-real-state-gaps", Topic: "gap validation", TestPattern: "TestRealStateGapFamily", Destructive: false, RequiredState: "spec-owned gap artifact consistency"},
		{Name: "integration-cli-real-state-jobs", Topic: "jobs", TestPattern: "TestRealStateJobsFamily", Destructive: true, RequiredState: "suite-owned active jobs"},
		{Name: "integration-cli-real-state-incidents", Topic: "incidents", TestPattern: "TestRealStateIncidentsFamily", Destructive: true, RequiredState: "suite-owned incidents with related jobs"},
		{Name: "integration-cli-real-state-listeners", Topic: "listeners", TestPattern: "TestRealStateListenersFamily", Destructive: true, RequiredState: "listener jobs or listener element evidence"},
		{Name: "integration-cli-real-state-bpmn-error", Topic: "bpmn-error", TestPattern: "TestRealStateBPMNErrorFamily", Destructive: true, RequiredState: "BPMN error-capable job"},
		{Name: "integration-cli-real-state-retention", Topic: "retention", TestPattern: "TestRealStateRetentionFamily", Destructive: true, RequiredState: "deterministic completed retention candidates"},
		{Name: "integration-cli-real-state-destructive", Topic: "destructive", TestPattern: "TestRealStateDestructiveFamily", Destructive: true, RequiredState: "real purge, delete, cancel, resolve, repair, and mixed-failure candidates"},
		{Name: "integration-cli-real-state-cascade", Topic: "cascade", TestPattern: "TestRealStateCascadeFamily", Destructive: true, RequiredState: "real parent and child process-instance families"},
	}
}

// selectedRealStateProfiles returns reachable 8.9/8.10 profiles selected from
// the default local config while leaving 8.8 baseline/volume selection intact.
func selectedRealStateProfiles(t *testing.T) []integrationProfile {
	t.Helper()
	profiles := requireSelectedProfiles(t)
	var realStateProfiles []integrationProfile
	for _, profile := range profiles {
		if !realStateProfileSupported(profile) {
			continue
		}
		ready, err := checkProfileReadiness(t, profile)
		if err != nil {
			t.Fatalf("real-state profile %q is not ready: %v", profile.Name, err)
		}
		realStateProfiles = append(realStateProfiles, ready)
	}
	if len(realStateProfiles) == 0 {
		t.Skipf("no Camunda 8.9 or 8.10 profiles selected; set %s to a supported disposable profile", envITProfiles)
	}
	return realStateProfiles
}

// realStateProfileSupported classifies profiles by their canonical configured
// minor before any readiness check or mutation is attempted.
func realStateProfileSupported(profile integrationProfile) bool {
	_, err := validation.RealStateMinor(profile.ExpectedVersion)
	return err == nil
}

// supportedRealStateVersions is the shared explicit affected-version list for
// gap artifacts and tests.
func supportedRealStateVersions() []string {
	return []string{"8.9", "8.10"}
}

// TestRealStateTargetCatalog verifies the reserved real-state targets stay explicit and family-addressable.
func TestRealStateTargetCatalog(t *testing.T) {
	targets := allRealStateTargets()
	if len(targets) != 8 {
		t.Fatalf("real-state target count = %d, want 8", len(targets))
	}
	seen := map[string]struct{}{}
	for _, target := range targets {
		if !strings.HasPrefix(target.Name, "integration-cli-real-state-") {
			t.Fatalf("real-state target %q does not use required naming", target.Name)
		}
		if !strings.HasPrefix(target.TestPattern, "TestRealState") || !strings.HasSuffix(target.TestPattern, "Family") {
			t.Fatalf("real-state target %q has invalid test pattern %q", target.Name, target.TestPattern)
		}
		if strings.TrimSpace(target.Topic) == "" || strings.TrimSpace(target.RequiredState) == "" {
			t.Fatalf("real-state target has empty topic or required state: %+v", target)
		}
		if _, ok := seen[target.Name]; ok {
			t.Fatalf("duplicate real-state target %q", target.Name)
		}
		seen[target.Name] = struct{}{}
	}
}

// TestRealStateProfileClassification keeps live targets scoped to 8.9/8.10.
func TestRealStateProfileClassification(t *testing.T) {
	cases := []struct {
		name    string
		profile integrationProfile
		want    bool
	}{
		{name: "c89", profile: integrationProfile{Name: "c89", ExpectedVersion: "8.9"}, want: true},
		{name: "c810", profile: integrationProfile{Name: "c810", ExpectedVersion: "8.10"}, want: true},
		{name: "c88", profile: integrationProfile{Name: "c88", ExpectedVersion: "8.8"}, want: false},
		{name: "patch", profile: integrationProfile{Name: "patch", ExpectedVersion: "8.10.3"}, want: false},
		{name: "unparseable", profile: integrationProfile{Name: "bad", ExpectedVersion: "unknown"}, want: false},
		{name: "blank", profile: integrationProfile{Name: "default"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := realStateProfileSupported(tc.profile); got != tc.want {
				t.Fatalf("realStateProfileSupported(%+v) = %t, want %t", tc.profile, got, tc.want)
			}
		})
	}
}
