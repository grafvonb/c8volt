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
// Generated mutation calls are fixture-only until public suspension workflows exist.
type suspendedFixture struct {
	profile    integrationProfile
	client     liveBatchClient
	dataset    volumeDataset
	resumeKeys map[string]bool
	records    []evidenceRecord
	name       string
	// Cascade definitions are shared; only the run-owned root is cleaned up.
	cleanupRootKey string
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
	return newSuspendedHeldFixture(t, profile, "", 3)
}

// newSuspendedHeldFixture reuses the held model with distinct scenario identities.
func newSuspendedHeldFixture(t *testing.T, profile integrationProfile, label string, count int) *suspendedFixture {
	t.Helper()
	profile = requireSuspendedProfile(t, profile)
	client, err := newLiveBatchClient(profile.Name)
	require.NoError(t, err)
	name, seedLabel := profile.Name, "suspended"
	if label != "" {
		name += "-" + label
		seedLabel += "-" + label
	}
	fixture := &suspendedFixture{profile: profile, client: client, name: name, resumeKeys: map[string]bool{}}
	t.Cleanup(func() { fixture.cleanup(t) })
	fixture.dataset, fixture.records, err = seedIsolatedHeldDataset(t, profile, seedLabel, count)
	writeDataEvidence(t, "suspended-fixture-"+fixture.name+".json", fixture.dataset)
	require.NoError(t, err)
	require.Len(t, fixture.dataset.PositiveProcessInstanceKeys, count)
	require.Len(t, fixture.dataset.PositiveProcessDefinitionKeys, 1)
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
			writeDataEvidence(t, fmt.Sprintf("suspended-observation-%s-%s-%s.json", f.name, key, state), last)
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
	if result.Err != nil {
		record.Outcome = "nonzero"
	}
	f.records = append(f.records, record)
	return result
}

// cleanup resumes only surviving suspended instances, then removes the owned
// definition or cascade root. Cleanup remains separate from acceptance evidence.
func (f *suspendedFixture) cleanup(t *testing.T) {
	t.Helper()
	cleanup := cleanupRecord{Profile: f.profile.Name, ProcessInstanceKeys: f.dataset.PositiveProcessInstanceKeys, Attempted: true, Status: "cleaned"}
	var failures []string
	for key := range f.resumeKeys {
		state, _, err := f.cleanupState(t, key)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if state != "SUSPENDED" {
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
	if len(failures) == 0 {
		if f.cleanupRootKey != "" {
			_, exists, err := f.cleanupState(t, f.cleanupRootKey)
			if err != nil {
				failures = append(failures, err.Error())
			} else if exists {
				result := f.run(t, "cleanup-root", "delete", "pi", "--key", f.cleanupRootKey, "--force", "--automation", "--json")
				if result.Err != nil {
					failures = append(failures, "delete owned root "+f.cleanupRootKey+": "+result.Stderr)
				}
			}
		}
		for _, key := range f.dataset.PositiveProcessDefinitionKeys {
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
	}
	if len(failures) > 0 {
		cleanup.Status, cleanup.Reason = "cleanup_failed", strings.Join(failures, "; ")
		t.Errorf("suspended fixture cleanup failed separately from acceptance: %s", cleanup.Reason)
	}
	writeJSON(t, "suspended-cleanup-"+f.name+".json", cleanup)
	writeEvidenceRecords(t, "suspended-commands-"+f.name+".json", f.records)
}

// cleanupState distinguishes already canceled/deleted fixtures from instances
// needing resumption without interpreting authorization or transport errors as absence.
func (f *suspendedFixture) cleanupState(t *testing.T, key string) (string, bool, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	response, err := f.client.generated.GetProcessInstanceWithResponse(ctx, key)
	if err != nil {
		return "", false, err
	}
	if response == nil {
		return "", false, fmt.Errorf("cleanup observation %s returned no response", key)
	}
	writeDataEvidence(t, fmt.Sprintf("suspended-cleanup-observation-%s-%s.json", f.name, key), map[string]any{"key": key, "status": response.StatusCode(), "instance": response.JSON200})
	if response.StatusCode() == 404 {
		return "", false, nil
	}
	if response.StatusCode() != 200 || response.JSON200 == nil {
		return "", false, fmt.Errorf("cleanup observation %s returned HTTP %d without an instance", key, response.StatusCode())
	}
	return string(response.JSON200.State), true, nil
}

// writeSuspendedRevisionEvidence captures source/provenance without configuration secrets.
func writeSuspendedRevisionEvidence(t *testing.T, commands ...string) {
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
	if len(commands) == 0 {
		commands = []string{"get process-instance", "expect process-instance"}
	}
	writeCommandProposals(t, registerDirectCamundaSetupFallback(nil, "SUSPENDED with optional cleanup resumption", "SuspendProcessInstance/ResumeProcessInstance generated calls create fixture state unavailable through the CLI", commands, []string{"8.10"}, "Fixture setup and cleanup only; no public suspend/resume command is exercised"))
}
