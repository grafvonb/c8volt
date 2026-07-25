// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/toolx"
	"gopkg.in/yaml.v3"
)

const (
	envITBin      = "C8VOLT_IT_BIN"
	envITBuild    = "C8VOLT_IT_BUILD"
	envITWorkdir  = "C8VOLT_IT_WORKDIR"
	envITProfiles = "C8VOLT_IT_PROFILES"

	defaultCommandTimeout = 2 * time.Minute
)

var suite integrationSuite

type integrationSuite struct {
	repoRoot string
	workDir  string
	binPath  string
	marker   string
}

type commandResult struct {
	Args       []string  `json:"args"`
	StdoutPath string    `json:"stdoutPath"`
	StderrPath string    `json:"stderrPath"`
	ExitCode   int       `json:"exitCode"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Stdout     string    `json:"-"`
	Stderr     string    `json:"-"`
	Err        error     `json:"-"`
}

type integrationProfile struct {
	Name            string `json:"name"`
	ExpectedVersion string `json:"expectedVersion,omitempty"`
	ActualVersion   string `json:"actualVersion,omitempty"`
	Reachable       bool   `json:"reachable"`
	Tenant          string `json:"tenant,omitempty"`
}

type runMetadata struct {
	Marker    string    `json:"marker"`
	StartedAt time.Time `json:"startedAt"`
	WorkDir   string    `json:"workDir"`
	BinPath   string    `json:"binPath"`
	Profiles  []string  `json:"profiles,omitempty"`
}

type evidenceRecord struct {
	CommandPath    string    `json:"commandPath"`
	ScenarioName   string    `json:"scenarioName"`
	Profile        string    `json:"profile,omitempty"`
	CamundaVersion string    `json:"camundaVersion,omitempty"`
	Arguments      []string  `json:"arguments"`
	StdoutPath     string    `json:"stdoutPath"`
	StderrPath     string    `json:"stderrPath"`
	ExitCode       int       `json:"exitCode"`
	StartedAt      time.Time `json:"startedAt"`
	FinishedAt     time.Time `json:"finishedAt"`
	DataOwnership  []string  `json:"dataOwnership,omitempty"`
	ResourceKeys   []string  `json:"resourceKeys,omitempty"`
	Outcome        string    `json:"outcome"`
	FailureClass   string    `json:"failureClass,omitempty"`
}

type proposalRecord struct {
	Kind             string   `json:"kind"`
	RequiredState    string   `json:"requiredState"`
	CoverageNeed     string   `json:"coverageNeed"`
	FallbackUsed     string   `json:"fallbackUsed"`
	AffectedCommands []string `json:"affectedCommands"`
	AffectedVersions []string `json:"affectedVersions"`
	OperatorValue    string   `json:"operatorValue"`
}

type defaultLocalConfig struct {
	ActiveProfile string                         `yaml:"active_profile"`
	App           defaultLocalConfigApp          `yaml:"app"`
	Profiles      map[string]defaultLocalProfile `yaml:"profiles"`
	sourcePath    string
}

type defaultLocalProfile struct {
	App defaultLocalConfigApp `yaml:"app"`
}

type defaultLocalConfigApp struct {
	CamundaVersion string `yaml:"camunda_version"`
	Tenant         string `yaml:"tenant"`
}

func TestMain(m *testing.M) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find repository root: %v\n", err)
		os.Exit(1)
	}

	workDir, cleanup, err := prepareWorkDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "prepare integration workdir: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	suite = integrationSuite{
		repoRoot: repoRoot,
		workDir:  workDir,
		binPath:  filepath.Join(workDir, "c8volt"),
		marker:   newRunMarker(),
	}

	if existing := strings.TrimSpace(os.Getenv(envITBin)); existing != "" && os.Getenv(envITBuild) == "0" {
		suite.binPath = existing
	} else if err := buildC8VoltBinary(repoRoot, suite.binPath, workDir); err != nil {
		fmt.Fprintf(os.Stderr, "build c8volt binary: %v\n", err)
		os.Exit(1)
	}

	_ = writeJSONFile(filepath.Join(workDir, "run.json"), runMetadata{
		Marker:    suite.marker,
		StartedAt: time.Now().UTC(),
		WorkDir:   workDir,
		BinPath:   suite.binPath,
		Profiles:  selectedProfileNames(),
	})

	os.Exit(m.Run())
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %q", dir)
		}
		dir = parent
	}
}

func prepareWorkDir() (string, func(), error) {
	if configured := strings.TrimSpace(os.Getenv(envITWorkdir)); configured != "" {
		clean := filepath.Clean(configured)
		if err := os.MkdirAll(clean, 0o755); err != nil {
			return "", func() {}, err
		}
		if err := ensureEvidenceDirs(clean); err != nil {
			return "", func() {}, err
		}
		return clean, func() {}, nil
	}

	dir, err := os.MkdirTemp("", "c8volt-all-command-it-*")
	if err != nil {
		return "", func() {}, err
	}
	if err := ensureEvidenceDirs(dir); err != nil {
		return "", func() {}, err
	}
	return dir, func() {}, nil
}

func ensureEvidenceDirs(dir string) error {
	for _, child := range []string{"logs", "data"} {
		if err := os.MkdirAll(filepath.Join(dir, child), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func buildC8VoltBinary(repoRoot string, binPath string, workDir string) error {
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = repoRoot
	cmd.Env = appendDefaultEnv(os.Environ(), "GOCACHE", filepath.Join(workDir, "gocache"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func appendDefaultEnv(env []string, key string, value string) []string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return env
		}
	}
	return append(env, prefix+value)
}

func runC8Volt(t *testing.T, scenarioName string, args ...string) commandResult {
	t.Helper()

	if err := rejectExplicitConfigArgs(args); err != nil {
		t.Fatalf("%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	started := time.Now().UTC()
	cmd := exec.CommandContext(ctx, suite.binPath, args...)
	cmd.Dir = suite.workDir
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	finished := time.Now().UTC()

	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	stdoutPath := writeLogFile(t, scenarioName, "stdout", stdout.String())
	stderrPath := writeLogFile(t, scenarioName, "stderr", stderr.String())

	return commandResult{
		Args:       append([]string(nil), args...),
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		ExitCode:   exitCode,
		StartedAt:  started,
		FinishedAt: finished,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		Err:        err,
	}
}

func runC8VoltForProfile(t *testing.T, profile string, scenarioName string, args ...string) commandResult {
	t.Helper()
	return runC8Volt(t, scenarioName, argsForProfile(profile, args...)...)
}

func checkProfileVersion(t *testing.T, profile string, expectedVersion string) integrationProfile {
	t.Helper()
	result := runC8VoltForProfile(t, profile, "profile-"+profile+"-version", "get", "cluster", "version")
	evidence := integrationProfile{
		Name:            profile,
		ExpectedVersion: expectedVersion,
		ActualVersion:   strings.TrimSpace(result.Stdout),
		Reachable:       result.Err == nil,
	}
	if result.Err == nil && expectedVersion != "" && !strings.Contains(evidence.ActualVersion, expectedVersion) {
		t.Fatalf("profile %q version output %q does not contain expected version %q", profile, evidence.ActualVersion, expectedVersion)
	}
	return evidence
}

func writeJSON(t *testing.T, name string, value any) string {
	t.Helper()
	path := filepath.Join(suite.workDir, filepath.Clean(name))
	if !strings.HasPrefix(path, suite.workDir) {
		t.Fatalf("refusing to write evidence outside workdir: %s", name)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create evidence directory: %v", err)
	}
	if err := writeJSONFile(path, value); err != nil {
		t.Fatalf("write evidence %s: %v", name, err)
	}
	return path
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeLogFile(t *testing.T, scenarioName string, stream string, value string) string {
	t.Helper()
	name := sanitizeEvidenceName(scenarioName)
	path := filepath.Join(suite.workDir, "logs", name+"."+stream)
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("write %s log: %v", stream, err)
	}
	return path
}

func sanitizeEvidenceName(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func selectedProfileNames() []string {
	names, err := selectedProfileNamesFromDefaultConfig()
	if err != nil {
		return nil
	}
	return names
}

// selectedProfileNamesFromDefaultConfig chooses explicit suite profiles or the active default-local profile.
func selectedProfileNamesFromDefaultConfig() ([]string, error) {
	raw := strings.TrimSpace(os.Getenv(envITProfiles))
	if raw != "" {
		return splitProfileNames(raw), nil
	}

	cfg, err := readDefaultLocalConfig()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.ActiveProfile) == "" {
		return nil, nil
	}
	return []string{strings.TrimSpace(cfg.ActiveProfile)}, nil
}

// splitProfileNames parses the comma-separated profile selection environment value.
func splitProfileNames(raw string) []string {
	parts := strings.Split(raw, ",")
	profiles := make([]string, 0, len(parts))
	for _, part := range parts {
		profile := strings.TrimSpace(part)
		if profile != "" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

// selectedProfilesFromDefaultConfig resolves selected profile metadata from the operator's default local config.
func selectedProfilesFromDefaultConfig() ([]integrationProfile, error) {
	cfg, err := readDefaultLocalConfig()
	if err != nil {
		return nil, err
	}

	names := splitProfileNames(os.Getenv(envITProfiles))
	return profilesFromDefaultConfig(cfg, names)
}

// profilesFromDefaultConfig converts selected default-config entries into readiness evidence records.
func profilesFromDefaultConfig(cfg defaultLocalConfig, names []string) ([]integrationProfile, error) {
	if len(names) == 0 && strings.TrimSpace(cfg.ActiveProfile) != "" {
		names = []string{strings.TrimSpace(cfg.ActiveProfile)}
	}
	if len(names) == 0 {
		return nil, nil
	}

	baseVersion := normalizeExpectedCamundaVersion(cfg.App.CamundaVersion)
	profiles := make([]integrationProfile, 0, len(names))
	for _, name := range names {
		rawProfile, ok := cfg.Profiles[name]
		if !ok {
			return nil, fmt.Errorf("selected profile %q was not found in default local config %s", name, cfg.sourcePath)
		}

		expectedVersion := normalizeExpectedCamundaVersion(rawProfile.App.CamundaVersion)
		if expectedVersion == "" {
			expectedVersion = baseVersion
		}
		if expectedVersion == "" {
			expectedVersion = toolx.CurrentCamundaVersion.String()
		}

		profiles = append(profiles, integrationProfile{
			Name:            name,
			ExpectedVersion: expectedVersion,
			Tenant:          rawProfile.App.Tenant,
		})
	}
	return profiles, nil
}

// readDefaultLocalConfig loads profile metadata from local config homes, not generated cwd configs.
func readDefaultLocalConfig() (defaultLocalConfig, error) {
	paths := defaultLocalConfigPaths()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return defaultLocalConfig{}, fmt.Errorf("read default local config %s: %w", path, err)
		}

		var cfg defaultLocalConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return defaultLocalConfig{}, fmt.Errorf("parse default local config %s: %w", path, err)
		}
		cfg.sourcePath = path
		return cfg, nil
	}
	return defaultLocalConfig{}, fmt.Errorf("default local c8volt config not found in %s", strings.Join(paths, ", "))
}

// defaultLocalConfigPaths returns operator-local config paths used by the suite for profile metadata.
func defaultLocalConfigPaths() []string {
	var paths []string
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "c8volt", "config.yaml"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths,
			filepath.Join(home, ".config", "c8volt", "config.yaml"),
			filepath.Join(home, ".c8volt", "config.yaml"),
		)
	}
	return paths
}

// normalizeExpectedCamundaVersion records canonical minor versions when profile config uses shorthand.
func normalizeExpectedCamundaVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	version, err := toolx.NormalizeCamundaVersion(value)
	if err != nil {
		return value
	}
	return version.String()
}

func argsForProfile(profile string, args ...string) []string {
	if strings.TrimSpace(profile) == "" {
		return append([]string(nil), args...)
	}
	out := []string{"--profile", profile}
	return append(out, args...)
}

func rejectExplicitConfigArgs(args []string) error {
	for i, arg := range args {
		if arg == "--config" || arg == "-c" {
			return fmt.Errorf("integration suite must use default local c8volt config, found %s at argument %d", arg, i)
		}
		if strings.HasPrefix(arg, "--config=") {
			return fmt.Errorf("integration suite must use default local c8volt config, found %s", arg)
		}
	}
	return nil
}

func newRunMarker() string {
	return fmt.Sprintf("c8volt-it-%d-%d", time.Now().UTC().UnixNano(), os.Getpid())
}

func runMarkerVars(marker string) string {
	return fmt.Sprintf(`{"c8voltITRunId":%q}`, marker)
}

func commandEvidence(commandPath string, scenarioName string, result commandResult, outcome string) evidenceRecord {
	return evidenceRecord{
		CommandPath:  commandPath,
		ScenarioName: scenarioName,
		Arguments:    result.Args,
		StdoutPath:   result.StdoutPath,
		StderrPath:   result.StderrPath,
		ExitCode:     result.ExitCode,
		StartedAt:    result.StartedAt,
		FinishedAt:   result.FinishedAt,
		Outcome:      outcome,
	}
}

func writeEvidenceRecords(t *testing.T, name string, records []evidenceRecord) string {
	t.Helper()
	return writeJSON(t, name, records)
}

// writeDataEvidence stores reusable seeded-data identifiers under the suite data directory.
func writeDataEvidence(t *testing.T, name string, value any) string {
	t.Helper()
	return writeJSON(t, filepath.Join("data", name), value)
}

// decodeCommandPayload accepts either a shared command envelope or a direct JSON payload.
func decodeCommandPayload(output string, value any) error {
	var envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err == nil && len(envelope.Payload) > 0 {
		return json.Unmarshal(envelope.Payload, value)
	}
	return json.Unmarshal([]byte(output), value)
}

func writeCommandProposals(t *testing.T, proposals []proposalRecord) string {
	t.Helper()
	return writeJSON(t, "proposals-command.json", proposals)
}

func writeEmbeddedBPMNProposals(t *testing.T, proposals []proposalRecord) string {
	t.Helper()
	return writeJSON(t, "proposals-embedded-bpmn.json", proposals)
}

func requireTrimmedOutput(t *testing.T, output string) {
	t.Helper()
	if strings.TrimSpace(output) == "" {
		t.Fatal("expected non-empty output")
	}
}

func requireValidJSON(t *testing.T, output string) {
	t.Helper()
	var payload any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput:\n%s", err, output)
	}
}

func requireKeysOnlyOutput(t *testing.T, output string) {
	t.Helper()
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return
	}
	keyPattern := regexp.MustCompile(`^[0-9]+$`)
	for _, line := range strings.Split(trimmed, "\n") {
		if !keyPattern.MatchString(strings.TrimSpace(line)) {
			t.Fatalf("expected keys-only line, got %q in output:\n%s", line, output)
		}
	}
}

func requireSeededEvidence(t *testing.T, records []evidenceRecord) {
	t.Helper()
	for _, record := range records {
		for _, ownership := range record.DataOwnership {
			if ownership == "seeded" {
				return
			}
		}
	}
	t.Fatal("expected at least one seeded evidence record")
}

func requireNoExactGlobalCountAssertion(t *testing.T, description string) {
	t.Helper()
	if strings.Contains(strings.ToLower(description), "exact global count") {
		t.Fatalf("dirty-cluster-safe scenario must not rely on exact global counts: %s", description)
	}
}

func runFamilyCoverageScenarios(t *testing.T, family string, paths []string) {
	t.Helper()
	entries := requireFamilyManifestSatisfaction(t, family, paths)
	records := make([]evidenceRecord, 0, len(entries)*2)
	for _, entry := range entries {
		records = append(records, runCommandHelpScenario(t, family, entry, strings.Fields(entry.Path), "canonical"))
		for _, alias := range entry.Aliases {
			records = append(records, runCommandHelpScenario(t, family, entry, aliasCommandArgs(entry.Path, alias), "alias-"+alias))
		}
	}
	writeFamilyCoverageEvidence(t, family, entries, records)
}

func runCommandHelpScenario(t *testing.T, family string, entry coverageEntry, args []string, scenarioSuffix string) evidenceRecord {
	t.Helper()
	args = append(append([]string(nil), args...), "--help")
	scenario := "family-" + family + "-" + entry.Path + "-" + scenarioSuffix + "-help"
	result := runC8Volt(t, scenario, args...)
	record := commandEvidence(entry.Path, scenario, result, "pass")
	if result.Err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		t.Fatalf("%s help scenario failed: %v\nstderr:\n%s", entry.Path, result.Err, result.Stderr)
	}
	requireTrimmedOutput(t, result.Stdout)
	for _, flag := range entry.Flags {
		if !helpContainsFlag(result.Stdout, flag) {
			record.Outcome = "fail"
			record.FailureClass = "product"
			t.Fatalf("%s help output missing --%s\nstdout:\n%s", entry.Path, flag, result.Stdout)
		}
	}
	return record
}

func aliasCommandArgs(path string, alias string) []string {
	args := strings.Fields(path)
	if len(args) == 0 {
		return []string{alias}
	}
	args[len(args)-1] = alias
	return args
}

func helpContainsFlag(output string, flag string) bool {
	return strings.Contains(output, "--"+flag)
}
