// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	behaviorDiscovery         = "discovery"
	behaviorSuccess           = "success"
	behaviorValidation        = "validation"
	behaviorPreview           = "preview"
	behaviorConfirmedMutation = "confirmed-mutation"
)

type behaviorScenario struct {
	CommandPath              string
	Name                     string
	Args                     []string
	RequiresProfile          bool
	RequiresSeed             bool
	CoveredFlags             []string
	OutputMode               string
	Behavior                 string
	VersionBehavior          string
	DataOwnership            []string
	Preview                  bool
	ConfirmedMutation        bool
	WantJSON                 bool
	WantKeysOnly             bool
	WantNonEmpty             bool
	WantFailureSubstring     string
	AllowUnsupportedVersions []string
}

type behaviorScenarioReport struct {
	Family   string                  `json:"family"`
	Profiles []integrationProfile    `json:"profiles,omitempty"`
	Records  []evidenceRecord        `json:"records"`
	Summary  behaviorScenarioSummary `json:"summary"`
}

type behaviorScenarioSummary struct {
	ScenarioCount      int                 `json:"scenarioCount"`
	CommandsCovered    []string            `json:"commandsCovered"`
	FlagsCovered       map[string][]string `json:"flagsCovered"`
	OutputModesCovered map[string][]string `json:"outputModesCovered"`
	BehaviorsCovered   map[string][]string `json:"behaviorsCovered"`
	VersionBehaviors   map[string][]string `json:"versionBehaviors"`
	Previews           []string            `json:"previews,omitempty"`
	ConfirmedMutations []string            `json:"confirmedMutations,omitempty"`
	DataOwnership      map[string][]string `json:"dataOwnership,omitempty"`
	BlockedOrSkipped   []string            `json:"blockedOrSkipped,omitempty"`
}

// TestBehavioralScenarioCatalogCoversConvergenceContract checks the scenario catalog before any cluster mutation.
func TestBehavioralScenarioCatalogCoversConvergenceContract(t *testing.T) {
	scenarios := allBehaviorScenarios()
	if len(scenarios) == 0 {
		t.Fatal("expected behavioral scenarios")
	}

	byPath := map[string][]behaviorScenario{}
	for _, scenario := range scenarios {
		if strings.TrimSpace(scenario.CommandPath) == "" {
			t.Fatalf("scenario %q has empty command path", scenario.Name)
		}
		if strings.TrimSpace(scenario.Name) == "" {
			t.Fatalf("%s has empty scenario name", scenario.CommandPath)
		}
		if strings.TrimSpace(scenario.Behavior) == "" {
			t.Fatalf("%s/%s has empty behavior", scenario.CommandPath, scenario.Name)
		}
		if strings.TrimSpace(scenario.VersionBehavior) == "" {
			t.Fatalf("%s/%s has empty version behavior", scenario.CommandPath, scenario.Name)
		}
		if len(scenario.DataOwnership) == 0 {
			t.Fatalf("%s/%s has empty data ownership classification", scenario.CommandPath, scenario.Name)
		}
		byPath[scenario.CommandPath] = append(byPath[scenario.CommandPath], scenario)
	}

	for path, entry := range commandCoverageManifest {
		pathScenarios := byPath[path]
		if len(pathScenarios) == 0 {
			t.Fatalf("manifest entry %q has no behavioral scenario", path)
		}
		if entry.Destructive && isLeafCommandPath(path) && containsString(entry.Flags, "dry-run") {
			if !hasBehavior(pathScenarios, behaviorPreview) {
				t.Fatalf("destructive command %q has no preview scenario", path)
			}
			if !hasBehavior(pathScenarios, behaviorConfirmedMutation) {
				t.Fatalf("destructive command %q has no confirmed mutation scenario", path)
			}
		}
	}
}

// runBehavioralCoverageScenarios executes real-profile scenarios for one command family and writes behavior evidence.
func runBehavioralCoverageScenarios(t *testing.T, family string) {
	t.Helper()
	scenarios := behavioralScenariosForFamily(family)
	if len(scenarios) == 0 {
		return
	}

	profiles, err := selectedProfilesFromDefaultConfig()
	if err != nil {
		t.Fatalf("select integration profiles: %v", err)
	}

	report := behaviorScenarioReport{Family: family}
	var records []evidenceRecord
	var failures []string

	localScenarios, profileScenarios := splitProfileScenarios(scenarios)
	for _, scenario := range localScenarios {
		record, err := runBehaviorScenario(t, integrationProfile{}, scenario)
		records = append(records, record)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	if len(profileScenarios) > 0 && len(profiles) == 0 {
		for _, scenario := range profileScenarios {
			records = append(records, blockedBehaviorRecord(scenario, "environment_availability", "no selected profile from default local config"))
		}
	} else {
		for _, profile := range profiles {
			if _, err := checkProfileReadiness(t, profile); err != nil {
				failures = append(failures, err.Error())
				for _, scenario := range profileScenarios {
					records = append(records, blockedBehaviorRecord(scenario, "environment_availability", err.Error()))
				}
				continue
			}
			report.Profiles = append(report.Profiles, profile)
			for _, scenario := range profileScenarios {
				record, err := runBehaviorScenario(t, profile, scenario)
				records = append(records, record)
				if err != nil {
					failures = append(failures, err.Error())
				}
			}
		}
	}

	report.Records = records
	report.Summary = summarizeBehaviorRecords(records)
	writeJSON(t, "behavior-"+sanitizeEvidenceName(family)+".json", report)
	if len(failures) > 0 {
		t.Fatalf("behavioral %s scenarios failed:\n%s", family, strings.Join(failures, "\n"))
	}
}

// runBehaviorScenario executes a single scenario through the built CLI and validates its declared output contract.
func runBehaviorScenario(t *testing.T, profile integrationProfile, scenario behaviorScenario) (evidenceRecord, error) {
	t.Helper()
	args, seed, err := materializeBehaviorArgs(t, profile, scenario)
	if err != nil {
		record := blockedBehaviorRecord(scenario, "harness_setup", err.Error())
		record.Profile = profile.Name
		record.CamundaVersion = profile.ExpectedVersion
		return record, err
	}

	var result commandResult
	if profile.Name != "" {
		result = runC8VoltForProfile(t, profile.Name, scenario.Name, args...)
	} else {
		result = runC8Volt(t, scenario.Name, args...)
	}

	record := commandEvidence(scenario.CommandPath, scenario.Name, result, "pass")
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), scenario.CoveredFlags...)
	record.OutputMode = scenario.OutputMode
	record.Behavior = scenario.Behavior
	record.VersionBehavior = scenario.VersionBehavior
	record.DataOwnership = append([]string(nil), scenario.DataOwnership...)
	record.Preview = scenario.Preview
	record.ConfirmedMutation = scenario.ConfirmedMutation
	if seed != nil {
		record.ResourceKeys = append(record.ResourceKeys, seed.ProcessInstanceKeys...)
		record.ResourceKeys = append(record.ResourceKeys, seed.ProcessDefinitionKeys...)
		record.ResourceKeys = append(record.ResourceKeys, seed.ResourceIDs...)
	}

	if err := validateBehaviorResult(scenario, profile, result); err != nil {
		record.Outcome = "fail"
		record.FailureClass = classifyBehaviorFailure(scenario, profile, result)
		return record, err
	}
	return record, nil
}

// materializeBehaviorArgs substitutes suite-owned seeded keys, fixture paths, and report paths into scenario args.
func materializeBehaviorArgs(t *testing.T, profile integrationProfile, scenario behaviorScenario) ([]string, *seededProfileData, error) {
	t.Helper()
	args := append([]string(nil), scenario.Args...)
	var seed *seededProfileData
	if scenario.RequiresSeed {
		if profile.Name == "" {
			return nil, nil, fmt.Errorf("%s requires a selected profile and seed data", scenario.Name)
		}
		data, records, cleanup, err := seedProfileData(t, profile)
		writeDataEvidence(t, scenario.Name+"-seed.json", seededDataReport{
			Marker:   suite.marker,
			Profiles: []seededProfileData{data},
			Records:  records,
			Cleanup:  cleanup,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("seed %s: %w", scenario.Name, err)
		}
		seed = &data
	}

	replacements := behaviorReplacements(t, profile, scenario.Name, seed)
	for i, arg := range args {
		args[i] = replaceScenarioTokens(arg, replacements)
	}
	return args, seed, nil
}

// behaviorReplacements returns dynamic values used by profile-backed scenario commands.
func behaviorReplacements(t *testing.T, profile integrationProfile, scenarioName string, seed *seededProfileData) map[string]string {
	t.Helper()
	reportDir := filepath.Join(suite.workDir, "reports")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("create behavior report directory: %v", err)
	}
	exportDir := filepath.Join(suite.workDir, "exports", sanitizeEvidenceName(profile.Name))
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		t.Fatalf("create behavior export directory: %v", err)
	}
	replacements := map[string]string{
		"{marker}":          suite.marker,
		"{impossible-bpmn}": "c8volt-it-impossible-" + suite.marker,
		"{report-json}":     filepath.Join(reportDir, sanitizeEvidenceName(scenarioName)+"-"+sanitizeEvidenceName(profile.Name)+".json"),
		"{report-md}":       filepath.Join(reportDir, sanitizeEvidenceName(scenarioName)+"-"+sanitizeEvidenceName(profile.Name)+".md"),
		"{export-dir}":      exportDir,
		"{vars-file}":       writeBehaviorVarsFile(t),
	}
	if seed == nil {
		return replacements
	}
	replacements["{pi-key}"] = firstString(seed.ProcessInstanceKeys)
	replacements["{another-pi-key}"] = stringAt(seed.ProcessInstanceKeys, 1)
	replacements["{pd-key}"] = firstString(seed.ProcessDefinitionKeys)
	replacements["{bpmn-process-id}"] = seed.BpmnProcessID
	replacements["{fixture}"] = seed.FixturePath
	replacements["{fixture-abs}"] = filepath.Join(suite.repoRoot, "embedded", filepath.FromSlash(seed.FixturePath))
	replacements["{resource-id}"] = firstString(seed.ResourceIDs)
	return replacements
}

// writeBehaviorVarsFile creates a reusable variables file for vars-file command coverage.
func writeBehaviorVarsFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(suite.workDir, "data", "behavior-vars-"+suite.marker+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create vars-file directory: %v", err)
	}
	payload := map[string]any{"c8voltITRunId": suite.marker, "behaviorScenario": true}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal behavior vars: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write behavior vars-file: %v", err)
	}
	return path
}

// replaceScenarioTokens expands dynamic tokens but keeps unresolved tokens visible for validation.
func replaceScenarioTokens(arg string, replacements map[string]string) string {
	for token, value := range replacements {
		if value != "" {
			arg = strings.ReplaceAll(arg, token, value)
		}
	}
	return arg
}

// validateBehaviorResult checks success, expected validation failures, output modes, and unsupported-version evidence.
func validateBehaviorResult(scenario behaviorScenario, profile integrationProfile, result commandResult) error {
	if scenario.WantFailureSubstring != "" {
		if result.Err == nil {
			return fmt.Errorf("%s succeeded, want validation failure containing %q", scenario.Name, scenario.WantFailureSubstring)
		}
		if isAllowedUnsupportedVersion(profile.ExpectedVersion, scenario.AllowUnsupportedVersions) && strings.Contains(strings.ToLower(result.Stderr), "unsupported") {
			return nil
		}
		combined := strings.TrimSpace(result.Stdout + "\n" + result.Stderr)
		if !strings.Contains(strings.ToLower(combined), strings.ToLower(scenario.WantFailureSubstring)) {
			return fmt.Errorf("%s failure did not contain %q: %s", scenario.Name, scenario.WantFailureSubstring, combined)
		}
		return nil
	}
	if result.Err != nil {
		if isAllowedUnsupportedVersion(profile.ExpectedVersion, scenario.AllowUnsupportedVersions) && strings.Contains(strings.ToLower(result.Stderr), "unsupported") {
			return nil
		}
		return fmt.Errorf("%s failed: %v; stderr: %s", scenario.Name, result.Err, strings.TrimSpace(result.Stderr))
	}
	if scenario.WantJSON {
		if err := validateJSONString(result.Stdout); err != nil {
			return fmt.Errorf("%s JSON output invalid: %w", scenario.Name, err)
		}
	}
	if scenario.WantKeysOnly {
		if err := validateKeysOnlyString(result.Stdout); err != nil {
			return fmt.Errorf("%s keys-only output invalid: %w", scenario.Name, err)
		}
	}
	if scenario.WantNonEmpty && strings.TrimSpace(result.Stdout) == "" && strings.TrimSpace(result.Stderr) == "" {
		return fmt.Errorf("%s produced empty output", scenario.Name)
	}
	return nil
}

// containsString reports whether values contains want exactly.
func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// validateJSONString provides non-testing JSON validation for scenario result checks.
func validateJSONString(output string) error {
	var payload any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return fmt.Errorf("expected valid JSON: %w; output: %s", err, output)
	}
	return nil
}

// validateKeysOnlyString validates the keys-only machine contract without failing the caller immediately.
func validateKeysOnlyString(output string) error {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil
	}
	for _, line := range strings.Split(trimmed, "\n") {
		if !isNumericString(strings.TrimSpace(line)) {
			return fmt.Errorf("line %q is not numeric", line)
		}
	}
	return nil
}

// classifyBehaviorFailure maps scenario failures to the durable evidence failure classes.
func classifyBehaviorFailure(scenario behaviorScenario, profile integrationProfile, result commandResult) string {
	if scenario.WantFailureSubstring != "" {
		return "product"
	}
	if isAllowedUnsupportedVersion(profile.ExpectedVersion, scenario.AllowUnsupportedVersions) && strings.Contains(strings.ToLower(result.Stderr), "unsupported") {
		return "missing_command_support"
	}
	if strings.Contains(strings.ToLower(result.Stderr), "connection") {
		return "environment_availability"
	}
	return "product"
}

// blockedBehaviorRecord records setup gaps without pretending that the command behavior executed.
func blockedBehaviorRecord(scenario behaviorScenario, failureClass string, reason string) evidenceRecord {
	now := timeNowUTC()
	return evidenceRecord{
		CommandPath:       scenario.CommandPath,
		ScenarioName:      scenario.Name,
		Arguments:         append([]string(nil), scenario.Args...),
		StartedAt:         now,
		FinishedAt:        now,
		CoveredFlags:      append([]string(nil), scenario.CoveredFlags...),
		OutputMode:        scenario.OutputMode,
		Behavior:          scenario.Behavior,
		VersionBehavior:   scenario.VersionBehavior,
		DataOwnership:     append([]string(nil), scenario.DataOwnership...),
		Preview:           scenario.Preview,
		ConfirmedMutation: scenario.ConfirmedMutation,
		Outcome:           "blocked",
		FailureClass:      failureClass,
		StderrPath:        reason,
	}
}

// timeNowUTC is a tiny seam for readable blocked records.
func timeNowUTC() time.Time {
	return time.Now().UTC()
}

// splitProfileScenarios separates local CLI assertions from real-cluster scenarios.
func splitProfileScenarios(scenarios []behaviorScenario) ([]behaviorScenario, []behaviorScenario) {
	var local []behaviorScenario
	var profile []behaviorScenario
	for _, scenario := range scenarios {
		if scenario.RequiresProfile || scenario.RequiresSeed {
			profile = append(profile, scenario)
		} else {
			local = append(local, scenario)
		}
	}
	return local, profile
}

// summarizeBehaviorRecords builds the per-family behavioral coverage report.
func summarizeBehaviorRecords(records []evidenceRecord) behaviorScenarioSummary {
	summary := behaviorScenarioSummary{
		FlagsCovered:       map[string][]string{},
		OutputModesCovered: map[string][]string{},
		BehaviorsCovered:   map[string][]string{},
		VersionBehaviors:   map[string][]string{},
		DataOwnership:      map[string][]string{},
	}
	commands := map[string]struct{}{}
	for _, record := range records {
		summary.ScenarioCount++
		commands[record.CommandPath] = struct{}{}
		summary.FlagsCovered[record.CommandPath] = append(summary.FlagsCovered[record.CommandPath], record.CoveredFlags...)
		if record.OutputMode != "" {
			summary.OutputModesCovered[record.CommandPath] = append(summary.OutputModesCovered[record.CommandPath], record.OutputMode)
		}
		if record.Behavior != "" {
			summary.BehaviorsCovered[record.CommandPath] = append(summary.BehaviorsCovered[record.CommandPath], record.Behavior)
		}
		if record.VersionBehavior != "" {
			summary.VersionBehaviors[record.CommandPath] = append(summary.VersionBehaviors[record.CommandPath], record.VersionBehavior)
		}
		summary.DataOwnership[record.CommandPath] = append(summary.DataOwnership[record.CommandPath], record.DataOwnership...)
		if record.Preview {
			summary.Previews = append(summary.Previews, record.CommandPath)
		}
		if record.ConfirmedMutation {
			summary.ConfirmedMutations = append(summary.ConfirmedMutations, record.CommandPath)
		}
		if record.Outcome == "blocked" || record.Outcome == "skipped" {
			summary.BlockedOrSkipped = append(summary.BlockedOrSkipped, record.CommandPath+":"+record.ScenarioName)
		}
	}
	for command := range commands {
		summary.CommandsCovered = append(summary.CommandsCovered, command)
	}
	summary.CommandsCovered = sortedStrings(summary.CommandsCovered)
	for command, flags := range summary.FlagsCovered {
		summary.FlagsCovered[command] = uniqueSortedStrings(flags)
	}
	for command, modes := range summary.OutputModesCovered {
		summary.OutputModesCovered[command] = uniqueSortedStrings(modes)
	}
	for command, behaviors := range summary.BehaviorsCovered {
		summary.BehaviorsCovered[command] = uniqueSortedStrings(behaviors)
	}
	for command, behaviors := range summary.VersionBehaviors {
		summary.VersionBehaviors[command] = uniqueSortedStrings(behaviors)
	}
	for command, ownership := range summary.DataOwnership {
		summary.DataOwnership[command] = uniqueSortedStrings(ownership)
	}
	summary.Previews = uniqueSortedStrings(summary.Previews)
	summary.ConfirmedMutations = uniqueSortedStrings(summary.ConfirmedMutations)
	summary.BlockedOrSkipped = uniqueSortedStrings(summary.BlockedOrSkipped)
	return summary
}

// uniqueSortedStrings removes duplicates before persisting summary fields.
func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return sortedStrings(out)
}

// hasBehavior checks whether a command's scenario list contains a behavior kind.
func hasBehavior(scenarios []behaviorScenario, behavior string) bool {
	for _, scenario := range scenarios {
		if scenario.Behavior == behavior {
			return true
		}
	}
	return false
}

// isLeafCommandPath treats command paths without children in the manifest as leaves.
func isLeafCommandPath(path string) bool {
	prefix := path + " "
	for other := range commandCoverageManifest {
		if strings.HasPrefix(other, prefix) {
			return false
		}
	}
	return true
}

// isAllowedUnsupportedVersion checks whether a failing scenario is expected for a profile version.
func isAllowedUnsupportedVersion(expected string, versions []string) bool {
	for _, version := range versions {
		if strings.Contains(expected, version) {
			return true
		}
	}
	return false
}

// behavioralScenariosForFamily returns the executable convergence scenarios owned by a family.
func behavioralScenariosForFamily(family string) []behaviorScenario {
	var scenarios []behaviorScenario
	for _, scenario := range allBehaviorScenarios() {
		if entry, ok := commandCoverageManifest[scenario.CommandPath]; ok && entry.Family == family {
			scenarios = append(scenarios, scenario)
		}
	}
	return scenarios
}

// allBehaviorScenarios declares the real or locally validating scenario catalog used by convergence coverage.
func allBehaviorScenarios() []behaviorScenario {
	var scenarios []behaviorScenario
	for path, entry := range commandCoverageManifest {
		if !isLeafCommandPath(path) {
			scenarios = append(scenarios, behaviorScenario{
				CommandPath:     path,
				Name:            "behavior-" + sanitizeEvidenceName(path) + "-no-argument",
				Args:            strings.Fields(path),
				CoveredFlags:    entry.Flags,
				OutputMode:      "one-line",
				Behavior:        behaviorDiscovery,
				VersionBehavior: "version-neutral",
				DataOwnership:   []string{"preexisting"},
				WantNonEmpty:    true,
			})
		}
	}
	scenarios = append(scenarios,
		behaviorScenario{CommandPath: "version", Name: "behavior-version-json", Args: []string{"--json", "version"}, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "version-neutral", DataOwnership: []string{"preexisting"}, WantJSON: true, CoveredFlags: commandCoverageManifest["version"].Flags},
		behaviorScenario{CommandPath: "capabilities", Name: "behavior-capabilities-json", Args: []string{"capabilities", "--json"}, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "version-neutral", DataOwnership: []string{"preexisting"}, WantJSON: true, CoveredFlags: commandCoverageManifest["capabilities"].Flags},
		behaviorScenario{CommandPath: "config show", Name: "behavior-config-show-template", Args: []string{"config", "show", "--template"}, OutputMode: "one-line", Behavior: behaviorSuccess, VersionBehavior: "version-neutral", DataOwnership: []string{"preexisting"}, WantNonEmpty: true, CoveredFlags: []string{"template"}},
		behaviorScenario{CommandPath: "config template", Name: "behavior-config-template", Args: []string{"config", "template"}, OutputMode: "one-line", Behavior: behaviorSuccess, VersionBehavior: "version-neutral", DataOwnership: []string{"preexisting"}, WantNonEmpty: true},
		behaviorScenario{CommandPath: "config validate", Name: "behavior-config-validate", Args: []string{"config", "validate"}, RequiresProfile: true, OutputMode: "one-line", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"preexisting"}},
		behaviorScenario{CommandPath: "config test-connection", Name: "behavior-config-test-connection-json", Args: []string{"--json", "config", "test-connection"}, RequiresProfile: true, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"preexisting"}, WantJSON: true},
	)
	scenarios = append(scenarios, dataSetupBehaviorScenarios()...)
	scenarios = append(scenarios, readBehaviorScenarios()...)
	scenarios = append(scenarios, mutationBehaviorScenarios()...)
	scenarios = append(scenarios, opsBehaviorScenarios()...)
	return scenarios
}

// dataSetupBehaviorScenarios covers deploy/embed/run setup commands with c8volt-created fixtures.
func dataSetupBehaviorScenarios() []behaviorScenario {
	return []behaviorScenario{
		{CommandPath: "embed list", Name: "behavior-embed-list-details-json", Args: []string{"--json", "embed", "list", "--details"}, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "version-neutral", DataOwnership: []string{"preexisting"}, WantJSON: true, CoveredFlags: []string{"details"}},
		{CommandPath: "embed deploy", Name: "behavior-embed-deploy-file-no-wait", Args: []string{"embed", "deploy", "--file", "{fixture}", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "mutated", "retained"}, ConfirmedMutation: true, CoveredFlags: []string{"file", "no-wait"}},
		{CommandPath: "embed export", Name: "behavior-embed-export-file-force", Args: []string{"embed", "export", "--file", "{fixture}", "--out", "{export-dir}", "--force"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "version-neutral-local-filesystem", DataOwnership: []string{"seeded", "mutated", "retained"}, ConfirmedMutation: true, CoveredFlags: []string{"file", "out", "force"}},
		{CommandPath: "deploy process-definition", Name: "behavior-deploy-process-definition-file-no-wait", Args: []string{"deploy", "process-definition", "--file", "{fixture-abs}", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "mutated", "retained"}, ConfirmedMutation: true, CoveredFlags: []string{"file", "no-wait"}},
		{CommandPath: "run process-instance", Name: "behavior-run-pi-json-count-vars-workers", Args: []string{"--json", "run", "pi", "--pd-key", "{pd-key}", "--count", "1", "--workers", "1", "--vars", `{"c8voltITRunId":"{marker}","behavior":"run"}`}, RequiresProfile: true, RequiresSeed: true, OutputMode: "json", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "mutated", "retained"}, WantJSON: true, ConfirmedMutation: true, CoveredFlags: []string{"pd-key", "count", "workers", "vars"}},
		{CommandPath: "run process-instance", Name: "behavior-run-pi-keys-only-no-wait", Args: []string{"--keys-only", "run", "pi", "--bpmn-process-id", "{bpmn-process-id}", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "keys-only", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "mutated", "retained"}, WantKeysOnly: true, ConfirmedMutation: true, CoveredFlags: []string{"bpmn-process-id", "no-wait"}},
	}
}

// readBehaviorScenarios covers real read/search/state commands against seeded and dirty cluster data.
func readBehaviorScenarios() []behaviorScenario {
	return []behaviorScenario{
		{CommandPath: "get cluster version", Name: "behavior-get-cluster-version-with-brokers", Args: []string{"get", "cluster", "version", "--with-brokers"}, RequiresProfile: true, OutputMode: "one-line", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"preexisting"}, WantNonEmpty: true, CoveredFlags: []string{"with-brokers"}},
		{CommandPath: "get cluster topology", Name: "behavior-get-cluster-topology-json", Args: []string{"--json", "get", "cluster", "topology"}, RequiresProfile: true, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"preexisting"}, WantJSON: true},
		{CommandPath: "get cluster license", Name: "behavior-get-cluster-license-json", Args: []string{"--json", "get", "cluster", "license"}, RequiresProfile: true, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"preexisting"}, WantJSON: true},
		{CommandPath: "get process-definition", Name: "behavior-get-pd-json-key-latest", Args: []string{"--json", "get", "pd", "--key", "{pd-key}"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, WantJSON: true, CoveredFlags: []string{"key", "latest", "bpmn-process-id", "pd-version", "pd-version-tag", "stat", "xml"}},
		{CommandPath: "get process-instance", Name: "behavior-get-pi-json-key-vars-elements-incidents", Args: []string{"--json", "get", "pi", "--key", "{pi-key}", "--with-vars", "--with-elements", "--with-incidents"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, WantJSON: true, CoveredFlags: commandCoverageManifest["get process-instance"].Flags},
		{CommandPath: "get element", Name: "behavior-get-element-json-pi-key-limit", Args: []string{"--json", "get", "element", "--pi-key", "{pi-key}", "--limit", "10"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, WantJSON: true, CoveredFlags: commandCoverageManifest["get element"].Flags},
		{CommandPath: "get incident", Name: "behavior-get-incident-keys-only-no-match", Args: []string{"--keys-only", "get", "incident", "--state", "active", "--bpmn-process-id", "{bpmn-process-id}", "--limit", "1"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "keys-only", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, WantKeysOnly: true, CoveredFlags: commandCoverageManifest["get incident"].Flags},
		{CommandPath: "get job", Name: "behavior-get-job-json-pi-key", Args: []string{"--json", "get", "job", "--pi-key", "{pi-key}", "--limit", "1"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, WantJSON: true, CoveredFlags: commandCoverageManifest["get job"].Flags},
		{CommandPath: "get resource", Name: "behavior-get-resource-id-validation-gap", Args: []string{"get", "resource", "--id", "{resource-id}"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorValidation, VersionBehavior: "resource-id-not-returned-by-deployment-output", DataOwnership: []string{"seeded", "preexisting"}, CoveredFlags: []string{"id"}, WantFailureSubstring: "bad request"},
		{CommandPath: "get tenant", Name: "behavior-get-tenant-filter", Args: []string{"get", "tenant", "--filter", "c8volt-it-no-match"}, RequiresProfile: true, OutputMode: "one-line", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"preexisting"}, CoveredFlags: []string{"filter", "key"}},
		{CommandPath: "walk process-instance", Name: "behavior-walk-pi-with-vars-elements-incidents-flat", Args: []string{"walk", "pi", "--key", "{pi-key}", "--with-vars", "--with-elements", "--with-incidents", "--flat"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated-with-listener-proposals", DataOwnership: []string{"seeded", "preexisting"}, CoveredFlags: commandCoverageManifest["walk process-instance"].Flags},
		{CommandPath: "expect process-instance", Name: "behavior-expect-pi-active", Args: []string{"expect", "pi", "--key", "{pi-key}", "--state", "active"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded"}, CoveredFlags: []string{"key", "state"}},
	}
}

// mutationBehaviorScenarios covers destructive preview and confirmed mutation paths for non-ops families.
func mutationBehaviorScenarios() []behaviorScenario {
	return []behaviorScenario{
		{CommandPath: "update process-instance", Name: "behavior-update-pi-dry-run-vars", Args: []string{"update", "pi", "--key", "{pi-key}", "--vars", `{"behavior":"update-preview"}`, "--dry-run"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "unsupported-before-mutation-on-8.7", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: []string{"key", "vars", "dry-run", "workers", "fail-fast", "no-worker-limit", "no-wait", "vars-file"}, AllowUnsupportedVersions: []string{"8.7"}},
		{CommandPath: "update process-instance", Name: "behavior-update-pi-confirmed-no-wait", Args: []string{"--automation", "update", "pi", "--key", "{pi-key}", "--vars-file", "{vars-file}", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "unsupported-before-mutation-on-8.7", DataOwnership: []string{"seeded", "mutated", "retained"}, ConfirmedMutation: true, CoveredFlags: []string{"key", "vars-file", "no-wait"}, AllowUnsupportedVersions: []string{"8.7"}},
		{CommandPath: "update job", Name: "behavior-update-job-dry-run-validation-or-unsupported", Args: []string{"--json", "update", "job", "--key", "1", "--retries", "3", "--timeout", "1m", "--dry-run"}, RequiresProfile: true, OutputMode: "json", Behavior: behaviorPreview, VersionBehavior: "unsupported-before-mutation-on-8.7-or-not-found", DataOwnership: []string{"preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["update job"].Flags, WantFailureSubstring: "not found", AllowUnsupportedVersions: []string{"8.7"}},
		{CommandPath: "update job", Name: "behavior-update-job-confirmed-validation-or-unsupported", Args: []string{"--automation", "update", "job", "--key", "1", "--fail", "--retries", "0", "--message", "c8volt-it", "--no-wait"}, RequiresProfile: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "unsupported-before-mutation-on-8.7-or-not-found", DataOwnership: []string{"preexisting", "mutated"}, ConfirmedMutation: true, CoveredFlags: []string{"key", "fail", "retries", "message", "no-wait"}, WantFailureSubstring: "not found", AllowUnsupportedVersions: []string{"8.7"}},
		{CommandPath: "cancel process-instance", Name: "behavior-cancel-pi-dry-run-key", Args: []string{"cancel", "pi", "--key", "{pi-key}", "--dry-run", "--workers", "1"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["cancel process-instance"].Flags},
		{CommandPath: "cancel process-instance", Name: "behavior-cancel-pi-confirmed-force-no-wait", Args: []string{"--automation", "--auto-confirm", "cancel", "pi", "--key", "{pi-key}", "--force", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "mutated", "retained"}, ConfirmedMutation: true, CoveredFlags: []string{"key", "force", "no-wait"}},
		{CommandPath: "delete process-instance", Name: "behavior-delete-pi-dry-run-key-force", Args: []string{"delete", "pi", "--key", "{pi-key}", "--force", "--dry-run"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["delete process-instance"].Flags},
		{CommandPath: "delete process-instance", Name: "behavior-delete-pi-confirmed-force-no-wait", Args: []string{"--automation", "--auto-confirm", "delete", "pi", "--key", "{pi-key}", "--force", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "mutated", "cleanup_failed"}, ConfirmedMutation: true, CoveredFlags: []string{"key", "force", "no-wait"}},
		{CommandPath: "delete process-definition", Name: "behavior-delete-pd-dry-run-key-force", Args: []string{"delete", "pd", "--key", "{pd-key}", "--force", "--dry-run"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "process-definition-delete-version-specific", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["delete process-definition"].Flags, AllowUnsupportedVersions: []string{"8.7", "8.8"}},
		{CommandPath: "delete process-definition", Name: "behavior-delete-pd-confirmed-force-no-wait", Args: []string{"--automation", "--auto-confirm", "delete", "pd", "--key", "{pd-key}", "--force", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "process-definition-delete-version-specific", DataOwnership: []string{"seeded", "mutated", "cleanup_failed"}, ConfirmedMutation: true, CoveredFlags: []string{"key", "force", "no-wait"}, AllowUnsupportedVersions: []string{"8.7", "8.8"}},
		{CommandPath: "resolve process-instance", Name: "behavior-resolve-pi-dry-run", Args: []string{"resolve", "pi", "--key", "{pi-key}", "--dry-run"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["resolve process-instance"].Flags},
		{CommandPath: "resolve process-instance", Name: "behavior-resolve-pi-confirmed-no-wait", Args: []string{"--automation", "--auto-confirm", "resolve", "pi", "--key", "{pi-key}", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "mutated", "retained"}, ConfirmedMutation: true, CoveredFlags: []string{"key", "no-wait"}},
		{CommandPath: "resolve incident", Name: "behavior-resolve-incident-dry-run-no-match", Args: []string{"resolve", "incident", "--key", "2251799813685249", "--dry-run"}, RequiresProfile: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["resolve incident"].Flags, WantFailureSubstring: "not found"},
		{CommandPath: "resolve incident", Name: "behavior-resolve-incident-confirmed-no-match", Args: []string{"--automation", "--auto-confirm", "resolve", "incident", "--key", "2251799813685249", "--no-wait"}, RequiresProfile: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"preexisting", "mutated"}, ConfirmedMutation: true, CoveredFlags: []string{"key", "no-wait"}, WantFailureSubstring: "not found"},
	}
}

// opsBehaviorScenarios covers high-level ops preview, report, output-mode, and confirmed paths.
func opsBehaviorScenarios() []behaviorScenario {
	return []behaviorScenario{
		{CommandPath: "ops analyse slow-process-instances", Name: "behavior-ops-analyse-json-timeline", Args: []string{"--json", "ops", "analyse", "slow-process-instances", "--key", "{pi-key}", "--with-full-timeline", "--dur-longer", "1s", "--dur-element-longer", "1s"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "json", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated-with-proposals", DataOwnership: []string{"seeded", "preexisting"}, WantJSON: true, CoveredFlags: commandCoverageManifest["ops analyse slow-process-instances"].Flags},
		{CommandPath: "ops analyse slow-process-instances", Name: "behavior-ops-analyse-keys-only-filter", Args: []string{"--keys-only", "ops", "analyse", "spi", "--bpmn-process-id", "{bpmn-process-id}", "--limit", "1"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "keys-only", Behavior: behaviorSuccess, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "preexisting"}, WantKeysOnly: true, CoveredFlags: []string{"bpmn-process-id", "limit"}},
		{CommandPath: "ops execute smoke-test", Name: "behavior-ops-execute-smoke-test-dry-run-report", Args: []string{"ops", "execute", "smoke-test", "--dry-run", "--count", "1", "--workers", "1", "--report-file", "{report-json}", "--report-format", "json"}, RequiresProfile: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["ops execute smoke-test"].Flags},
		{CommandPath: "ops execute smoke-test", Name: "behavior-ops-execute-smoke-test-confirmed", Args: []string{"--automation", "ops", "execute", "smoke-test", "--count", "1", "--no-wait", "--no-cleanup", "--report-file", "{report-md}"}, RequiresProfile: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated", DataOwnership: []string{"seeded", "mutated", "retained"}, ConfirmedMutation: true, CoveredFlags: []string{"count", "no-wait", "no-cleanup", "report-file"}},
		{CommandPath: "ops execute retention-policy", Name: "behavior-ops-retention-dry-run-no-match", Args: []string{"ops", "execute", "retention-policy", "--bpmn-process-id", "{bpmn-process-id}", "--retention-days", "9999", "--dry-run", "--report-file", "{report-json}", "--report-format", "json"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["ops execute retention-policy"].Flags},
		{CommandPath: "ops execute retention-policy", Name: "behavior-ops-retention-confirmed-no-match", Args: []string{"--automation", "--auto-confirm", "ops", "execute", "retention-policy", "--bpmn-process-id", "{bpmn-process-id}", "--retention-days", "9999", "--force", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"seeded", "preexisting", "mutated"}, ConfirmedMutation: true, CoveredFlags: []string{"bpmn-process-id", "retention-days", "force", "no-wait"}},
		{CommandPath: "ops purge all-process-definitions", Name: "behavior-ops-purge-all-pds-dry-run", Args: []string{"ops", "purge", "all-process-definitions", "--bpmn-process-id", "{bpmn-process-id}", "--latest", "--dry-run", "--limit", "1", "--report-file", "{report-json}", "--report-format", "json"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "8.9-supported-earlier-unsupported", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["ops purge all-process-definitions"].Flags, AllowUnsupportedVersions: []string{"8.7", "8.8"}},
		{CommandPath: "ops purge all-process-definitions", Name: "behavior-ops-purge-all-pds-confirmed", Args: []string{"--automation", "--auto-confirm", "ops", "purge", "all-process-definitions", "--bpmn-process-id", "{bpmn-process-id}", "--latest", "--force", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "8.9-supported-earlier-unsupported", DataOwnership: []string{"seeded", "preexisting", "mutated", "cleanup_failed"}, ConfirmedMutation: true, CoveredFlags: []string{"bpmn-process-id", "latest", "force", "no-wait"}, AllowUnsupportedVersions: []string{"8.7", "8.8"}},
		{CommandPath: "ops purge orphan-process-instances", Name: "behavior-ops-purge-orphans-dry-run", Args: []string{"ops", "purge", "orphan-process-instances", "--bpmn-process-id", "{bpmn-process-id}", "--dry-run", "--limit", "1", "--report-file", "{report-json}", "--report-format", "json"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["ops purge orphan-process-instances"].Flags},
		{CommandPath: "ops purge orphan-process-instances", Name: "behavior-ops-purge-orphans-confirmed", Args: []string{"--automation", "--auto-confirm", "ops", "purge", "orphan-process-instances", "--bpmn-process-id", "{bpmn-process-id}", "--force", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"seeded", "preexisting", "mutated"}, ConfirmedMutation: true, CoveredFlags: []string{"bpmn-process-id", "force", "no-wait"}},
		{CommandPath: "ops purge process-instances-with-incidents", Name: "behavior-ops-purge-piwi-dry-run", Args: []string{"ops", "purge", "process-instances-with-incidents", "--bpmn-process-id", "{bpmn-process-id}", "--dry-run", "--limit", "1", "--report-file", "{report-json}", "--report-format", "json"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["ops purge process-instances-with-incidents"].Flags},
		{CommandPath: "ops purge process-instances-with-incidents", Name: "behavior-ops-purge-piwi-confirmed", Args: []string{"--automation", "--auto-confirm", "ops", "purge", "process-instances-with-incidents", "--bpmn-process-id", "{bpmn-process-id}", "--force", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"seeded", "preexisting", "mutated"}, ConfirmedMutation: true, CoveredFlags: []string{"bpmn-process-id", "force", "no-wait"}},
		{CommandPath: "ops repair incident", Name: "behavior-ops-repair-incident-dry-run-no-match", Args: []string{"ops", "repair", "incident", "--bpmn-process-id", "{bpmn-process-id}", "--state", "active", "--limit", "1", "--dry-run", "--report-file", "{report-json}", "--report-format", "json"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["ops repair incident"].Flags},
		{CommandPath: "ops repair incident", Name: "behavior-ops-repair-incident-confirmed-no-match", Args: []string{"--automation", "--auto-confirm", "ops", "repair", "incident", "--bpmn-process-id", "{bpmn-process-id}", "--state", "active", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated-no-match", DataOwnership: []string{"seeded", "preexisting", "mutated"}, ConfirmedMutation: true, CoveredFlags: []string{"bpmn-process-id", "state", "no-wait"}},
		{CommandPath: "ops repair process-instance", Name: "behavior-ops-repair-pi-dry-run", Args: []string{"ops", "repair", "process-instance", "--key", "{pi-key}", "--vars", `{"behavior":"repair-preview"}`, "--retries", "1", "--dry-run", "--report-file", "{report-json}", "--report-format", "json"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorPreview, VersionBehavior: "selected-profile-version-gated-with-proposals", DataOwnership: []string{"seeded", "preexisting"}, Preview: true, CoveredFlags: commandCoverageManifest["ops repair process-instance"].Flags},
		{CommandPath: "ops repair process-instance", Name: "behavior-ops-repair-pi-confirmed-no-wait", Args: []string{"--automation", "--auto-confirm", "ops", "repair", "process-instance", "--key", "{pi-key}", "--vars-file", "{vars-file}", "--retries", "1", "--no-wait"}, RequiresProfile: true, RequiresSeed: true, OutputMode: "one-line", Behavior: behaviorConfirmedMutation, VersionBehavior: "selected-profile-version-gated-with-proposals", DataOwnership: []string{"seeded", "mutated", "retained"}, ConfirmedMutation: true, CoveredFlags: []string{"key", "vars-file", "retries", "no-wait"}},
	}
}
