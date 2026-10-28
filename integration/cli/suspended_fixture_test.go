// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	"github.com/stretchr/testify/require"
)

// suspendedFixture owns an isolated held dataset and its setup/cleanup evidence.
// Generated mutations remain setup/cleanup controls; intended resumption coverage uses the CLI.
type suspendedFixture struct {
	name         string
	observations int
	profile      integrationProfile
	client       liveBatchClient
	dataset      volumeDataset
	resumeKeys   map[string]bool
	records      []evidenceRecord
}

// requireSuspendedProfile checks the full observed patch before any setup mutation.
func requireSuspendedProfile(t *testing.T, profile integrationProfile) integrationProfile {
	t.Helper()
	ready, err := checkProfileReadiness(t, profile)
	writeJSON(t, "suspended-profile-"+profile.Name+".json", ready)
	require.NoError(t, err)
	require.Equal(t, "8.10", ready.ExpectedVersion, "suspension fixture requires configured minor 8.10")
	require.Equal(t, "8.10.4", ready.ActualVersion, "suspension acceptance requires observed server patch 8.10.4 before mutation")
	return ready
}

// newSuspendedFixture registers cleanup before seeding so partial setup is auditable.
func newSuspendedFixture(t *testing.T, profile integrationProfile) *suspendedFixture {
	t.Helper()
	fixture := newSuspendedFixtureBase(t, profile)
	var err error
	fixture.dataset, fixture.records, err = seedIsolatedHeldDataset(t, fixture.profile, sanitizeEvidenceName(t.Name()), 3)
	writeDataEvidence(t, "suspended-fixture-"+fixture.name+".json", fixture.dataset)
	require.NoError(t, err)
	require.Len(t, fixture.dataset.PositiveProcessInstanceKeys, 3)
	require.Len(t, fixture.dataset.PositiveProcessDefinitionKeys, 1)
	return fixture
}

func newSuspendedFixtureBase(t *testing.T, profile integrationProfile) *suspendedFixture {
	t.Helper()
	profile = requireSuspendedProfile(t, profile)
	client, err := newLiveBatchClient(profile.Name)
	require.NoError(t, err)
	fixture := &suspendedFixture{name: sanitizeEvidenceName(t.Name()), profile: profile, client: client, resumeKeys: map[string]bool{}}
	t.Cleanup(func() { fixture.cleanup(t) })
	return fixture
}

// transition changes only owned fixture instances and saves HTTP setup evidence.
func (f *suspendedFixture) transition(t *testing.T, key string, suspend bool) error {
	t.Helper()
	if !slices.Contains(f.dataset.PositiveProcessInstanceKeys, key) {
		return fmt.Errorf("refusing fixture transition for unowned key %q", key)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	operation := "resume"
	status := 0
	var body []byte
	var err error
	if suspend {
		operation = "suspend"
		// Track before submission: a transport error can still follow an accepted mutation.
		f.resumeKeys[key] = true
		response, callErr := f.client.generated.SuspendProcessInstanceWithResponse(ctx, key, camundav810.SuspendProcessInstanceJSONRequestBody{})
		err = callErr
		if response != nil {
			status, body = response.StatusCode(), response.Body
		}
	} else {
		response, callErr := f.client.generated.ResumeProcessInstanceWithResponse(ctx, key, camundav810.ResumeProcessInstanceJSONRequestBody{})
		err = callErr
		if response != nil {
			status, body = response.StatusCode(), response.Body
		}
	}
	writeDataEvidence(t, fmt.Sprintf("suspended-%s-%s-%s.json", f.name, operation, key), map[string]any{"operation": operation, "key": key, "status": status, "body": string(body), "transportFailed": err != nil})
	if err != nil {
		return err
	}
	if status != 204 {
		return fmt.Errorf("fixture %s %s returned HTTP %d: %s", operation, key, status, body)
	}
	return nil
}

// resumeWithCLI preserves ownership and cleanup tracking before exercising public resumption.
// Only the independent observation clears tracking, even when command confirmation succeeds.
func (f *suspendedFixture) resumeWithCLI(t *testing.T, key string) commandResult {
	t.Helper()
	require.Contains(t, f.dataset.PositiveProcessInstanceKeys, key, "refusing CLI resumption for unowned key")
	f.resumeKeys[key] = true
	return f.run(t, "resume-"+key, "resume", "pi", "--key", key, "--automation", "--json")
}

// observe polls the generated read independently of the facade under test.
func (f *suspendedFixture) observe(t *testing.T, key, state string, suspendedDate bool) (camundav810.ProcessInstanceResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var last camundav810.ProcessInstanceResult
	for attempt := 1; ; attempt++ {
		response, err := f.client.generated.GetProcessInstanceWithResponse(ctx, key)
		if err != nil {
			return last, err
		}
		if response == nil || response.JSON200 == nil {
			return last, fmt.Errorf("fixture observation %s returned no instance", key)
		}
		last = *response.JSON200
		if string(last.State) == state && (last.SuspendedDate != nil && !last.SuspendedDate.IsZero()) == suspendedDate {
			if state == "ACTIVE" && !suspendedDate {
				delete(f.resumeKeys, key)
			}
			f.observations++
			writeDataEvidence(t, fmt.Sprintf("suspended-observation-%s-%03d-%s-%s.json", f.name, f.observations, key, state), last)
			return last, nil
		}
		select {
		case <-ctx.Done():
			return last, fmt.Errorf("fixture observation %s after %d attempts: %w", key, attempt, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// run retains separate command streams and exit codes even when an assertion fails.
func (f *suspendedFixture) run(t *testing.T, name string, args ...string) commandResult {
	t.Helper()
	result := runC8VoltForProfile(t, f.profile.Name, "suspended-"+f.name+"-"+name, args...)
	record := commandEvidence(strings.Join(args[:2], " "), name, result, "pass")
	record.Profile, record.CamundaVersion = f.profile.Name, f.profile.ActualVersion
	record.DataOwnership = []string{"seeded"}
	if args[0] == "suspend" || args[0] == "resume" {
		record.CommandPath = args[0] + " process-instance"
		record.OutputMode = "json"
		for _, arg := range args {
			if strings.HasPrefix(arg, "--") {
				record.CoveredFlags = append(record.CoveredFlags, strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)[0])
			}
		}
		record.VersionBehavior = "observed-8.10.4-before-mutation"
		record.Preview = slices.Contains(args, "--dry-run")
		if record.Preview {
			record.Behavior = behaviorPreview
		} else if result.Err == nil {
			// Empty tree success is not a confirmed mutation.
			var envelope struct {
				Payload struct {
					Items []struct {
						Confirmation string `json:"confirmation"`
					} `json:"items"`
				} `json:"payload"`
			}
			if json.Unmarshal([]byte(result.Stdout), &envelope) == nil {
				for _, item := range envelope.Payload.Items {
					if item.Confirmation == "confirmed" {
						record.ConfirmedMutation = true
					}
				}
			}
			record.Behavior = behaviorSuccess
			if record.ConfirmedMutation {
				record.Behavior = behaviorConfirmedMutation
			}
		}
	}
	if result.Err != nil {
		record.Outcome = "nonzero"
	}
	f.records = append(f.records, record)
	return result
}

// cleanup resumes potentially suspended instances before ordinary owned-definition
// deletion. Failed cleanup is recorded separately and never masks scenario evidence.
func (f *suspendedFixture) cleanup(t *testing.T) {
	t.Helper()
	cleanup := cleanupRecord{Profile: f.profile.Name, ProcessInstanceKeys: f.dataset.PositiveProcessInstanceKeys, Attempted: true, Status: "cleaned"}
	var failures []string
	for key := range f.resumeKeys {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		response, err := f.client.generated.GetProcessInstanceWithResponse(ctx, key)
		cancel()
		if err != nil || response == nil {
			failures = append(failures, fmt.Sprintf("inspect owned instance %s before cleanup: %v", key, err))
			continue
		}
		if response.StatusCode() == 404 {
			continue
		}
		if response.JSON200 == nil {
			failures = append(failures, fmt.Sprintf("inspect owned instance %s before cleanup: HTTP %d", key, response.StatusCode()))
			continue
		}
		if string(response.JSON200.State) != "SUSPENDED" {
			continue
		}
		if err := f.transition(t, key, false); err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if _, err := f.observe(t, key, "ACTIVE", false); err != nil {
			failures = append(failures, err.Error())
		}
	}
	// Always attempt owned deletion, even when recovery failed for another key.
	// A cleanup failure remains evidence; it must not strand unrelated fixtures.
	for i := len(f.dataset.PositiveProcessDefinitionKeys) - 1; i >= 0; i-- {
		key := f.dataset.PositiveProcessDefinitionKeys[i]
		result := f.run(t, "cleanup-"+key, "delete", "pd", "--key", key, "--force", "--automation", "--json")
		if result.Err != nil {
			failures = append(failures, "delete owned definition "+key+": "+result.Stderr)
		}
	}
	for _, key := range f.dataset.PositiveProcessInstanceKeys {
		if err := requireProcessInstanceAbsentEventually(t, f.profile, key, 30*time.Second); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		cleanup.Status, cleanup.Reason = "cleanup_failed", strings.Join(failures, "; ")
		t.Errorf("suspended fixture cleanup failed separately from acceptance: %s", cleanup.Reason)
	}
	writeJSON(t, "suspended-cleanup-"+f.name+".json", cleanup)
	writeEvidenceRecords(t, "suspended-commands-"+f.name+".json", f.records)
}

// writeSuspendedRevisionEvidence captures source/provenance without configuration secrets.
func writeSuspendedRevisionEvidence(t *testing.T) {
	t.Helper()
	git := func(args ...string) string {
		command := exec.Command("git", args...)
		command.Dir = suite.repoRoot
		output, err := command.Output()
		require.NoError(t, err)
		return strings.TrimSpace(string(output))
	}
	provenance, err := os.ReadFile(filepath.Join(suite.repoRoot, "internal/clients/camunda/v810/camunda/provenance.json"))
	require.NoError(t, err)
	writeJSON(t, "suspended-revision.json", map[string]any{"revision": git("rev-parse", "HEAD"), "dirtyStatus": git("status", "--short", "--untracked-files=all"), "clientProvenance": json.RawMessage(provenance)})
	writeCommandProposals(t, registerDirectCamundaSetupFallback(nil, "SUSPENDED setup and cleanup recovery", "Generated suspension prepares the held fixture; generated resumption is retained only for cleanup recovery independent of the CLI under test", []string{"get process-instance", "expect process-instance", "resume process-instance"}, []string{"8.10"}, "The scenario resumes through resume pi and asserts accepted/confirmed evidence; direct calls are setup/cleanup controls, not CLI acceptance"))
}
