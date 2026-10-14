// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/integration/internal/validation"
)

// fakeExecutor records process ordering and lets each test control durable
// child evidence without starting the tagged integration harness.
type fakeExecutor struct {
	mu      sync.Mutex
	specs   []commandSpec
	handler func(context.Context, commandSpec) commandResult
}

// Run records an immutable command copy before invoking the test behavior.
func (f *fakeExecutor) Run(ctx context.Context, spec commandSpec) commandResult {
	f.mu.Lock()
	spec.Args = append([]string(nil), spec.Args...)
	spec.Env = append([]string(nil), spec.Env...)
	f.specs = append(f.specs, spec)
	f.mu.Unlock()
	return f.handler(ctx, spec)
}

// commands returns a race-safe copy of all observed command specifications.
func (f *fakeExecutor) commands() []commandSpec {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]commandSpec(nil), f.specs...)
}

// TestInvalidControlsExitTwoWithoutSubprocesses verifies unsafe selectors fail
// before build, preflight, filesystem mutation, or child execution.
func TestInvalidControlsExitTwoWithoutSubprocesses(t *testing.T) {
	execer := &fakeExecutor{handler: func(context.Context, commandSpec) commandResult {
		t.Fatal("invalid controls must not execute a subprocess")
		return commandResult{}
	}}
	exitCode, err := runFromEnvironment(context.Background(), []string{
		envReleaseVersion + "=8.10",
		envProfiles + "=dev810",
		envGoTestFlags + "=-run TestGetFamily",
	}, execer)
	if exitCode != exitInvalid || err == nil {
		t.Fatalf("runFromEnvironment exit/error = %d/%v, want 2/non-nil", exitCode, err)
	}
	if len(execer.commands()) != 0 {
		t.Fatalf("commands = %d, want 0", len(execer.commands()))
	}
}

// TestPreflightMismatchBlocksEverySlice verifies all target gates complete and
// a wrong minor produces zero mutation-capable child processes.
func TestPreflightMismatchBlocksEverySlice(t *testing.T) {
	cfg := testRunnerConfig(t, []string{"dev810", "wrong89"})
	catalog, gaps := testPolicy("baseline-get")
	execer := &fakeExecutor{handler: func(_ context.Context, spec commandSpec) commandResult {
		switch spec.Kind {
		case commandBuild:
			return commandResult{}
		case commandPreflight:
			version := "8.10.3"
			if spec.Profile == "wrong89" {
				version = "8.9.9"
			}
			return readinessResult(version)
		case commandSlice:
			t.Fatalf("slice %s/%s ran after failed aggregate preflight", spec.SliceID, spec.Profile)
		}
		return commandResult{}
	}}

	exitCode, err := run(context.Background(), cfg, catalog, gaps, execer, fixedClock())
	if exitCode != exitFailed || err == nil {
		t.Fatalf("run exit/error = %d/%v, want 1/non-nil", exitCode, err)
	}
	if countKind(execer.commands(), commandSlice) != 0 {
		t.Fatal("preflight mismatch submitted a mutation-capable child")
	}
	if countKind(execer.commands(), commandPreflight) != 2 {
		t.Fatalf("preflight calls = %d, want both selected targets", countKind(execer.commands(), commandPreflight))
	}
	summary := readOnlyReleaseSummary(t, cfg.EvidenceParent)
	if summary.Verdict != validation.VerdictFailed {
		t.Fatalf("verdict = %q, want failed", summary.Verdict)
	}
}

// TestExistingBinarySkipsBuild verifies the contracted build override is
// honored without invoking the compiler a second time.
func TestExistingBinarySkipsBuild(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "c8volt")
	if err := os.WriteFile(binaryPath, []byte("fake"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	execer := &fakeExecutor{handler: func(context.Context, commandSpec) commandResult {
		t.Fatal("existing binary preparation must not execute a subprocess")
		return commandResult{}
	}}
	path, err := prepareBinary(context.Background(), runnerConfig{
		BuildBinary: false, ExistingBinary: binaryPath,
	}, t.TempDir(), execer)
	if err != nil || path != binaryPath {
		t.Fatalf("prepareBinary path/error = %q/%v, want existing path/nil", path, err)
	}
	if len(execer.commands()) != 0 {
		t.Fatalf("commands = %d, want 0", len(execer.commands()))
	}
}

// TestSuccessfulRunBuildsOnceAndIsolatesEvidence verifies one build is shared
// while every invocation, profile, and slice receives a unique directory.
func TestSuccessfulRunBuildsOnceAndIsolatesEvidence(t *testing.T) {
	cfg := testRunnerConfig(t, []string{"dev810a", "dev810b"})
	catalog, gaps := testPolicy("baseline-get")
	execer := passingExecutor(t, "8.10.3")

	exitCode, err := run(context.Background(), cfg, catalog, gaps, execer, fixedClock())
	if exitCode != exitPassed || err != nil {
		t.Fatalf("run exit/error = %d/%v, want 0/nil", exitCode, err)
	}
	commands := execer.commands()
	if countKind(commands, commandBuild) != 1 || countKind(commands, commandSlice) != 2 {
		t.Fatalf("build/slice calls = %d/%d, want 1/2", countKind(commands, commandBuild), countKind(commands, commandSlice))
	}
	paths := map[string]struct{}{}
	for _, spec := range commands {
		if spec.Kind != commandSlice {
			continue
		}
		childEnv := environmentMap(spec.Env)
		if childEnv[envReleaseVersion] != "8.10" || childEnv[envRequireTenants] != "none" || childEnv[envBuild] != "0" {
			t.Fatalf("child release/tenant/build controls = %q/%q/%q, want 8.10/none/0", childEnv[envReleaseVersion], childEnv[envRequireTenants], childEnv[envBuild])
		}
		if _, duplicate := paths[spec.EvidencePath]; duplicate {
			t.Fatalf("duplicate evidence path %s", spec.EvidencePath)
		}
		paths[spec.EvidencePath] = struct{}{}
		if !strings.Contains(spec.EvidencePath, spec.Profile) || !strings.Contains(spec.EvidencePath, spec.SliceID) {
			t.Fatalf("evidence path %s lacks profile/slice isolation", spec.EvidencePath)
		}
	}
	summary := readOnlyReleaseSummary(t, cfg.EvidenceParent)
	if summary.Verdict != validation.VerdictPassed || !summary.CoverageComplete {
		t.Fatalf("summary = %+v, want complete pass", summary)
	}
}

// TestChildFailureDoesNotSuppressLaterIndependentSlice verifies ordinary
// failures remain durable while a later safe slice still collects evidence.
func TestChildFailureDoesNotSuppressLaterIndependentSlice(t *testing.T) {
	cfg := testRunnerConfig(t, []string{"dev810"})
	catalog, gaps := testPolicy("baseline-a", "baseline-b")
	execer := &fakeExecutor{handler: func(_ context.Context, spec commandSpec) commandResult {
		switch spec.Kind {
		case commandBuild:
			return commandResult{}
		case commandPreflight:
			return readinessResult("8.10.3")
		case commandSlice:
			if spec.SliceID == "baseline-a" {
				return commandResult{Stdout: goTestEvent("fail"), Stderr: []byte("first child failed"), ExitCode: 1, Err: errors.New("exit status 1")}
			}
			writePassingEvidence(t, spec, "8.10.3")
			return commandResult{Stdout: goTestEvent("pass")}
		}
		return commandResult{}
	}}

	exitCode, err := run(context.Background(), cfg, catalog, gaps, execer, fixedClock())
	if exitCode != exitFailed || err != nil {
		t.Fatalf("run exit/error = %d/%v, want 1/nil", exitCode, err)
	}
	if countKind(execer.commands(), commandSlice) != 2 {
		t.Fatalf("slice calls = %d, want 2", countKind(execer.commands(), commandSlice))
	}
	manifest := readOnlyManifest(t, cfg.EvidenceParent)
	if manifest.Slices[0].State != validation.SliceFailed || manifest.Slices[1].State != validation.SliceFinished {
		t.Fatalf("slice states = %q/%q, want failed/finished", manifest.Slices[0].State, manifest.Slices[1].State)
	}
	summary := readOnlyReleaseSummary(t, cfg.EvidenceParent)
	if summary.Verdict != validation.VerdictFailed || len(summary.Failures) == 0 {
		t.Fatalf("later success overwrote failure: %+v", summary)
	}
}

// TestMalformedTestEventsFailAZeroExitChild verifies process success alone
// cannot establish a valid slice result.
func TestMalformedTestEventsFailAZeroExitChild(t *testing.T) {
	cfg := testRunnerConfig(t, []string{"dev810"})
	catalog, gaps := testPolicy("baseline-get")
	execer := &fakeExecutor{handler: func(_ context.Context, spec commandSpec) commandResult {
		switch spec.Kind {
		case commandBuild:
			return commandResult{}
		case commandPreflight:
			return readinessResult("8.10.3")
		case commandSlice:
			writePassingEvidence(t, spec, "8.10.3")
			return commandResult{Stdout: []byte("not-json\n")}
		}
		return commandResult{}
	}}

	exitCode, err := run(context.Background(), cfg, catalog, gaps, execer, fixedClock())
	if exitCode != exitFailed || err != nil {
		t.Fatalf("run exit/error = %d/%v, want 1/nil", exitCode, err)
	}
	manifest := readOnlyManifest(t, cfg.EvidenceParent)
	if manifest.Slices[0].State != validation.SliceFailed || !strings.Contains(manifest.Slices[0].TerminationReason, "malformed") {
		t.Fatalf("slice = %+v, want malformed failure", manifest.Slices[0])
	}
}

// TestReadinessLossStopsLaterProfileMutations verifies a failed per-slice
// recheck prevents every remaining mutation-capable child for that target.
func TestReadinessLossStopsLaterProfileMutations(t *testing.T) {
	cfg := testRunnerConfig(t, []string{"dev810"})
	catalog, gaps := testPolicy("baseline-a", "baseline-b")
	preflightCalls := 0
	execer := &fakeExecutor{handler: func(_ context.Context, spec commandSpec) commandResult {
		switch spec.Kind {
		case commandBuild:
			return commandResult{}
		case commandPreflight:
			preflightCalls++
			if preflightCalls == 3 {
				return readinessResult("8.9.9")
			}
			return readinessResult("8.10.3")
		case commandSlice:
			writePassingEvidence(t, spec, "8.10.3")
			return commandResult{Stdout: goTestEvent("pass")}
		}
		return commandResult{}
	}}

	exitCode, err := run(context.Background(), cfg, catalog, gaps, execer, fixedClock())
	if exitCode != exitFailed || err != nil {
		t.Fatalf("run exit/error = %d/%v, want 1/nil", exitCode, err)
	}
	if countKind(execer.commands(), commandSlice) != 1 {
		t.Fatalf("slice calls = %d, want 1 after readiness loss", countKind(execer.commands(), commandSlice))
	}
	manifest := readOnlyManifest(t, cfg.EvidenceParent)
	if manifest.Slices[1].State != validation.SlicePlanned {
		t.Fatalf("later slice state = %q, want planned", manifest.Slices[1].State)
	}
}

// TestInterruptionFinalizesWithExit130 verifies cancellation terminates the
// active child, retains an incomplete manifest, and writes aggregate evidence.
func TestInterruptionFinalizesWithExit130(t *testing.T) {
	cfg := testRunnerConfig(t, []string{"dev810"})
	catalog, gaps := testPolicy("baseline-get")
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	execer := &fakeExecutor{handler: func(commandContext context.Context, spec commandSpec) commandResult {
		switch spec.Kind {
		case commandBuild:
			return commandResult{}
		case commandPreflight:
			return readinessResult("8.10.3")
		case commandSlice:
			close(started)
			<-commandContext.Done()
			return commandResult{ExitCode: -1, Err: commandContext.Err()}
		}
		return commandResult{}
	}}
	go func() {
		<-started
		cancel()
	}()

	exitCode, err := run(ctx, cfg, catalog, gaps, execer, fixedClock())
	if exitCode != exitInterrupted || !errors.Is(err, context.Canceled) {
		t.Fatalf("run exit/error = %d/%v, want 130/context canceled", exitCode, err)
	}
	manifest := readOnlyManifest(t, cfg.EvidenceParent)
	if manifest.State != manifestInterrupted || manifest.Slices[0].State != validation.SliceInterrupted || manifest.Slices[0].ExitCode != nil {
		t.Fatalf("interrupted manifest = %+v", manifest)
	}
	if _, statErr := os.Stat(filepath.Join(singleInvocationRoot(t, cfg.EvidenceParent), releaseFileName)); statErr != nil {
		t.Fatalf("release.json missing after interruption: %v", statErr)
	}
}

// TestManifestIsIncompleteBeforeChildCompletion verifies an uncatchable hard
// kill leaves only planned/running state, never a false finalized pass.
func TestManifestIsIncompleteBeforeChildCompletion(t *testing.T) {
	cfg := testRunnerConfig(t, []string{"dev810"})
	catalog, gaps := testPolicy("baseline-get")
	ctx, cancel := context.WithCancel(context.Background())
	execer := &fakeExecutor{handler: func(_ context.Context, spec commandSpec) commandResult {
		switch spec.Kind {
		case commandBuild:
			return commandResult{}
		case commandPreflight:
			return readinessResult("8.10.3")
		case commandSlice:
			manifest := readOnlyManifest(t, cfg.EvidenceParent)
			if manifest.State != manifestRunning || manifest.Slices[0].State == validation.SliceFinished {
				t.Fatalf("pre-completion manifest could be interpreted as final: %+v", manifest)
			}
			cancel()
			return commandResult{ExitCode: -1, Err: context.Canceled}
		}
		return commandResult{}
	}}

	exitCode, _ := run(ctx, cfg, catalog, gaps, execer, fixedClock())
	if exitCode != exitInterrupted {
		t.Fatalf("exit = %d, want 130", exitCode)
	}
}

// testRunnerConfig returns deterministic validated controls for one fake run.
func testRunnerConfig(t *testing.T, profiles []string) runnerConfig {
	t.Helper()
	return runnerConfig{
		RepoRoot: t.TempDir(), EvidenceParent: t.TempDir(), ReleaseVersion: "8.10",
		Profiles: profiles, TenantCoverage: validation.TenantCoverageNone,
		BuildBinary: true, Timeout: time.Minute, VolumeTimeout: time.Minute, RealTimeout: time.Minute,
	}
}

// testPolicy creates a minimal valid release catalog for runner orchestration
// tests without coupling them to every checked-in family.
func testPolicy(sliceIDs ...string) (validation.CoverageCatalog, validation.GapCatalog) {
	catalog := validation.CoverageCatalog{SchemaVersion: validation.SchemaVersion, Revision: "test.1"}
	for _, sliceID := range sliceIDs {
		catalog.Families = append(catalog.Families, validation.CoverageFamily{
			ID: sliceID, Slice: sliceID, ApplicableMinors: []string{"8.10"}, MinimumScenarios: 1,
		})
		catalog.Requirements = append(catalog.Requirements, validation.CoverageRequirement{
			ScenarioID: syntheticTestName(sliceID),
			Slice:      sliceID, Family: sliceID, ApplicableMinors: []string{"8.10"},
			Required: true, MinimumEvidence: validation.EvidenceReadOnly,
		})
	}
	return catalog, validation.GapCatalog{SchemaVersion: validation.SchemaVersion, Revision: catalog.Revision}
}

// syntheticTestName converts a safe slice identity into a stable Go test name.
func syntheticTestName(sliceID string) string {
	var name strings.Builder
	name.WriteString("Test")
	uppercaseNext := true
	for _, character := range sliceID {
		if character == '-' || character == '_' || character == '.' {
			uppercaseNext = true
			continue
		}
		if uppercaseNext && character >= 'a' && character <= 'z' {
			character -= 'a' - 'A'
		}
		name.WriteRune(character)
		uppercaseNext = false
	}
	return name.String()
}

// passingExecutor returns valid readiness, Go test events, and normalized
// evidence for every requested fake child.
func passingExecutor(t *testing.T, version string) *fakeExecutor {
	t.Helper()
	return &fakeExecutor{handler: func(_ context.Context, spec commandSpec) commandResult {
		switch spec.Kind {
		case commandBuild:
			return commandResult{}
		case commandPreflight:
			return readinessResult(version)
		case commandSlice:
			writePassingEvidence(t, spec, version)
			return commandResult{Stdout: goTestEvent("pass")}
		}
		return commandResult{}
	}}
}

// readinessResult encodes the stable fields consumed by the remote safety gate.
func readinessResult(version string) commandResult {
	return commandResult{Stdout: []byte(`{"ok":true,"cluster":{"gateway_version":"` + version + `"}}`)}
}

// writePassingEvidence materializes one current-run normalized record using
// the runner-owned child environment.
func writePassingEvidence(t *testing.T, spec commandSpec, version string) {
	t.Helper()
	env := environmentMap(spec.Env)
	records := []validation.ScenarioEvidence{{
		SchemaVersion: validation.SchemaVersion,
		RunID:         env["C8VOLT_IT_RUN_ID"], SliceID: spec.SliceID,
		ScenarioID: selectedTestFromArgs(t, spec.Args), Profile: spec.Profile,
		ObservedVersion: version, Outcome: validation.OutcomePassed,
		EvidenceLevel: validation.EvidenceReadOnly,
	}}
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal fake evidence: %v", err)
	}
	if err := os.WriteFile(spec.EvidencePath, data, 0o644); err != nil {
		t.Fatalf("write fake evidence: %v", err)
	}
}

// selectedTestFromArgs extracts the single synthetic test identity from the
// anchored runner-owned regex.
func selectedTestFromArgs(t *testing.T, args []string) string {
	t.Helper()
	for index, arg := range args {
		if arg == "-run" && index+1 < len(args) {
			return strings.TrimSuffix(strings.TrimPrefix(args[index+1], "^(?:"), ")$")
		}
	}
	t.Fatal("slice command is missing -run")
	return ""
}

// goTestEvent returns one valid JSONL event with the requested terminal action.
func goTestEvent(action string) []byte {
	return []byte(`{"Time":"2026-10-14T10:00:00Z","Action":"` + action + `","Package":"example"}` + "\n")
}

// fixedClock gives manifests stable nonzero timestamps while run IDs remain
// unique through their random suffix.
func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC) }
}

// countKind counts one subprocess phase in the fake executor trace.
func countKind(specs []commandSpec, kind commandKind) int {
	count := 0
	for _, spec := range specs {
		if spec.Kind == kind {
			count++
		}
	}
	return count
}

// singleInvocationRoot locates the one run directory created by a test.
func singleInvocationRoot(t *testing.T, parent string) string {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatalf("read invocation parent: %v", err)
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("invocation entries = %v, want one directory", entries)
	}
	return filepath.Join(parent, entries[0].Name())
}

// readOnlyManifest decodes the durable manifest without modifying test state.
func readOnlyManifest(t *testing.T, parent string) runnerManifest {
	t.Helper()
	var manifest runnerManifest
	readJSONFile(t, filepath.Join(singleInvocationRoot(t, parent), manifestFileName), &manifest)
	return manifest
}

// readOnlyReleaseSummary decodes the canonical aggregate result.
func readOnlyReleaseSummary(t *testing.T, parent string) validation.ReleaseSummary {
	t.Helper()
	var summary validation.ReleaseSummary
	readJSONFile(t, filepath.Join(singleInvocationRoot(t, parent), releaseFileName), &summary)
	return summary
}

// readJSONFile strictly decodes one test artifact and requires end-of-file.
func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		t.Fatalf("decode trailing %s: %v", path, err)
	}
}
