// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/stretchr/testify/require"
)

// TestCancelFamilySuspendedState exercises root cancellation and engine cascade
// from suspended/mixed fixtures through the existing cancel family slice.
func TestCancelFamilySuspendedState(t *testing.T) {
	runSuspendedMutationScenarios(t, "cancel")
}

// TestDeleteFamilySuspendedState checks non-force refusal and forced deletion of
// suspended/mixed fixtures through the existing delete family slice.
func TestDeleteFamilySuspendedState(t *testing.T) {
	runSuspendedMutationScenarios(t, "delete")
}

// runSuspendedMutationScenarios uses fresh owned instances for each mutation and
// wait mode. No resume occurs between the setup observations and postconditions.
func runSuspendedMutationScenarios(t *testing.T, operation string) {
	t.Helper()
	writeSuspendedRevisionEvidence(t, operation+" process-instance")
	for _, profile := range requireSelectedProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			if profile.ExpectedVersion != "8.10" {
				t.Skip("suspension scenario requires a disposable 8.10.4 server")
			}
			for _, shape := range []string{"suspended-root", "active-root-suspended-child", "suspended-root-active-child"} {
				for _, mode := range []string{"wait", "no-wait"} {
					t.Run(shape+"/"+mode, func(t *testing.T) {
						label := operation + "-" + shape + "-" + mode
						var fixture *suspendedFixture
						var root, selected, suspended string
						if shape == "suspended-root" {
							fixture = newSuspendedHeldFixture(t, profile, label, 1)
							root = fixture.dataset.PositiveProcessInstanceKeys[0]
							selected, suspended = root, root
						} else {
							var tree realStateCascadeDataset
							fixture, tree = newSuspendedCascadeFixture(t, profile, label)
							root, selected = tree.RootProcessInstanceKey, tree.MidProcessInstanceKey
							suspended = root
							if shape == "active-root-suspended-child" {
								suspended = selected
							}
						}
						require.NoError(t, fixture.transition(t, suspended, true))
						requireSuspendedMutationStates(t, fixture, suspended, "before")

						args := []string{operation, "pi", "--key", selected, "--automation", "--json"}
						outcome := "succeeded"
						if mode == "no-wait" {
							args = append(args, "--no-wait")
							outcome = "accepted"
						}
						if operation == "delete" {
							refused := fixture.run(t, "without-force", args...)
							requireSuspendedDeleteRefusal(t, refused)
							requireSuspendedMutationStates(t, fixture, suspended, "after-refusal")
							args = append(args, "--force")
						}
						result := fixture.run(t, "mutation", args...)
						require.NoError(t, result.Err, result.Stderr)
						require.Zero(t, result.ExitCode)
						var payload struct {
							Items []realStateCascadeDeleteItem `json:"items"`
						}
						decodeSuspendedLiveEnvelope(t, result.Stdout, outcome, &payload)
						require.Equal(t, []realStateCascadeDeleteItem{{Key: root, Ok: true}}, payload.Items,
							"child selection must report the resolved root")

						// Poll independently after --no-wait: accepted is not a completion
						// claim, but the engine must eventually finish the entire family.
						keys := fixture.dataset.PositiveProcessInstanceKeys
						if operation == "cancel" {
							for _, key := range keys {
								if mode == "no-wait" {
									require.NoError(t, requireProcessInstanceStateEventually(t, fixture.profile, key, []string{"CANCELED", "TERMINATED"}, 2*time.Minute))
								} else {
									require.NoError(t, requireProcessInstanceStateOneOf(t, fixture.profile, key, "CANCELED", "TERMINATED"))
								}
							}
						} else if mode == "no-wait" {
							require.NoError(t, requireProcessInstancesAbsentEventually(t, fixture.profile, keys, 2*time.Minute))
						} else {
							for _, key := range keys {
								require.NoError(t, requireProcessInstanceAbsent(t, fixture.profile, key))
							}
						}
						writeDataEvidence(t, "suspended-postcondition-"+fixture.name+".json", map[string]any{
							"operation": operation, "mode": mode, "rootKey": root, "familyKeys": keys,
							"commandOutcome": outcome, "postcondition": "verified",
						})
					})
				}
			}
		})
	}
}

// newSuspendedCascadeFixture reuses the embedded parent/mid/leaf seeder and the
// suspension helper, while owning only its root/family, never shared definitions.
func newSuspendedCascadeFixture(t *testing.T, profile integrationProfile, label string) (*suspendedFixture, realStateCascadeDataset) {
	t.Helper()
	profile = requireSuspendedProfile(t, profile)
	client, err := newLiveBatchClient(profile.Name)
	require.NoError(t, err)
	fixture := &suspendedFixture{profile: profile, client: client, name: profile.Name + "-" + label, resumeKeys: map[string]bool{}}
	t.Cleanup(func() { fixture.cleanup(t) })
	tree, records, err := seedRealStateCascadeDataset(t, profile, "suspended-"+label)
	fixture.records = records
	fixture.cleanupRootKey = tree.RootProcessInstanceKey
	fixture.dataset.PositiveProcessInstanceKeys = append([]string(nil), tree.FamilyProcessInstanceKeys...)
	// Retain the root for cleanup even if family discovery failed after creation.
	if len(fixture.dataset.PositiveProcessInstanceKeys) == 0 && tree.RootProcessInstanceKey != "" {
		fixture.dataset.PositiveProcessInstanceKeys = []string{tree.RootProcessInstanceKey}
	}
	writeDataEvidence(t, "suspended-fixture-"+fixture.name+".json", tree)
	require.NoError(t, err)
	require.NotEmpty(t, tree.RootProcessInstanceKey)
	require.NotEmpty(t, tree.MidProcessInstanceKey)
	require.GreaterOrEqual(t, len(tree.LeafProcessInstanceKeys), 2)
	return fixture, tree
}

// requireSuspendedMutationStates proves the mixed fixture before mutation and
// preserves all family states after refusal, recording both snapshots separately.
func requireSuspendedMutationStates(t *testing.T, fixture *suspendedFixture, suspendedKey, phase string) {
	t.Helper()
	observations := make(map[string]any)
	for _, key := range fixture.dataset.PositiveProcessInstanceKeys {
		state := "ACTIVE"
		if key == suspendedKey {
			state = "SUSPENDED"
		}
		observed, err := fixture.observe(t, key, state, key == suspendedKey)
		require.NoError(t, err)
		observations[key] = observed
	}
	writeDataEvidence(t, "suspended-states-"+fixture.name+"-"+phase+".json", observations)
}

// requireSuspendedDeleteRefusal distinguishes the force precondition from an
// unrelated transport failure and requires exactly one failed JSON envelope.
func requireSuspendedDeleteRefusal(t *testing.T, result commandResult) {
	t.Helper()
	require.Error(t, result.Err)
	require.Equal(t, exitcode.Error, result.ExitCode)
	var envelope struct {
		Outcome string `json:"outcome"`
		Class   string `json:"class"`
		Detail  struct {
			Message string `json:"message"`
		} `json:"detail"`
	}
	decoder := json.NewDecoder(strings.NewReader(result.Stdout))
	require.NoError(t, decoder.Decode(&envelope))
	require.Equal(t, io.EOF, decoder.Decode(new(any)))
	require.Equal(t, "failed", envelope.Outcome)
	require.Equal(t, "local_precondition", envelope.Class)
	require.Contains(t, envelope.Detail.Message, "not in a final state")
	require.Contains(t, envelope.Detail.Message, "no delete request was submitted")
}
