// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/stretchr/testify/require"
)

// lifecyclePayload is the common JSON projection for suspension and resumption.
// A pointer distinguishes an omitted suspendedDate from an empty string.
type lifecyclePayload struct {
	Items []lifecycleItem `json:"items"`
	Plan  *struct {
		RequestedKeys []string `json:"requestedKeys"`
		TargetKeys    []string `json:"targetKeys"`
		Instances     []struct {
			Key            string `json:"key"`
			ObservedState  string `json:"observedState"`
			Classification string `json:"classification"`
		} `json:"instances"`
	} `json:"plan"`
}

type lifecycleItem struct {
	Key               string  `json:"key"`
	Status            string  `json:"status"`
	Submission        string  `json:"submission"`
	Confirmation      string  `json:"confirmation"`
	StatusCode        int     `json:"statusCode"`
	HasObservation    bool    `json:"hasObservation"`
	LastObservedState string  `json:"lastObservedState"`
	SuspendedDate     *string `json:"suspendedDate"`
	Error             string  `json:"error"`
}

// TestProcessInstanceLifecycle uses only owned held instances on an explicitly
// selected 8.10.4 profile. Each scenario cleans up before the next one starts.
func TestProcessInstanceLifecycle(t *testing.T) {
	writeSuspendedRevisionEvidence(t)
	for _, operation := range []string{"suspend", "resume"} {
		runFamilyCoverageScenarios(t, operation, []string{operation, operation + " process-instance"})
	}
	for _, profile := range requireSelectedProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			if profile.ExpectedVersion != "8.10" {
				t.Skip("process-instance lifecycle requires an explicitly selected 8.10.4 profile")
			}
			t.Run("explicit-round-trip", func(t *testing.T) { testLifecycleRoundTrip(t, profile) })
			t.Run("tree-boundaries", func(t *testing.T) { testLifecycleTree(t, profile) })
			t.Run("partial-rejection", func(t *testing.T) { testLifecyclePartialRejection(t, profile) })
		})
	}
}

func testLifecycleRoundTrip(t *testing.T, profile integrationProfile) {
	f := newSuspendedFixture(t, profile)
	keys := f.dataset.PositiveProcessInstanceKeys
	targets := keys[:2]
	requireLifecycleStates(t, f, nil)
	for _, operation := range []string{"suspend", "resume"} {
		before := []string(nil)
		after := targets
		if operation == "resume" {
			before, after = targets, nil
		}
		preview := lifecycleCommand(t, f, operation+"-preview", operation, targets, "--dry-run")
		requireLifecycleItems(t, preview, targets, "preview", "")
		require.Nil(t, preview.Plan)
		requireLifecycleStates(t, f, before)
		result := lifecycleCommand(t, f, operation+"-execute", operation, targets)
		state := "SUSPENDED"
		if operation == "resume" {
			state = "ACTIVE"
		}
		requireLifecycleItems(t, result, targets, "confirmed", state)
		require.Nil(t, result.Plan)
		requireLifecycleStates(t, f, after) // Includes the untouched third instance.
	}
}

func testLifecycleTree(t *testing.T, profile integrationProfile) {
	f, root, mid, leaf, sibling := newLifecycleTreeFixture(t, profile)
	requireLifecycleStates(t, f, nil)
	// Exact selection must neither escalate to the root nor cascade to the child.
	for _, operation := range []string{"suspend", "resume"} {
		state, suspended := "SUSPENDED", []string{mid}
		if operation == "resume" {
			state, suspended = "ACTIVE", nil
		}
		result := lifecycleCommand(t, f, "exact-mid-"+operation, operation, []string{mid})
		requireLifecycleItems(t, result, []string{mid}, "confirmed", state)
		require.Nil(t, result.Plan)
		requireLifecycleStates(t, f, suspended)
		if operation == "suspend" {
			// A suspended parent must not hide its active child from planning.
			preview := lifecycleCommand(t, f, "suspended-parent-preview", "suspend", []string{mid}, "--tree", "--dry-run")
			requireLifecyclePlan(t, preview, []string{mid}, []string{mid, leaf}, []string{leaf})
			requireLifecycleItems(t, preview, []string{leaf}, "preview", "")
			requireLifecycleStates(t, f, suspended)
		}
	}
	// Prepare one independently suspended descendant. Overlapping seeds must
	// appear once; the already-suspended leaf must not be submitted again.
	require.NoError(t, f.transition(t, leaf, true))
	requireLifecycleStates(t, f, []string{leaf})
	seeds := []string{mid, leaf}
	preview := lifecycleCommand(t, f, "tree-suspend-preview", "suspend", seeds, "--tree", "--dry-run")
	requireLifecyclePlan(t, preview, seeds, []string{mid, leaf}, []string{mid})
	requireLifecycleItems(t, preview, []string{mid}, "preview", "")
	requireLifecycleStates(t, f, []string{leaf})
	suspended := lifecycleCommand(t, f, "tree-suspend", "suspend", seeds, "--tree")
	requireLifecyclePlan(t, suspended, seeds, []string{mid, leaf}, []string{mid})
	requireLifecycleItems(t, suspended, []string{mid}, "confirmed", "SUSPENDED")
	requireLifecycleStates(t, f, []string{mid, leaf})
	empty := lifecycleCommand(t, f, "tree-suspend-empty", "suspend", seeds, "--tree")
	requireLifecyclePlan(t, empty, seeds, []string{mid, leaf}, nil)
	require.NotNil(t, empty.Items)
	require.Empty(t, empty.Items)
	requireLifecycleStates(t, f, []string{mid, leaf})

	// Restore only the intermediate node: resumption planning must traverse it
	// while ACTIVE to reach the independently suspended leaf.
	resumed := lifecycleCommand(t, f, "exact-mid-restore", "resume", []string{mid})
	requireLifecycleItems(t, resumed, []string{mid}, "confirmed", "ACTIVE")
	requireLifecycleStates(t, f, []string{leaf})
	seeds = []string{mid} // The leaf must be discovered through its active parent.
	preview = lifecycleCommand(t, f, "tree-resume-preview", "resume", seeds, "--tree", "--dry-run")
	requireLifecyclePlan(t, preview, seeds, []string{mid, leaf}, []string{leaf})
	requireLifecycleItems(t, preview, []string{leaf}, "preview", "")
	requireLifecycleStates(t, f, []string{leaf})
	resumed = lifecycleCommand(t, f, "tree-resume", "resume", seeds, "--tree")
	requireLifecyclePlan(t, resumed, seeds, []string{mid, leaf}, []string{leaf})
	requireLifecycleItems(t, resumed, []string{leaf}, "confirmed", "ACTIVE")
	requireLifecycleStates(t, f, nil)
	empty = lifecycleCommand(t, f, "tree-resume-empty", "resume", seeds, "--tree")
	requireLifecyclePlan(t, empty, seeds, []string{mid, leaf}, nil)
	require.NotNil(t, empty.Items)
	require.Empty(t, empty.Items)
	requireLifecycleStates(t, f, nil)
	require.NotContains(t, seeds, root)
	require.NotContains(t, seeds, sibling)
}

func testLifecyclePartialRejection(t *testing.T, profile integrationProfile) {
	for _, operation := range []string{"suspend", "resume"} {
		t.Run(operation, func(t *testing.T) {
			f := newSuspendedFixture(t, profile)
			keys := f.dataset.PositiveProcessInstanceKeys
			valid, invalid := keys[0], keys[1]
			suspended := invalid
			state := "SUSPENDED"
			if operation == "resume" {
				suspended, state = valid, "ACTIVE"
			}
			require.NoError(t, f.transition(t, suspended, true))
			requireLifecycleStates(t, f, []string{suspended})
			// Reject first, then succeed: proves ordinary collection execution continues.
			targets := []string{invalid, valid}
			for _, key := range targets {
				f.resumeKeys[key] = true
			}
			result := f.run(t, operation+"-partial", operation, "pi", "--key", strings.Join(targets, ","), "--workers", "1", "--fail-fast=false", "--automation", "--json")
			require.Error(t, result.Err)
			require.Equal(t, exitcode.Conflict, result.ExitCode)
			require.Empty(t, result.Stderr, "JSON failure must not emit a second diagnostic")
			var payload lifecyclePayload
			decodeSuspendedLiveEnvelope(t, result.Stdout, "failed", &payload)
			var envelope struct {
				Class string `json:"class"`
			}
			require.NoError(t, json.Unmarshal([]byte(result.Stdout), &envelope))
			require.Equal(t, "conflict", envelope.Class)
			require.Len(t, payload.Items, 2)
			rejected := payload.Items[0]
			require.Equal(t, invalid, rejected.Key)
			require.Equal(t, "failed", rejected.Status)
			require.Equal(t, "rejected", rejected.Submission)
			require.Equal(t, "not_requested", rejected.Confirmation)
			require.Equal(t, 409, rejected.StatusCode)
			require.False(t, rejected.HasObservation)
			require.Empty(t, rejected.LastObservedState)
			require.Nil(t, rejected.SuspendedDate)
			require.NotEmpty(t, rejected.Error)
			requireLifecycleItems(t, lifecyclePayload{Items: payload.Items[1:]}, []string{valid}, "confirmed", state)
			expected := targets
			if operation == "resume" {
				expected = nil
			}
			requireLifecycleStates(t, f, expected)
		})
	}
}

func lifecycleCommand(t *testing.T, f *suspendedFixture, name, operation string, keys []string, extra ...string) lifecyclePayload {
	t.Helper()
	for _, key := range keys {
		require.Contains(t, f.dataset.PositiveProcessInstanceKeys, key)
	}
	// Tree mutations may affect any owned descendant. Track before submission,
	// including when a failing command leaves its actual outcome uncertain.
	for _, key := range f.dataset.PositiveProcessInstanceKeys {
		f.resumeKeys[key] = true
	}
	args := []string{operation, "pi", "--key", strings.Join(keys, ","), "--automation", "--json"}
	args = append(args, extra...)
	result := f.run(t, name, args...)
	require.NoError(t, result.Err, result.Stderr)
	require.Zero(t, result.ExitCode)
	require.Empty(t, result.Stderr)
	var payload lifecyclePayload
	decodeSuspendedLiveEnvelope(t, result.Stdout, "succeeded", &payload)
	return payload
}

func requireLifecycleItems(t *testing.T, payload lifecyclePayload, keys []string, status, state string) {
	t.Helper()
	require.Len(t, payload.Items, len(keys))
	for i, item := range payload.Items {
		require.Equal(t, keys[i], item.Key)
		require.Equal(t, status, item.Status)
		require.Empty(t, item.Error)
		if status == "preview" {
			require.Equal(t, "not_submitted", item.Submission)
			require.Equal(t, "not_requested", item.Confirmation)
			require.Zero(t, item.StatusCode)
			require.False(t, item.HasObservation)
			require.Empty(t, item.LastObservedState)
			require.Nil(t, item.SuspendedDate)
			continue
		}
		require.Equal(t, "accepted", item.Submission)
		require.Equal(t, "confirmed", item.Confirmation)
		require.Equal(t, 204, item.StatusCode)
		require.True(t, item.HasObservation)
		require.Equal(t, state, item.LastObservedState)
		if state == "SUSPENDED" {
			require.NotNil(t, item.SuspendedDate)
			require.NotEmpty(t, *item.SuspendedDate)
		} else {
			require.Nil(t, item.SuspendedDate)
		}
	}
}

func requireLifecyclePlan(t *testing.T, payload lifecyclePayload, seeds, scope, targets []string) {
	t.Helper()
	require.NotNil(t, payload.Plan)
	require.Equal(t, seeds, payload.Plan.RequestedKeys)
	require.ElementsMatch(t, targets, payload.Plan.TargetKeys)
	var actual []string
	for _, item := range payload.Plan.Instances {
		actual = append(actual, item.Key)
		require.NotEmpty(t, item.ObservedState)
		if containsString(targets, item.Key) {
			require.Equal(t, "eligible", item.Classification)
		} else {
			require.Contains(t, []string{"excluded", "already_suspended"}, item.Classification)
		}
	}
	require.ElementsMatch(t, scope, actual)
}

func requireLifecycleStates(t *testing.T, f *suspendedFixture, suspended []string) {
	t.Helper()
	for _, key := range f.dataset.PositiveProcessInstanceKeys {
		state := "ACTIVE"
		for _, candidate := range suspended {
			if key == candidate {
				state = "SUSPENDED"
			}
		}
		observed, err := f.observe(t, key, state, state == "SUSPENDED")
		require.NoError(t, err)
		require.Equal(t, key, string(observed.ProcessInstanceKey))
	}
}

// TestProcessInstanceLifecycleCatalog is local-only: it checks registration of
// the dedicated lifecycle assertions without selecting profiles or seeding data.
func TestProcessInstanceLifecycleCatalog(t *testing.T) {
	manifest := make(map[string]coverageEntry)
	for _, operation := range []string{"suspend", "resume"} {
		path := operation + " process-instance"
		entry, ok := commandCoverageManifest[path]
		require.True(t, ok, "missing lifecycle command %s", path)
		manifest[path] = entry
	}
	requireBehaviorScenarioCatalog(t, lifecycleBehaviorScenarios(), manifest)
}

// lifecycleBehaviorScenarios registers the explicit-round-trip assertions in the
// catalog without executing mutations twice through the generic scenario runner.
func lifecycleBehaviorScenarios() []behaviorScenario {
	var scenarios []behaviorScenario
	for _, operation := range []string{"suspend", "resume"} {
		for _, preview := range []bool{true, false} {
			behavior := behaviorConfirmedMutation
			flags := []string{"key"}
			if preview {
				behavior = behaviorPreview
				flags = append(flags, "dry-run")
			}
			scenarios = append(scenarios, behaviorScenario{
				CommandPath:     operation + " process-instance",
				Name:            "TestProcessInstanceLifecycle/explicit-round-trip/" + operation + "-" + behavior,
				RequiresProfile: true, OutputMode: "json", Behavior: behavior,
				VersionBehavior: "observed-8.10.4-before-mutation", DataOwnership: []string{"seeded"},
				Preview: preview, ConfirmedMutation: !preview, WantJSON: true, CoveredFlags: flags,
			})
		}
	}
	return scenarios
}
