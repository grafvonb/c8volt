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
	}

	if existing := strings.TrimSpace(os.Getenv(envITBin)); existing != "" && os.Getenv(envITBuild) == "0" {
		suite.binPath = existing
	} else if err := buildC8VoltBinary(repoRoot, suite.binPath, workDir); err != nil {
		fmt.Fprintf(os.Stderr, "build c8volt binary: %v\n", err)
		os.Exit(1)
	}

	_ = writeJSONFile(filepath.Join(workDir, "run.json"), runMetadata{
		Marker:    newRunMarker(),
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

	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	started := time.Now().UTC()
	cmd := exec.CommandContext(ctx, suite.binPath, args...)
	cmd.Dir = suite.repoRoot
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
	raw := strings.TrimSpace(os.Getenv(envITProfiles))
	if raw == "" {
		return nil
	}
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
