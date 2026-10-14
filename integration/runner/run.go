// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/integration/internal/validation"
)

const (
	envReleaseVersion  = "C8VOLT_IT_RELEASE_VERSION"
	envProfiles        = "C8VOLT_IT_PROFILES"
	envRequireTenants  = "C8VOLT_IT_REQUIRE_TENANTS"
	envWorkdir         = "C8VOLT_IT_WORKDIR"
	envBinary          = "C8VOLT_IT_BIN"
	envBuild           = "C8VOLT_IT_BUILD"
	envGoTestFlags     = "C8VOLT_IT_GO_TEST_FLAGS"
	envVerbose         = "C8VOLT_IT_VERBOSE"
	envTimeout         = "C8VOLT_IT_TIMEOUT"
	envVolumeTimeout   = "C8VOLT_IT_VOLUME_TIMEOUT"
	envRealTimeout     = "C8VOLT_IT_REAL_STATE_TIMEOUT"
	envObservedVersion = "C8VOLT_IT_OBSERVED_VERSION"

	manifestFileName = "manifest.json"
	releaseFileName  = "release.json"
	summaryFileName  = "summary.md"
	evidenceFileName = "scenario-evidence.json"
)

const (
	exitPassed      = 0
	exitFailed      = 1
	exitInvalid     = 2
	exitInterrupted = 130
)

var safeComponentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// manifestState describes the aggregate lifecycle persisted independently of
// child exit status so an abrupt stop can never look finalized.
type manifestState string

const (
	manifestRunning     manifestState = "running"
	manifestFinished    manifestState = "finished"
	manifestFailed      manifestState = "failed"
	manifestInterrupted manifestState = "interrupted"
)

// runnerManifest freezes the request and planned child matrix before any
// remote preflight or mutation-capable child starts.
type runnerManifest struct {
	SchemaVersion int                           `json:"schemaVersion"`
	State         manifestState                 `json:"state"`
	Request       validation.ValidationRequest  `json:"request"`
	Targets       []validation.ValidationTarget `json:"targets,omitempty"`
	Slices        []validation.SliceExecution   `json:"slices"`
}

// runnerConfig contains validated operator controls without retaining raw
// configuration files, credentials, or URLs.
type runnerConfig struct {
	RepoRoot       string
	EvidenceParent string
	ReleaseVersion string
	Profiles       []string
	TenantCoverage validation.TenantCoverage
	ExistingBinary string
	BuildBinary    bool
	GoTestFlags    []string
	Timeout        time.Duration
	VolumeTimeout  time.Duration
	RealTimeout    time.Duration
}

// commandKind lets fake executors distinguish safety gates from
// mutation-capable child invocations without parsing shell text.
type commandKind string

const (
	commandBuild     commandKind = "build"
	commandPreflight commandKind = "preflight"
	commandSlice     commandKind = "slice"
)

// commandSpec is a shell-free subprocess request with isolated streams.
type commandSpec struct {
	Kind         commandKind
	Name         string
	Args         []string
	Dir          string
	Env          []string
	Profile      string
	SliceID      string
	EvidencePath string
}

// commandResult preserves subprocess output and its exact exit status.
type commandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
}

// executor abstracts child execution so ordering, interruption, and evidence
// handling are testable without a cluster or tagged harness.
type executor interface {
	Run(context.Context, commandSpec) commandResult
}

// osExecutor runs commands directly, never through a shell.
type osExecutor struct{}

// Run captures stdout and stderr separately and preserves process exit codes.
func (osExecutor) Run(ctx context.Context, spec commandSpec) commandResult {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	return commandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: exitCode, Err: err}
}

// runFromEnvironment is the testable entry point used by main and validates
// controls before creating an invocation or launching any subprocess.
func runFromEnvironment(ctx context.Context, environ []string, execer executor) (int, error) {
	cfg, err := parseRunnerConfig(environ)
	if err != nil {
		return exitInvalid, err
	}
	catalog, gaps, err := validation.LoadDefaultPolicy()
	if err != nil {
		return exitInvalid, fmt.Errorf("load release policy: %w", err)
	}
	return run(ctx, cfg, catalog, gaps, execer, time.Now)
}

// run executes one sequential invocation and always finalizes recoverable
// evidence after the initial manifest has been persisted.
func run(
	ctx context.Context,
	cfg runnerConfig,
	catalog validation.CoverageCatalog,
	gaps validation.GapCatalog,
	execer executor,
	now func() time.Time,
) (int, error) {
	startedAt := now().UTC()
	runID, err := newRunID(startedAt)
	if err != nil {
		return exitInvalid, fmt.Errorf("create run identity: %w", err)
	}
	invocationRoot, err := createInvocationRoot(cfg.EvidenceParent, runID)
	if err != nil {
		return exitInvalid, err
	}

	mode := validation.ModeExploratory
	if cfg.ReleaseVersion != "" {
		mode = validation.ModeRelease
	}
	request := validation.ValidationRequest{
		SchemaVersion:          validation.SchemaVersion,
		RunID:                  runID,
		Mode:                   mode,
		RequestedMinor:         cfg.ReleaseVersion,
		Profiles:               append([]string(nil), cfg.Profiles...),
		RequiredTenantCoverage: cfg.TenantCoverage,
		CatalogRevision:        catalog.Revision,
		StartedAt:              startedAt,
		EvidenceRoot:           invocationRoot,
	}
	if err := validation.ValidateRequest(request, catalog); err != nil {
		return exitInvalid, fmt.Errorf("validate release request: %w", err)
	}

	manifest := runnerManifest{
		SchemaVersion: validation.SchemaVersion,
		State:         manifestRunning,
		Request:       request,
		Slices:        plannedSlices(request, catalog, startedAt),
	}
	manifestPath := filepath.Join(invocationRoot, manifestFileName)
	if err := writeJSONAtomic(manifestPath, manifest); err != nil {
		return exitInvalid, fmt.Errorf("write initial manifest: %w", err)
	}

	binaryPath, buildFailure := prepareBinary(ctx, cfg, invocationRoot, execer)
	if buildFailure != nil {
		manifest.State = manifestFailed
		summary := failureSummary(request, validation.FailureHarness, buildFailure.Error())
		return finalizeInvocation(manifestPath, invocationRoot, manifest, summary, exitFailed, buildFailure)
	}

	targets, preflightFailure := preflightAll(ctx, cfg, request, binaryPath, execer)
	manifest.Targets = targets
	if preflightFailure != nil || ctx.Err() != nil {
		interrupted := ctx.Err() != nil
		input := validation.ReductionInput{
			Request: request, Catalog: catalog, GapCatalog: gaps,
			Targets: targets, Slices: manifest.Slices, Interrupted: interrupted,
		}
		summary, reduceErr := validation.Reduce(input)
		if reduceErr != nil {
			summary = failureSummary(request, validation.FailureHarness, reduceErr.Error())
		}
		if interrupted {
			manifest.State = manifestInterrupted
			return finalizeInvocation(manifestPath, invocationRoot, manifest, summary, exitInterrupted, ctx.Err())
		}
		manifest.State = manifestFailed
		return finalizeInvocation(manifestPath, invocationRoot, manifest, summary, exitFailed, preflightFailure)
	}

	if err := writeJSONAtomic(manifestPath, manifest); err != nil {
		return exitFailed, fmt.Errorf("persist preflight manifest: %w", err)
	}

	var evidence []validation.ScenarioEvidence
	unsafeProfiles := map[string]bool{}
	for index := range manifest.Slices {
		slice := &manifest.Slices[index]
		if unsafeProfiles[slice.Profile] {
			continue
		}

		target, recheckErr := preflightTarget(ctx, cfg, request, binaryPath, slice.Profile, execer)
		if recheckErr != nil {
			unsafeProfiles[slice.Profile] = true
			replaceTarget(&manifest.Targets, target)
			if ctx.Err() != nil {
				break
			}
			continue
		}
		replaceTarget(&manifest.Targets, target)

		slice.State = validation.SliceRunning
		slice.StartedAt = now().UTC()
		if err := writeJSONAtomic(manifestPath, manifest); err != nil {
			return exitFailed, fmt.Errorf("persist running slice manifest: %w", err)
		}
		childEvidence, interrupted := runSlice(ctx, cfg, request, binaryPath, target.ObservedVersion, slice, execer, now)
		evidence = append(evidence, childEvidence...)
		if err := writeJSONAtomic(manifestPath, manifest); err != nil {
			return exitFailed, fmt.Errorf("persist slice manifest: %w", err)
		}
		if interrupted {
			break
		}
	}

	interrupted := ctx.Err() != nil
	input := validation.ReductionInput{
		Request: request, Catalog: catalog, GapCatalog: gaps,
		Targets: manifest.Targets, Slices: manifest.Slices,
		Evidence: evidence, Interrupted: interrupted,
	}
	summary, reduceErr := validation.Reduce(input)
	if reduceErr != nil {
		summary = failureSummary(request, validation.FailureHarness, reduceErr.Error())
	}
	if interrupted {
		manifest.State = manifestInterrupted
		return finalizeInvocation(manifestPath, invocationRoot, manifest, summary, exitInterrupted, ctx.Err())
	}
	if summary.Verdict == validation.VerdictPassed || summary.Verdict == validation.VerdictPassedWithGaps {
		manifest.State = manifestFinished
		return finalizeInvocation(manifestPath, invocationRoot, manifest, summary, exitPassed, reduceErr)
	}
	manifest.State = manifestFailed
	return finalizeInvocation(manifestPath, invocationRoot, manifest, summary, exitFailed, reduceErr)
}

// parseRunnerConfig accepts only the documented environment surface and
// rejects test selection overrides before any build or remote preflight.
func parseRunnerConfig(environ []string) (runnerConfig, error) {
	env := environmentMap(environ)
	repoRoot, err := findRepoRoot()
	if err != nil {
		return runnerConfig{}, err
	}

	parent := strings.TrimSpace(env[envWorkdir])
	if parent == "" {
		parent = filepath.Join(os.TempDir(), "c8volt-release-validation")
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		return runnerConfig{}, fmt.Errorf("resolve evidence parent: %w", err)
	}
	parent = filepath.Clean(parent)
	if err := validation.ValidateEvidenceRoot(parent); err != nil {
		return runnerConfig{}, fmt.Errorf("validate evidence parent: %w", err)
	}

	profiles, err := parseProfiles(env[envProfiles])
	if err != nil {
		return runnerConfig{}, err
	}
	if len(profiles) == 0 {
		return runnerConfig{}, errors.New("aggregate validation requires C8VOLT_IT_PROFILES")
	}
	releaseVersion := strings.TrimSpace(env[envReleaseVersion])
	if releaseVersion != "" {
		if _, err := validation.CanonicalMinor(releaseVersion); err != nil {
			return runnerConfig{}, err
		}
		if releaseVersion == "8.8" {
			return runnerConfig{}, errors.New("release validation requires Camunda 8.9 or 8.10")
		}
	}

	tenantCoverage := validation.TenantCoverage(strings.TrimSpace(env[envRequireTenants]))
	if tenantCoverage == "" {
		tenantCoverage = validation.TenantCoverageNone
	}
	switch tenantCoverage {
	case validation.TenantCoverageNone, validation.TenantCoverageLogical, validation.TenantCoveragePhysical:
	default:
		return runnerConfig{}, fmt.Errorf("unsupported %s value %q", envRequireTenants, tenantCoverage)
	}

	buildBinary := true
	if raw, exists := env[envBuild]; exists && strings.TrimSpace(raw) != "" {
		switch strings.TrimSpace(raw) {
		case "1":
			buildBinary = true
		case "0":
			buildBinary = false
		default:
			return runnerConfig{}, fmt.Errorf("%s must be 0 or 1", envBuild)
		}
	}
	existingBinary := strings.TrimSpace(env[envBinary])
	if !buildBinary {
		if existingBinary == "" || !filepath.IsAbs(existingBinary) || filepath.Clean(existingBinary) != existingBinary {
			return runnerConfig{}, fmt.Errorf("%s must be a clean absolute path when %s=0", envBinary, envBuild)
		}
		info, statErr := os.Stat(existingBinary)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			return runnerConfig{}, fmt.Errorf("%s does not name an executable regular file", envBinary)
		}
	}

	goTestFlags, err := parseGoTestFlags(env[envGoTestFlags])
	if err != nil {
		return runnerConfig{}, err
	}
	timeout, err := parseDurationControl(env, envTimeout, 60*time.Minute)
	if err != nil {
		return runnerConfig{}, err
	}
	volumeTimeout, err := parseDurationControl(env, envVolumeTimeout, 90*time.Minute)
	if err != nil {
		return runnerConfig{}, err
	}
	realTimeout, err := parseDurationControl(env, envRealTimeout, 90*time.Minute)
	if err != nil {
		return runnerConfig{}, err
	}

	return runnerConfig{
		RepoRoot: repoRoot, EvidenceParent: parent, ReleaseVersion: releaseVersion,
		Profiles: profiles, TenantCoverage: tenantCoverage,
		ExistingBinary: existingBinary, BuildBinary: buildBinary, GoTestFlags: goTestFlags,
		Timeout: timeout, VolumeTimeout: volumeTimeout, RealTimeout: realTimeout,
	}, nil
}

// parseProfiles keeps profile identities safe for directory names and rejects
// duplicates instead of silently changing the requested target matrix.
func parseProfiles(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	profiles := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		profile := strings.TrimSpace(part)
		if !safeComponentPattern.MatchString(profile) {
			return nil, fmt.Errorf("invalid profile name %q", profile)
		}
		if _, exists := seen[profile]; exists {
			return nil, fmt.Errorf("profile %q is selected more than once", profile)
		}
		seen[profile] = struct{}{}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

// parseGoTestFlags permits verbosity and fail-fast reporting without allowing
// operators to remove required tests or replace runner-owned controls.
func parseGoTestFlags(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	flags := strings.Fields(raw)
	for _, flag := range flags {
		if flag != "-v" && flag != "-failfast" {
			return nil, fmt.Errorf("unsafe or unsupported %s value %q", envGoTestFlags, flag)
		}
	}
	return flags, nil
}

// parseDurationControl validates positive runner-owned timeouts.
func parseDurationControl(env map[string]string, name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(env[name])
	if raw == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return duration, nil
}

// plannedSlices freezes one child for every applicable catalog slice and
// selected profile before any preflight runs.
func plannedSlices(request validation.ValidationRequest, catalog validation.CoverageCatalog, startedAt time.Time) []validation.SliceExecution {
	sliceTests := map[string][]string{}
	for _, requirement := range catalog.Requirements {
		if request.RequestedMinor != "" && !validation.Applicable(request.RequestedMinor, requirement.ApplicableMinors) {
			continue
		}
		if strings.HasPrefix(requirement.ScenarioID, "Test") {
			sliceTests[requirement.Slice] = append(sliceTests[requirement.Slice], requirement.ScenarioID)
		}
	}
	sliceIDs := make([]string, 0, len(sliceTests))
	for sliceID := range sliceTests {
		sliceIDs = append(sliceIDs, sliceID)
		sort.Strings(sliceTests[sliceID])
	}
	sort.Slice(sliceIDs, func(i, j int) bool {
		leftPriority := slicePriority(sliceIDs[i])
		rightPriority := slicePriority(sliceIDs[j])
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return sliceIDs[i] < sliceIDs[j]
	})

	var slices []validation.SliceExecution
	for _, sliceID := range sliceIDs {
		for _, profile := range request.Profiles {
			directory := filepath.Join(request.EvidenceRoot, "slices", sliceID, profile)
			slices = append(slices, validation.SliceExecution{
				SchemaVersion: validation.SchemaVersion, RunID: request.RunID,
				SliceID: sliceID, Profile: profile, StartedAt: startedAt,
				EvidenceDirectory: directory, State: validation.SlicePlanned,
				SelectedTests: append([]string(nil), sliceTests[sliceID]...),
			})
		}
	}
	return slices
}

// slicePriority keeps every nonmutating common gate ahead of examples and
// mutation-capable family slices without coupling execution to Make recipes.
func slicePriority(sliceID string) int {
	switch sliceID {
	case "common-preflight":
		return 0
	case "common-examples":
		return 1
	default:
		return 2
	}
}

// prepareBinary either honors a validated executable override or builds one
// binary for every subsequent child invocation.
func prepareBinary(ctx context.Context, cfg runnerConfig, invocationRoot string, execer executor) (string, error) {
	if !cfg.BuildBinary {
		return cfg.ExistingBinary, nil
	}
	binaryPath := filepath.Join(invocationRoot, "bin", "c8volt")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return "", fmt.Errorf("create binary directory: %w", err)
	}
	result := execer.Run(ctx, commandSpec{
		Kind: commandBuild, Name: "go", Args: []string{"build", "-trimpath", "-o", binaryPath, "."},
		Dir: cfg.RepoRoot, Env: os.Environ(),
	})
	if result.Err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("build c8volt (exit %d): %s", result.ExitCode, redactedText(result.Stderr))
	}
	return binaryPath, nil
}

// preflightAll verifies every selected target before any mutation-capable
// child process is allowed to start.
func preflightAll(
	ctx context.Context,
	cfg runnerConfig,
	request validation.ValidationRequest,
	binaryPath string,
	execer executor,
) ([]validation.ValidationTarget, error) {
	targets := make([]validation.ValidationTarget, 0, len(request.Profiles))
	var failures []error
	for _, profile := range request.Profiles {
		target, err := preflightTarget(ctx, cfg, request, binaryPath, profile, execer)
		targets = append(targets, target)
		if err != nil {
			failures = append(failures, err)
		}
		if ctx.Err() != nil {
			break
		}
	}
	return targets, errors.Join(failures...)
}

// preflightTarget uses stable JSON output to verify topology readiness and the
// exact observed minor without serializing route or credential configuration.
func preflightTarget(
	ctx context.Context,
	cfg runnerConfig,
	request validation.ValidationRequest,
	binaryPath string,
	profile string,
	execer executor,
) (validation.ValidationTarget, error) {
	expected := request.RequestedMinor
	target := validation.ValidationTarget{
		SchemaVersion: validation.SchemaVersion, Profile: profile, ExpectedMinor: expected,
	}
	result := execer.Run(ctx, commandSpec{
		Kind: commandPreflight, Name: binaryPath,
		Args: []string{"--profile", profile, "--json", "config", "test-connection"},
		Dir:  cfg.RepoRoot, Env: os.Environ(), Profile: profile,
	})
	if ctx.Err() != nil {
		target.Reason = "preflight interrupted"
		return target, ctx.Err()
	}
	if result.Err != nil || result.ExitCode != 0 {
		target.Reason = redactedText(result.Stderr)
		return target, fmt.Errorf("profile %q preflight failed (exit %d): %s", profile, result.ExitCode, target.Reason)
	}

	var payload struct {
		OK      bool `json:"ok"`
		Cluster struct {
			GatewayVersion string `json:"gateway_version"`
		} `json:"cluster"`
	}
	decoder := json.NewDecoder(bytes.NewReader(result.Stdout))
	if err := decoder.Decode(&payload); err != nil || !payload.OK {
		target.Reason = "invalid readiness response"
		return target, fmt.Errorf("profile %q returned invalid readiness JSON", profile)
	}
	if err := requireJSONEOF(decoder); err != nil {
		target.Reason = "invalid readiness response"
		return target, fmt.Errorf("profile %q returned trailing readiness JSON", profile)
	}
	observedMinor, err := validation.ObservedMinor(payload.Cluster.GatewayVersion)
	if err != nil {
		target.Reason = "gateway version is not verifiable"
		return target, fmt.Errorf("profile %q: %w", profile, err)
	}
	if expected == "" {
		expected = observedMinor
		target.ExpectedMinor = observedMinor
	}
	target.ObservedVersion = payload.Cluster.GatewayVersion
	if observedMinor != expected {
		target.Reason = fmt.Sprintf("expected minor %s but observed %s", expected, observedMinor)
		return target, fmt.Errorf("profile %q %s", profile, target.Reason)
	}
	target.Ready = true
	return target, nil
}

// runSlice persists running state before launch, captures all child artifacts,
// and returns only strictly decoded current-run evidence.
func runSlice(
	ctx context.Context,
	cfg runnerConfig,
	request validation.ValidationRequest,
	binaryPath string,
	observedVersion string,
	slice *validation.SliceExecution,
	execer executor,
	now func() time.Time,
) ([]validation.ScenarioEvidence, bool) {
	if err := os.MkdirAll(slice.EvidenceDirectory, 0o755); err != nil {
		markSliceFailed(slice, now().UTC(), -1, fmt.Sprintf("create evidence directory: %v", err))
		return nil, false
	}
	evidencePath := filepath.Join(slice.EvidenceDirectory, evidenceFileName)

	args := []string{"test", "-json", "-tags=integration", "./integration/cli"}
	args = append(args, cfg.GoTestFlags...)
	args = append(args,
		"-run", exactTestPattern(slice.SelectedTests),
		"-count=1", "-timeout="+sliceTimeout(cfg, slice.SliceID).String(),
	)
	env := append([]string(nil), os.Environ()...)
	verbose := "0"
	if containsString(cfg.GoTestFlags, "-v") {
		verbose = "1"
	}
	env = appendEnvironment(env,
		envBinary, binaryPath,
		envBuild, "0",
		envProfiles, slice.Profile,
		envWorkdir, slice.EvidenceDirectory,
		envReleaseVersion, request.RequestedMinor,
		envRequireTenants, string(request.RequiredTenantCoverage),
		envVerbose, verbose,
		envObservedVersion, observedVersion,
		"C8VOLT_IT_RUN_ID", request.RunID,
		"C8VOLT_IT_SLICE_ID", slice.SliceID,
		"C8VOLT_IT_EVIDENCE_FILE", evidencePath,
	)
	result := execer.Run(ctx, commandSpec{
		Kind: commandSlice, Name: "go", Args: args, Dir: cfg.RepoRoot, Env: env,
		Profile: slice.Profile, SliceID: slice.SliceID, EvidencePath: evidencePath,
	})
	finishedAt := now().UTC()
	if err := writeChildArtifacts(slice.EvidenceDirectory, result); err != nil {
		markSliceFailed(slice, finishedAt, result.ExitCode, err.Error())
		return nil, false
	}

	if ctx.Err() != nil {
		slice.State = validation.SliceInterrupted
		slice.FinishedAt = &finishedAt
		slice.TerminationReason = "interrupted"
		slice.ExitCode = nil
		return nil, true
	}
	if eventErr := validateTestEvents(result.Stdout); eventErr != nil {
		markSliceFailed(slice, finishedAt, result.ExitCode, eventErr.Error())
		return nil, false
	}
	if result.Err != nil || result.ExitCode != 0 {
		markSliceFailed(slice, finishedAt, result.ExitCode, redactedText(result.Stderr))
		evidence, evidenceErr := readScenarioEvidence(evidencePath, request, *slice)
		if evidenceErr != nil {
			return nil, false
		}
		return evidence, false
	}
	evidence, err := readScenarioEvidence(evidencePath, request, *slice)
	if err != nil {
		markSliceFailed(slice, finishedAt, result.ExitCode, err.Error())
		return nil, false
	}
	exitCode := result.ExitCode
	slice.State = validation.SliceFinished
	slice.FinishedAt = &finishedAt
	slice.ExitCode = &exitCode
	return evidence, false
}

// containsString reports exact membership for the small validated flag set.
func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// writeChildArtifacts preserves separate process streams and the immutable Go
// test event stream before any child outcome is interpreted.
func writeChildArtifacts(directory string, result commandResult) error {
	artifacts := []struct {
		name string
		data []byte
	}{
		{name: "stdout.log", data: result.Stdout},
		{name: "stderr.log", data: result.Stderr},
		{name: "test-events.jsonl", data: result.Stdout},
	}
	for _, artifact := range artifacts {
		if err := os.WriteFile(filepath.Join(directory, artifact.name), artifact.data, 0o644); err != nil {
			return fmt.Errorf("write child artifact %s: %w", artifact.name, err)
		}
	}
	return nil
}

// markSliceFailed retains the first failing child result for aggregate
// reduction; later successful children are stored in separate records.
func markSliceFailed(slice *validation.SliceExecution, finishedAt time.Time, exitCode int, reason string) {
	slice.State = validation.SliceFailed
	slice.FinishedAt = &finishedAt
	slice.ExitCode = &exitCode
	slice.TerminationReason = redactedText([]byte(reason))
}

// readScenarioEvidence strictly decodes the child-owned normalized evidence
// file and binds every record to the planned slice and profile.
func readScenarioEvidence(path string, request validation.ValidationRequest, slice validation.SliceExecution) ([]validation.ScenarioEvidence, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read normalized evidence: %w", err)
	}
	var records []validation.ScenarioEvidence
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("decode normalized evidence: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode normalized evidence: %w", err)
	}
	for _, record := range records {
		if record.RunID != request.RunID || record.SliceID != slice.SliceID || record.Profile != slice.Profile {
			return nil, fmt.Errorf("normalized evidence provenance does not match planned child")
		}
	}
	return records, nil
}

// validateTestEvents rejects successful exits with malformed or empty Go test
// event streams, which otherwise could conceal a wrapper or harness failure.
func validateTestEvents(data []byte) error {
	lines := bytes.Split(data, []byte("\n"))
	count := 0
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		count++
		var event struct {
			Action string `json:"Action"`
		}
		if err := json.Unmarshal(line, &event); err != nil || event.Action == "" {
			return errors.New("malformed go test event stream")
		}
	}
	if count == 0 {
		return errors.New("empty go test event stream")
	}
	return nil
}

// exactTestPattern anchors every catalog-owned test identity so similarly
// prefixed entry points cannot be selected accidentally.
func exactTestPattern(tests []string) string {
	quoted := make([]string, 0, len(tests))
	for _, test := range tests {
		quoted = append(quoted, regexp.QuoteMeta(test))
	}
	return "^(?:" + strings.Join(quoted, "|") + ")$"
}

// sliceTimeout selects the existing timeout control for each family class.
func sliceTimeout(cfg runnerConfig, sliceID string) time.Duration {
	switch {
	case strings.HasPrefix(sliceID, "volume-"):
		return cfg.VolumeTimeout
	case strings.HasPrefix(sliceID, "real-state-"):
		return cfg.RealTimeout
	default:
		return cfg.Timeout
	}
}

// replaceTarget keeps one authoritative latest readiness record per profile.
func replaceTarget(targets *[]validation.ValidationTarget, replacement validation.ValidationTarget) {
	for index := range *targets {
		if (*targets)[index].Profile == replacement.Profile {
			(*targets)[index] = replacement
			return
		}
	}
	*targets = append(*targets, replacement)
}

// finalizeInvocation atomically writes the final manifest, machine summary,
// and redacted human summary without appending post-commit bookkeeping.
func finalizeInvocation(
	manifestPath string,
	invocationRoot string,
	manifest runnerManifest,
	summary validation.ReleaseSummary,
	exitCode int,
	runErr error,
) (int, error) {
	var writeErrors []error
	if err := writeJSONAtomic(manifestPath, manifest); err != nil {
		writeErrors = append(writeErrors, fmt.Errorf("write final manifest: %w", err))
	}
	if err := writeJSONAtomic(filepath.Join(invocationRoot, releaseFileName), summary); err != nil {
		writeErrors = append(writeErrors, fmt.Errorf("write release summary: %w", err))
	}
	if err := writeFileAtomic(filepath.Join(invocationRoot, summaryFileName), []byte(renderSummary(summary)), 0o644); err != nil {
		writeErrors = append(writeErrors, fmt.Errorf("write human summary: %w", err))
	}
	return exitCode, errors.Join(runErr, errors.Join(writeErrors...))
}

// failureSummary constructs a redacted machine result for failures that occur
// before the reducer has enough records to evaluate the frozen matrix.
func failureSummary(request validation.ValidationRequest, class validation.FailureClass, reason string) validation.ReleaseSummary {
	return validation.ReleaseSummary{
		SchemaVersion: validation.SchemaVersion, RunID: request.RunID,
		Mode: request.Mode, RequestedMinor: request.RequestedMinor,
		FamilyCounters: map[string]validation.OutcomeCounters{},
		Failures:       []validation.Finding{{FailureClass: class, Reason: redactedText([]byte(reason))}},
		Verdict:        validation.VerdictFailed,
	}
}

// renderSummary produces a compact audit index; detailed counters and findings
// remain canonical in release.json.
func renderSummary(summary validation.ReleaseSummary) string {
	var output strings.Builder
	output.WriteString("# c8volt Release Validation Summary\n\n")
	fmt.Fprintf(&output, "- Run: `%s`\n", summary.RunID)
	fmt.Fprintf(&output, "- Mode: `%s`\n", summary.Mode)
	if summary.RequestedMinor != "" {
		fmt.Fprintf(&output, "- Requested minor: `%s`\n", summary.RequestedMinor)
	}
	fmt.Fprintf(&output, "- Verdict: `%s`\n", summary.Verdict)
	fmt.Fprintf(&output, "- Coverage complete: `%t`\n", summary.CoverageComplete)
	fmt.Fprintf(&output, "- Missing requirements: `%d`\n", len(summary.MissingRequirements))
	fmt.Fprintf(&output, "- Gaps: `%d`\n", len(summary.Gaps))
	fmt.Fprintf(&output, "- Failures: `%d`\n", len(summary.Failures))
	if len(summary.Failures) > 0 {
		output.WriteString("\n## Failures\n\n")
		for _, finding := range summary.Failures {
			fmt.Fprintf(&output, "- %s\n", redactedText([]byte(finding.Reason)))
		}
	}
	return output.String()
}

// writeJSONAtomic prevents readers from treating a truncated manifest or
// aggregate result as final evidence.
func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}

// writeFileAtomic replaces one file only after its complete temporary content
// has been written and closed in the same directory.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	removeTemporary = false
	return nil
}

// newRunID combines time and cryptographic entropy to keep concurrent
// invocations isolated without exposing operator configuration.
func newRunID(now time.Time) (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "release-" + now.Format("20060102T150405Z") + "-" + hex.EncodeToString(random), nil
}

// createInvocationRoot creates a unique child beneath the validated parent.
func createInvocationRoot(parent string, runID string) (string, error) {
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", fmt.Errorf("create evidence parent: %w", err)
	}
	root := filepath.Join(parent, runID)
	if err := validation.ValidateChildPath(parent, root); err != nil {
		return "", err
	}
	if err := os.Mkdir(root, 0o755); err != nil {
		return "", fmt.Errorf("create invocation directory: %w", err)
	}
	return root, nil
}

// findRepoRoot locates go.mod without assuming the runner's working directory.
func findRepoRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("go.mod not found")
		}
		directory = parent
	}
}

// environmentMap implements last-value-wins environment semantics.
func environmentMap(environ []string) map[string]string {
	result := map[string]string{}
	for _, entry := range environ {
		key, value, found := strings.Cut(entry, "=")
		if found {
			result[key] = value
		}
	}
	return result
}

// appendEnvironment replaces runner-owned child controls deterministically.
func appendEnvironment(environ []string, pairs ...string) []string {
	values := environmentMap(environ)
	for index := 0; index < len(pairs); index += 2 {
		values[pairs[index]] = pairs[index+1]
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

// requireJSONEOF rejects concatenated values that could make readiness or
// evidence interpretation ambiguous.
func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

// redactedText strips secret assignments and bearer tokens through the shared
// policy redactor before text enters durable summaries.
func redactedText(data []byte) string {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "no diagnostic output"
	}
	return validation.RedactSecrets(text)
}
