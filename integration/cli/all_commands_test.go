// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const expectedCommandInventoryCount = 55

type capabilityDocument struct {
	Command  string              `json:"command"`
	Version  string              `json:"version"`
	Commands []commandCapability `json:"commands"`
}

type commandCapability struct {
	Path        string               `json:"path"`
	Aliases     []string             `json:"aliases,omitempty"`
	Mutation    string               `json:"mutation"`
	OutputModes []outputModeContract `json:"outputModes"`
	Flags       []flagContract       `json:"flags,omitempty"`
	Children    []commandCapability  `json:"children,omitempty"`
}

type outputModeContract struct {
	Name      string `json:"name"`
	Supported bool   `json:"supported"`
}

type flagContract struct {
	Name string `json:"name"`
}

type coverageEntry struct {
	Path          string   `json:"path"`
	Family        string   `json:"family"`
	ScenarioOwner string   `json:"scenarioOwner"`
	Aliases       []string `json:"aliases"`
	Flags         []string `json:"flags"`
	OutputModes   []string `json:"outputModes"`
	Destructive   bool     `json:"destructive"`
}

type coverageReport struct {
	ExpectedCount int      `json:"expectedCount"`
	ActualCount   int      `json:"actualCount"`
	MissingPaths  []string `json:"missingPaths"`
	StalePaths    []string `json:"stalePaths"`
	Mismatches    []string `json:"mismatches"`
}

type familyCoverageReport struct {
	Family  string                `json:"family"`
	Entries []coverageEntry       `json:"entries"`
	Records []evidenceRecord      `json:"records"`
	Summary familyCoverageSummary `json:"summary"`
}

type familyCoverageSummary struct {
	CommandCount     int      `json:"commandCount"`
	DestructivePaths []string `json:"destructivePaths,omitempty"`
	OutputModes      []string `json:"outputModes,omitempty"`
}

var commandCoverageManifest = map[string]coverageEntry{
	"cancel":                             {Path: "cancel", Family: "cancel", ScenarioOwner: "integration/cli/cancel_test.go", Aliases: []string{"abort", "c", "cn", "stop"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"cancel process-instance":            {Path: "cancel process-instance", Family: "cancel", ScenarioOwner: "integration/cli/cancel_test.go", Aliases: []string{"pi"}, Flags: []string{"batch-size", "bpmn-process-id", "dry-run", "end-date-after", "end-date-before", "end-date-newer-days", "end-date-older-days", "fail-fast", "force", "key", "limit", "no-state-check", "no-wait", "no-worker-limit", "pd-version", "pd-version-tag", "start-date-after", "start-date-before", "start-date-newer-days", "start-date-older-days", "state", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"capabilities":                       {Path: "capabilities", Family: "capabilities", ScenarioOwner: "integration/cli/all_commands_test.go", Aliases: []string{}, Flags: []string{"auto-confirm", "automation", "config", "debug", "help", "json", "keys-only", "log-level", "no-indicator", "profile", "quiet", "tenant", "timeout", "verbose"}, OutputModes: []string{"json", "one-line"}, Destructive: false},
	"config":                             {Path: "config", Family: "config", ScenarioOwner: "integration/cli/config_test.go", Aliases: []string{"cfg"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"config show":                        {Path: "config show", Family: "config", ScenarioOwner: "integration/cli/config_test.go", Aliases: []string{}, Flags: []string{"template", "validate"}, OutputModes: []string{"one-line"}, Destructive: false},
	"config template":                    {Path: "config template", Family: "config", ScenarioOwner: "integration/cli/config_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"config test-connection":             {Path: "config test-connection", Family: "config", ScenarioOwner: "integration/cli/config_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"json"}, Destructive: false},
	"config validate":                    {Path: "config validate", Family: "config", ScenarioOwner: "integration/cli/config_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"delete":                             {Path: "delete", Family: "delete", ScenarioOwner: "integration/cli/delete_test.go", Aliases: []string{"d", "del", "remove", "rm"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"delete process-definition":          {Path: "delete process-definition", Family: "delete", ScenarioOwner: "integration/cli/delete_test.go", Aliases: []string{"pd"}, Flags: []string{"bpmn-process-id", "dry-run", "fail-fast", "force", "key", "latest", "no-state-check", "no-wait", "no-worker-limit", "pd-version", "pd-version-tag", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"delete process-instance":            {Path: "delete process-instance", Family: "delete", ScenarioOwner: "integration/cli/delete_test.go", Aliases: []string{"pi"}, Flags: []string{"batch-size", "bpmn-process-id", "dry-run", "end-date-after", "end-date-before", "end-date-newer-days", "end-date-older-days", "fail-fast", "force", "key", "limit", "no-state-check", "no-wait", "no-worker-limit", "pd-version", "pd-version-tag", "start-date-after", "start-date-before", "start-date-newer-days", "start-date-older-days", "state", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"deploy":                             {Path: "deploy", Family: "deploy", ScenarioOwner: "integration/cli/deploy_embed_run_test.go", Aliases: []string{"dep"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"deploy process-definition":          {Path: "deploy process-definition", Family: "deploy", ScenarioOwner: "integration/cli/deploy_embed_run_test.go", Aliases: []string{"pd"}, Flags: []string{"file", "no-wait", "run"}, OutputModes: []string{"one-line"}, Destructive: true},
	"embed":                              {Path: "embed", Family: "embed", ScenarioOwner: "integration/cli/deploy_embed_run_test.go", Aliases: []string{"em", "emb"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"embed deploy":                       {Path: "embed deploy", Family: "embed", ScenarioOwner: "integration/cli/deploy_embed_run_test.go", Aliases: []string{"dep"}, Flags: []string{"all", "file", "no-wait", "run"}, OutputModes: []string{"one-line"}, Destructive: true},
	"embed export":                       {Path: "embed export", Family: "embed", ScenarioOwner: "integration/cli/deploy_embed_run_test.go", Aliases: []string{"exp", "extract"}, Flags: []string{"all", "file", "force", "out"}, OutputModes: []string{"one-line"}, Destructive: true},
	"embed list":                         {Path: "embed list", Family: "embed", ScenarioOwner: "integration/cli/deploy_embed_run_test.go", Aliases: []string{"ls"}, Flags: []string{"details"}, OutputModes: []string{"json"}, Destructive: false},
	"expect":                             {Path: "expect", Family: "expect", ScenarioOwner: "integration/cli/expect_resolve_test.go", Aliases: []string{"e", "exp"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"expect process-instance":            {Path: "expect process-instance", Family: "expect", ScenarioOwner: "integration/cli/expect_resolve_test.go", Aliases: []string{"pi"}, Flags: []string{"fail-fast", "incident", "key", "no-worker-limit", "state", "workers"}, OutputModes: []string{"one-line"}, Destructive: false},
	"get":                                {Path: "get", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{"g", "read"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"get cluster":                        {Path: "get cluster", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"get cluster license":                {Path: "get cluster license", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{"licence"}, Flags: []string{}, OutputModes: []string{"json"}, Destructive: false},
	"get cluster topology":               {Path: "get cluster topology", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"json"}, Destructive: false},
	"get cluster version":                {Path: "get cluster version", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{}, Flags: []string{"with-brokers"}, OutputModes: []string{"json", "one-line"}, Destructive: false},
	"get element":                        {Path: "get element", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{"ei"}, Flags: []string{"batch-size", "bpmn-process-id", "element-id", "key", "limit", "pd-key", "pi-key", "state", "total", "type", "with-listeners"}, OutputModes: []string{"one-line"}, Destructive: false},
	"get incident":                       {Path: "get incident", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{"inc", "incidents"}, Flags: []string{"batch-size", "bpmn-process-id", "creation-time-after", "creation-time-before", "creation-time-newer-days", "creation-time-older-days", "element-id", "element-instance-key", "error-message", "error-message-limit", "error-type", "fail-fast", "key", "limit", "no-worker-limit", "pd-key", "pi-key", "pi-keys-only", "root-key", "state", "total", "with-no-error-message", "workers"}, OutputModes: []string{"one-line"}, Destructive: false},
	"get job":                            {Path: "get job", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{}, Flags: []string{"batch-size", "element-id", "element-instance-key", "error-message-limit", "key", "kind", "limit", "listener-event-type", "pi-key", "retries", "state", "total", "type", "worker"}, OutputModes: []string{"one-line"}, Destructive: false},
	"get process-definition":             {Path: "get process-definition", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{"pd", "pds"}, Flags: []string{"bpmn-process-id", "key", "latest", "pd-version", "pd-version-tag", "stat", "xml"}, OutputModes: []string{"json"}, Destructive: false},
	"get process-instance":               {Path: "get process-instance", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{"pi", "pis", "process-instances"}, Flags: []string{"batch-size", "bpmn-process-id", "children-only", "direct-incidents-only", "end-date-after", "end-date-before", "end-date-newer-days", "end-date-older-days", "fail-fast", "has-user-tasks", "incident-error-message", "incident-error-type", "incident-message-limit", "incident-state", "incidents-only", "key", "limit", "no-incidents-only", "no-worker-limit", "orphan-children-only", "parent-key", "pd-key", "pd-version", "pd-version-tag", "roots-only", "start-date-after", "start-date-before", "start-date-newer-days", "start-date-older-days", "state", "total", "var", "var-exists", "var-like", "var-value-limit", "with-elements", "with-incidents", "with-listeners", "with-vars", "workers"}, OutputModes: []string{"one-line"}, Destructive: false},
	"get resource":                       {Path: "get resource", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{"r"}, Flags: []string{"id"}, OutputModes: []string{"one-line"}, Destructive: false},
	"get tenant":                         {Path: "get tenant", Family: "get", ScenarioOwner: "integration/cli/get_test.go", Aliases: []string{"tenants"}, Flags: []string{"filter", "key"}, OutputModes: []string{"one-line"}, Destructive: false},
	"ops":                                {Path: "ops", Family: "ops", ScenarioOwner: "integration/cli/all_commands_test.go", Aliases: []string{"operations"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"ops analyse":                        {Path: "ops analyse", Family: "ops analyse", ScenarioOwner: "integration/cli/ops_analyse_test.go", Aliases: []string{"analyze"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"ops analyse slow-process-instances": {Path: "ops analyse slow-process-instances", Family: "ops analyse", ScenarioOwner: "integration/cli/ops_analyse_test.go", Aliases: []string{"slow-pi", "spi"}, Flags: []string{"batch-size", "bpmn-process-id", "dur-element-longer", "dur-longer", "element-id", "element-state", "end-date-after", "end-date-before", "key", "limit", "no-incidents-only", "pd-key", "start-date-after", "start-date-before", "state", "type", "with-full-timeline", "with-listeners"}, OutputModes: []string{"json", "keys-only", "one-line"}, Destructive: false},
	"ops execute":                        {Path: "ops execute", Family: "ops execute", ScenarioOwner: "integration/cli/ops_execute_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"ops execute retention-policy":       {Path: "ops execute retention-policy", Family: "ops execute", ScenarioOwner: "integration/cli/ops_execute_test.go", Aliases: []string{"ret-pol", "rp"}, Flags: []string{"batch-size", "bpmn-process-id", "children-only", "dry-run", "fail-fast", "force", "incidents-only", "key", "limit", "no-incidents-only", "no-state-check", "no-wait", "no-worker-limit", "parent-key", "pd-key", "pd-version", "pd-version-tag", "report-file", "report-format", "retention-days", "roots-only", "state", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"ops execute smoke-test":             {Path: "ops execute smoke-test", Family: "ops execute", ScenarioOwner: "integration/cli/ops_execute_test.go", Aliases: []string{"st"}, Flags: []string{"count", "dry-run", "fail-fast", "no-cleanup", "no-wait", "no-worker-limit", "report-file", "report-format", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"ops purge":                          {Path: "ops purge", Family: "ops purge", ScenarioOwner: "integration/cli/ops_purge_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"ops purge all-process-definitions":  {Path: "ops purge all-process-definitions", Family: "ops purge", ScenarioOwner: "integration/cli/ops_purge_test.go", Aliases: []string{"all-pds", "apd"}, Flags: []string{"batch-size", "bpmn-process-id", "dry-run", "fail-fast", "force", "key", "latest", "limit", "no-wait", "no-worker-limit", "pd-version", "pd-version-tag", "report-file", "report-format", "workers"}, OutputModes: []string{"json", "one-line"}, Destructive: true},
	"ops purge orphan-process-instances": {Path: "ops purge orphan-process-instances", Family: "ops purge", ScenarioOwner: "integration/cli/ops_purge_test.go", Aliases: []string{"opi", "orphan-pi"}, Flags: []string{"batch-size", "bpmn-process-id", "dry-run", "end-date-after", "end-date-before", "end-date-newer-days", "end-date-older-days", "fail-fast", "force", "incidents-only", "limit", "no-incidents-only", "no-wait", "no-worker-limit", "parent-key", "pd-key", "pd-version", "pd-version-tag", "report-file", "report-format", "start-date-after", "start-date-before", "start-date-newer-days", "start-date-older-days", "state", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"ops purge process-instances-with-incidents": {Path: "ops purge process-instances-with-incidents", Family: "ops purge", ScenarioOwner: "integration/cli/ops_purge_test.go", Aliases: []string{"pi-with-incidents", "piwi"}, Flags: []string{"batch-size", "bpmn-process-id", "creation-time-after", "creation-time-before", "creation-time-newer-days", "creation-time-older-days", "dry-run", "element-id", "element-instance-key", "error-message", "error-type", "fail-fast", "force", "inc-key", "limit", "no-wait", "no-worker-limit", "pd-key", "pi-key", "report-file", "report-format", "root-key", "state", "workers"}, OutputModes: []string{"json", "one-line"}, Destructive: true},
	"ops repair":                  {Path: "ops repair", Family: "ops repair", ScenarioOwner: "integration/cli/ops_repair_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"ops repair incident":         {Path: "ops repair incident", Family: "ops repair", ScenarioOwner: "integration/cli/ops_repair_test.go", Aliases: []string{"inc"}, Flags: []string{"batch-size", "bpmn-process-id", "creation-time-after", "creation-time-before", "creation-time-newer-days", "creation-time-older-days", "dry-run", "element-id", "element-instance-key", "error-message", "error-type", "fail-fast", "job-timeout", "key", "limit", "no-wait", "no-worker-limit", "pd-key", "pi-key", "report-file", "report-format", "retries", "root-key", "state", "vars", "vars-file", "workers"}, OutputModes: []string{"json", "one-line"}, Destructive: true},
	"ops repair process-instance": {Path: "ops repair process-instance", Family: "ops repair", ScenarioOwner: "integration/cli/ops_repair_test.go", Aliases: []string{"pi", "pis", "process-instances"}, Flags: []string{"batch-size", "bpmn-process-id", "children-only", "direct-incidents-only", "dry-run", "end-date-after", "end-date-before", "end-date-newer-days", "end-date-older-days", "fail-fast", "incident-error-message", "incident-error-type", "incident-state", "job-timeout", "key", "limit", "no-wait", "no-worker-limit", "parent-key", "pd-key", "pd-version", "pd-version-tag", "report-file", "report-format", "retries", "roots-only", "start-date-after", "start-date-before", "start-date-newer-days", "start-date-older-days", "state", "vars", "vars-file", "workers"}, OutputModes: []string{"json", "one-line"}, Destructive: true},
	"resolve":                     {Path: "resolve", Family: "resolve", ScenarioOwner: "integration/cli/expect_resolve_test.go", Aliases: []string{"res"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"resolve incident":            {Path: "resolve incident", Family: "resolve", ScenarioOwner: "integration/cli/expect_resolve_test.go", Aliases: []string{"inc"}, Flags: []string{"dry-run", "fail-fast", "key", "no-wait", "no-worker-limit", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"resolve process-instance":    {Path: "resolve process-instance", Family: "resolve", ScenarioOwner: "integration/cli/expect_resolve_test.go", Aliases: []string{"pi"}, Flags: []string{"dry-run", "fail-fast", "key", "no-wait", "no-worker-limit", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"run":                         {Path: "run", Family: "run", ScenarioOwner: "integration/cli/deploy_embed_run_test.go", Aliases: []string{"r"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"run process-instance":        {Path: "run process-instance", Family: "run", ScenarioOwner: "integration/cli/deploy_embed_run_test.go", Aliases: []string{"pi"}, Flags: []string{"bpmn-process-id", "count", "fail-fast", "no-wait", "no-worker-limit", "pd-key", "pd-version", "vars", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"update":                      {Path: "update", Family: "update", ScenarioOwner: "integration/cli/update_test.go", Aliases: []string{"u"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: true},
	"update job":                  {Path: "update job", Family: "update", ScenarioOwner: "integration/cli/update_test.go", Aliases: []string{}, Flags: []string{"complete", "dry-run", "fail", "key", "message", "no-wait", "retries", "retry-backoff", "throw-bpmn-error", "timeout", "vars"}, OutputModes: []string{"json", "one-line"}, Destructive: true},
	"update process-instance":     {Path: "update process-instance", Family: "update", ScenarioOwner: "integration/cli/update_test.go", Aliases: []string{"pi"}, Flags: []string{"dry-run", "fail-fast", "key", "no-wait", "no-worker-limit", "vars", "vars-file", "workers"}, OutputModes: []string{"one-line"}, Destructive: true},
	"version":                     {Path: "version", Family: "version", ScenarioOwner: "integration/cli/config_test.go", Aliases: []string{}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"walk":                        {Path: "walk", Family: "walk", ScenarioOwner: "integration/cli/walk_test.go", Aliases: []string{"traverse", "w"}, Flags: []string{}, OutputModes: []string{"one-line"}, Destructive: false},
	"walk process-instance":       {Path: "walk process-instance", Family: "walk", ScenarioOwner: "integration/cli/walk_test.go", Aliases: []string{"pi", "pis"}, Flags: []string{"children", "flat", "incident-message-limit", "incident-state", "key", "parent", "var-value-limit", "with-elements", "with-incidents", "with-listeners", "with-vars"}, OutputModes: []string{"one-line"}, Destructive: false},
}

func TestCommandInventory(t *testing.T) {
	result := runC8Volt(t, "command-inventory", "capabilities", "--json")
	if result.Err != nil {
		t.Fatalf("capabilities --json failed: %v\nstderr:\n%s", result.Err, result.Stderr)
	}

	var doc capabilityDocument
	if err := json.Unmarshal([]byte(result.Stdout), &doc); err != nil {
		t.Fatalf("unmarshal capabilities output: %v\nstdout:\n%s", err, result.Stdout)
	}
	writeJSON(t, "inventory.json", doc)

	capabilities := flattenCapabilities(doc.Commands)
	report := validateCoverageManifest(capabilities, commandCoverageManifest)
	writeJSON(t, "coverage.json", report)

	if report.ActualCount != expectedCommandInventoryCount {
		t.Fatalf("command inventory count = %d, want %d", report.ActualCount, expectedCommandInventoryCount)
	}
	if len(report.MissingPaths) > 0 || len(report.StalePaths) > 0 || len(report.Mismatches) > 0 {
		t.Fatalf("coverage manifest drift:\nmissing: %v\nstale: %v\nmismatches:\n%s", report.MissingPaths, report.StalePaths, strings.Join(report.Mismatches, "\n"))
	}
}

// TestCoreBehavioralCoverage records real executable coverage for root-level read-only command families.
func TestCoreBehavioralCoverage(t *testing.T) {
	runBehavioralCoverageScenarios(t, "capabilities")
	runBehavioralCoverageScenarios(t, "version")
	runBehavioralCoverageScenarios(t, "config")
}

func TestCoverageManifestValidationReportsMissingAndStalePaths(t *testing.T) {
	capabilities := []commandCapability{{Path: "get"}, {Path: "get process-instance"}}
	manifest := map[string]coverageEntry{
		"get":   {Path: "get", Family: "get", ScenarioOwner: "integration/cli/get_test.go"},
		"stale": {Path: "stale", Family: "get", ScenarioOwner: "integration/cli/get_test.go"},
	}

	report := validateCoverageManifest(capabilities, manifest)

	assertContains(t, report.MissingPaths, "get process-instance")
	assertContains(t, report.StalePaths, "stale")
}

func TestCoverageManifestValidationReportsMissingLeafFlags(t *testing.T) {
	capabilities := []commandCapability{{
		Path: "get process-instance",
		Flags: []flagContract{
			{Name: "key"},
			{Name: "state"},
		},
		OutputModes: []outputModeContract{{Name: "one-line", Supported: true}},
	}}
	manifest := map[string]coverageEntry{
		"get process-instance": {
			Path:          "get process-instance",
			Family:        "get",
			ScenarioOwner: "integration/cli/get_test.go",
			Flags:         []string{"key"},
			OutputModes:   []string{"one-line"},
		},
	}

	report := validateCoverageManifest(capabilities, manifest)

	assertContainsSubstring(t, report.Mismatches, `get process-instance missing flags: ["state"]`)
}

func requireFamilyManifestSatisfaction(t *testing.T, family string, paths []string) []coverageEntry {
	t.Helper()
	if len(paths) == 0 {
		t.Fatalf("family %q has no expected command paths", family)
	}

	entries := make([]coverageEntry, 0, len(paths))
	for _, path := range paths {
		entry, ok := commandCoverageManifest[path]
		if !ok {
			t.Fatalf("family %q missing manifest entry for %q", family, path)
		}
		if entry.Family != family {
			t.Fatalf("manifest entry %q family = %q, want %q", path, entry.Family, family)
		}
		if strings.TrimSpace(entry.ScenarioOwner) == "" {
			t.Fatalf("manifest entry %q has no scenario owner", path)
		}
		entries = append(entries, entry)
	}
	return entries
}

func writeFamilyCoverageEvidence(t *testing.T, family string, entries []coverageEntry, records []evidenceRecord) {
	t.Helper()
	report := familyCoverageReport{
		Family:  family,
		Entries: entries,
		Records: records,
		Summary: familyCoverageSummary{
			CommandCount:     len(entries),
			DestructivePaths: destructiveManifestPaths(entries),
			OutputModes:      familyOutputModes(entries),
		},
	}
	writeJSON(t, "coverage-"+sanitizeEvidenceName(family)+".json", report)
}

func destructiveManifestPaths(entries []coverageEntry) []string {
	var paths []string
	for _, entry := range entries {
		if entry.Destructive {
			paths = append(paths, entry.Path)
		}
	}
	sort.Strings(paths)
	return paths
}

func familyOutputModes(entries []coverageEntry) []string {
	seen := make(map[string]struct{})
	for _, entry := range entries {
		for _, mode := range entry.OutputModes {
			seen[mode] = struct{}{}
		}
	}
	modes := make([]string, 0, len(seen))
	for mode := range seen {
		modes = append(modes, mode)
	}
	sort.Strings(modes)
	return modes
}

func validateCoverageManifest(capabilities []commandCapability, manifest map[string]coverageEntry) coverageReport {
	liveByPath := make(map[string]commandCapability, len(capabilities))
	for _, capability := range capabilities {
		liveByPath[capability.Path] = capability
	}

	report := coverageReport{
		ExpectedCount: expectedCommandInventoryCount,
		ActualCount:   len(capabilities),
	}

	for _, capability := range capabilities {
		entry, ok := manifest[capability.Path]
		if !ok {
			report.MissingPaths = append(report.MissingPaths, capability.Path)
			continue
		}
		report.Mismatches = append(report.Mismatches, validateCoverageEntry(capability, entry)...)
	}
	for path := range manifest {
		if _, ok := liveByPath[path]; !ok {
			report.StalePaths = append(report.StalePaths, path)
		}
	}

	sort.Strings(report.MissingPaths)
	sort.Strings(report.StalePaths)
	sort.Strings(report.Mismatches)
	return report
}

func validateCoverageEntry(capability commandCapability, entry coverageEntry) []string {
	var mismatches []string
	if entry.Path != capability.Path {
		mismatches = append(mismatches, fmt.Sprintf("%s entry path mismatch: %q", capability.Path, entry.Path))
	}
	if strings.TrimSpace(entry.Family) == "" {
		mismatches = append(mismatches, fmt.Sprintf("%s missing family", capability.Path))
	}
	if strings.TrimSpace(entry.ScenarioOwner) == "" {
		mismatches = append(mismatches, fmt.Sprintf("%s missing scenario owner", capability.Path))
	}
	if missing := missingStrings(flagNames(capability.Flags), entry.Flags); len(missing) > 0 {
		mismatches = append(mismatches, fmt.Sprintf("%s missing flags: %q", capability.Path, missing))
	}
	if missing := missingStrings(capability.Aliases, entry.Aliases); len(missing) > 0 {
		mismatches = append(mismatches, fmt.Sprintf("%s missing aliases: %q", capability.Path, missing))
	}
	if missing := missingStrings(supportedOutputModeNames(capability.OutputModes), entry.OutputModes); len(missing) > 0 {
		mismatches = append(mismatches, fmt.Sprintf("%s missing output modes: %q", capability.Path, missing))
	}
	return mismatches
}

func flattenCapabilities(commands []commandCapability) []commandCapability {
	var flattened []commandCapability
	var walk func([]commandCapability)
	walk = func(items []commandCapability) {
		for _, item := range items {
			flattened = append(flattened, item)
			walk(item.Children)
		}
	}
	walk(commands)
	sort.Slice(flattened, func(i, j int) bool {
		return flattened[i].Path < flattened[j].Path
	})
	return flattened
}

func flagNames(flags []flagContract) []string {
	names := make([]string, 0, len(flags))
	for _, flag := range flags {
		names = append(names, flag.Name)
	}
	sort.Strings(names)
	return names
}

func supportedOutputModeNames(modes []outputModeContract) []string {
	names := make([]string, 0, len(modes))
	for _, mode := range modes {
		if mode.Supported {
			names = append(names, mode.Name)
		}
	}
	sort.Strings(names)
	return names
}

func missingStrings(want []string, got []string) []string {
	want = sortedStrings(want)
	got = sortedStrings(got)
	gotSet := make(map[string]struct{}, len(got))
	for _, value := range got {
		gotSet[value] = struct{}{}
	}
	var missing []string
	for _, value := range want {
		if _, ok := gotSet[value]; !ok {
			missing = append(missing, value)
		}
	}
	return missing
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%q not found in %v", want, values)
}

func assertContainsSubstring(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(value, want) {
			return
		}
	}
	t.Fatalf("%q not found in %v", want, values)
}

func assertStringSlicesEqual(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(sortedStrings(got), sortedStrings(want)) {
		t.Fatalf("strings = %v, want %v", got, want)
	}
}
