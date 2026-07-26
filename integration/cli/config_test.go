// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type configTestConnectionPayload struct {
	OK      bool `json:"ok"`
	Cluster struct {
		GatewayVersion string `json:"gateway_version"`
	} `json:"cluster"`
}

// TestRejectExplicitGeneratedConfigUsage preserves the suite contract that real runs never pass --config.
func TestRejectExplicitGeneratedConfigUsage(t *testing.T) {
	cases := [][]string{
		{"--config", "/tmp/generated-c8volt.yaml", "version"},
		{"--config=/tmp/generated-c8volt.yaml", "version"},
		{"-c", "/tmp/generated-c8volt.yaml", "version"},
	}

	for _, args := range cases {
		if err := rejectExplicitConfigArgs(args); err == nil {
			t.Fatalf("rejectExplicitConfigArgs(%v) returned nil", args)
		}
	}
	if err := rejectExplicitConfigArgs([]string{"--profile", "local-c89", "version"}); err != nil {
		t.Fatalf("profile selection should not be rejected: %v", err)
	}
}

// TestProfileSelectionRequiresDefaultLocalProfile verifies selected names must exist in default-local config metadata.
func TestProfileSelectionRequiresDefaultLocalProfile(t *testing.T) {
	profiles, err := profilesFromDefaultConfig(defaultLocalConfig{
		ActiveProfile: "dev89",
		App:           defaultLocalConfigApp{CamundaVersion: "8.8"},
		Profiles: map[string]defaultLocalProfile{
			"dev89": {App: defaultLocalConfigApp{CamundaVersion: "8.9", Tenant: "<default>"}},
		},
		sourcePath: "/home/operator/.config/c8volt/config.yaml",
	}, nil)
	if err != nil {
		t.Fatalf("profilesFromDefaultConfig returned error: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "dev89" || profiles[0].ExpectedVersion != "8.9" {
		t.Fatalf("profiles = %+v, want active dev89 with version 8.9", profiles)
	}

	_, err = profilesFromDefaultConfig(defaultLocalConfig{
		Profiles:   map[string]defaultLocalProfile{"dev89": {}},
		sourcePath: "/home/operator/.config/c8volt/config.yaml",
	}, []string{"missing"})
	if err == nil {
		t.Fatal("missing selected profile should fail")
	}
}

// TestProfiles writes readiness evidence and fails early when selected local profiles are not reachable.
func TestProfiles(t *testing.T) {
	profiles := requireSelectedProfiles(t)
	records := make([]integrationProfile, 0, len(profiles))
	var failures []string

	for _, profile := range profiles {
		record, err := checkProfileReadiness(t, profile)
		if err != nil {
			failures = append(failures, err.Error())
		}
		records = append(records, record)
	}

	writeJSON(t, "profiles.json", records)
	if len(failures) > 0 {
		t.Fatalf("profile readiness failed:\n%s", strings.Join(failures, "\n"))
	}
}

// TestReadOnlySmoke records smoke evidence for read-only commands that must succeed before destructive stories.
func TestReadOnlySmoke(t *testing.T) {
	profiles := requireSelectedProfiles(t)
	records := make([]evidenceRecord, 0, 2+len(profiles)*2)
	var failures []string

	for _, smoke := range []readOnlySmoke{
		{commandPath: "version", scenarioName: "readonly-version", args: []string{"version"}},
		{commandPath: "capabilities", scenarioName: "readonly-capabilities-json", args: []string{"capabilities", "--json"}, wantJSON: true},
	} {
		record, err := runReadOnlySmoke(t, smoke)
		if err != nil {
			failures = append(failures, err.Error())
		}
		records = append(records, record)
	}

	for _, profile := range profiles {
		for _, smoke := range []readOnlySmoke{
			{commandPath: "config validate", scenarioName: "readonly-" + profile.Name + "-config-validate", profile: profile, args: []string{"config", "validate"}},
			{commandPath: "config test-connection", scenarioName: "readonly-" + profile.Name + "-config-test-connection-json", profile: profile, args: []string{"--json", "config", "test-connection"}, wantJSON: true},
		} {
			record, err := runReadOnlySmoke(t, smoke)
			if err != nil {
				failures = append(failures, err.Error())
			}
			records = append(records, record)
		}
	}

	writeEvidenceRecords(t, "readonly-smoke.json", records)
	if len(failures) > 0 {
		t.Fatalf("read-only smoke failed:\n%s", strings.Join(failures, "\n"))
	}
}

type readOnlySmoke struct {
	commandPath  string
	scenarioName string
	profile      integrationProfile
	args         []string
	wantJSON     bool
}

// requireSelectedProfiles skips real-cluster gates when no default-local profile was selected.
func requireSelectedProfiles(t *testing.T) []integrationProfile {
	t.Helper()
	profiles, err := selectedProfilesFromDefaultConfig()
	if err != nil {
		t.Fatalf("select integration profiles: %v", err)
	}
	if len(profiles) == 0 {
		t.Skipf("no integration profiles selected; set %s to run profile gates", envITProfiles)
	}
	return profiles
}

// checkProfileReadiness runs the remote profile gate and returns evidence even for failures.
func checkProfileReadiness(t *testing.T, profile integrationProfile) (integrationProfile, error) {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "profile-"+profile.Name+"-test-connection-json", "--json", "config", "test-connection")
	record := profile
	record.Reachable = result.Err == nil
	if result.Err != nil {
		return record, fmt.Errorf("profile %q connection failed: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}

	payload, err := parseConfigTestConnection(result.Stdout)
	if err != nil {
		return record, fmt.Errorf("profile %q connection JSON invalid: %w", profile.Name, err)
	}
	record.ActualVersion = payload.Cluster.GatewayVersion
	if profile.ExpectedVersion != "" && !strings.Contains(record.ActualVersion, profile.ExpectedVersion) {
		return record, fmt.Errorf("profile %q gateway version %q does not contain expected version %q", profile.Name, record.ActualVersion, profile.ExpectedVersion)
	}
	return record, nil
}

// runReadOnlySmoke captures command evidence and validates the expected output shape.
func runReadOnlySmoke(t *testing.T, smoke readOnlySmoke) (evidenceRecord, error) {
	t.Helper()
	args := append([]string(nil), smoke.args...)
	var result commandResult
	if smoke.profile.Name != "" {
		result = runC8VoltForProfile(t, smoke.profile.Name, smoke.scenarioName, args...)
	} else {
		result = runC8Volt(t, smoke.scenarioName, args...)
	}

	outcome := "pass"
	var err error
	if result.Err != nil {
		outcome = "fail"
		err = fmt.Errorf("%s failed: %v; stderr: %s", smoke.commandPath, result.Err, strings.TrimSpace(result.Stderr))
	} else if smoke.wantJSON && !json.Valid([]byte(result.Stdout)) {
		outcome = "fail"
		err = fmt.Errorf("%s did not produce valid JSON: %s", smoke.commandPath, result.Stdout)
	} else if !smoke.wantJSON && strings.TrimSpace(result.Stdout) == "" && strings.TrimSpace(result.Stderr) == "" {
		outcome = "fail"
		err = fmt.Errorf("%s produced empty output", smoke.commandPath)
	}

	record := commandEvidence(smoke.commandPath, smoke.scenarioName, result, outcome)
	if smoke.profile.Name != "" {
		record.Profile = smoke.profile.Name
		record.CamundaVersion = smoke.profile.ExpectedVersion
	}
	return record, err
}

// parseConfigTestConnection decodes the stable readiness fields used by the profile gate.
func parseConfigTestConnection(output string) (configTestConnectionPayload, error) {
	var payload configTestConnectionPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return payload, err
	}
	if !payload.OK {
		return payload, fmt.Errorf("connection payload ok=false")
	}
	if strings.TrimSpace(payload.Cluster.GatewayVersion) == "" {
		return payload, fmt.Errorf("connection payload missing gateway version")
	}
	return payload, nil
}
