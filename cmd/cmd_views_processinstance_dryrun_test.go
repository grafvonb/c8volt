// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type cancelDryRunPreviewFixture struct {
	RequestedKeys      typex.Keys
	ResolvedRoots      typex.Keys
	AffectedFamilyKeys typex.Keys
	TraversalOutcome   process.TraversalOutcome
	Warning            string
	MissingAncestors   []process.MissingAncestor
}

// newCancelDryRunPreviewFixture returns the shared cancel dry-run payload fixture.
func newCancelDryRunPreviewFixture() cancelDryRunPreviewFixture {
	return cancelDryRunPreviewFixture{
		RequestedKeys:      typex.Keys{"2251799813711967"},
		ResolvedRoots:      typex.Keys{"2251799813711900"},
		AffectedFamilyKeys: typex.Keys{"2251799813711900", "2251799813711967"},
		TraversalOutcome:   process.TraversalOutcomePartial,
		Warning:            "one or more parent process instances were not found",
		MissingAncestors:   []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
	}
}

// requireCancelDryRunPreviewPayload verifies the stable cancel dry-run JSON payload shape.
func requireCancelDryRunPreviewPayload(t *testing.T, payload map[string]any, want cancelDryRunPreviewFixture) {
	t.Helper()

	require.Equal(t, "cancel", payload["operation"])
	requireDryRunPreviewStringSlice(t, payload, "requestedKeys", want.RequestedKeys)
	requireDryRunPreviewStringSlice(t, payload, "resolvedRoots", want.ResolvedRoots)
	requireDryRunPreviewStringSlice(t, payload, "affectedFamilyKeys", want.AffectedFamilyKeys)
	require.Equal(t, float64(len(want.RequestedKeys)), payload["requestedCount"])
	require.Equal(t, float64(len(want.ResolvedRoots)), payload["resolvedRootCount"])
	require.Equal(t, float64(len(want.AffectedFamilyKeys)), payload["affectedCount"])
	require.Equal(t, string(want.TraversalOutcome), payload["traversalOutcome"])
	require.Equal(t, want.TraversalOutcome == process.TraversalOutcomeComplete, payload["scopeComplete"])
	require.Equal(t, want.Warning, payload["warning"])
	requireDryRunPreviewMissingAncestors(t, payload, want.MissingAncestors)
	require.Equal(t, false, payload["mutationSubmitted"])
}

type deleteDryRunPreviewFixture struct {
	RequestedKeys      typex.Keys
	ResolvedRoots      typex.Keys
	AffectedFamilyKeys typex.Keys
	RequiresCancel     []process.ProcessInstance
	TraversalOutcome   process.TraversalOutcome
	Warning            string
	MissingAncestors   []process.MissingAncestor
}

// newDeleteDryRunPreviewFixture returns the shared delete dry-run payload fixture.
func newDeleteDryRunPreviewFixture() deleteDryRunPreviewFixture {
	return deleteDryRunPreviewFixture{
		RequestedKeys:      typex.Keys{"2251799813711967"},
		ResolvedRoots:      typex.Keys{"2251799813711900"},
		AffectedFamilyKeys: typex.Keys{"2251799813711900", "2251799813711967"},
		RequiresCancel:     []process.ProcessInstance{{Key: "2251799813711900", State: process.StateActive}},
		TraversalOutcome:   process.TraversalOutcomePartial,
		Warning:            "one or more parent process instances were not found",
		MissingAncestors:   []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
	}
}

// requireDeleteDryRunPreviewPayload verifies the stable delete dry-run JSON payload shape.
func requireDeleteDryRunPreviewPayload(t *testing.T, payload map[string]any, want deleteDryRunPreviewFixture) {
	t.Helper()

	require.Equal(t, "delete", payload["operation"])
	requireDryRunPreviewStringSlice(t, payload, "requestedKeys", want.RequestedKeys)
	requireDryRunPreviewStringSlice(t, payload, "resolvedRoots", want.ResolvedRoots)
	requireDryRunPreviewStringSlice(t, payload, "affectedFamilyKeys", want.AffectedFamilyKeys)
	require.Equal(t, float64(len(want.RequestedKeys)), payload["requestedCount"])
	require.Equal(t, float64(len(want.ResolvedRoots)), payload["resolvedRootCount"])
	require.Equal(t, float64(len(want.AffectedFamilyKeys)), payload["affectedCount"])
	require.Equal(t, float64(len(want.RequiresCancel)), payload["requiresCancelBeforeDeleteCount"])
	requiresCancel, ok := payload["requiresCancelBeforeDelete"].([]any)
	require.True(t, ok)
	require.Len(t, requiresCancel, len(want.RequiresCancel))
	for i, item := range want.RequiresCancel {
		got, ok := requiresCancel[i].(map[string]any)
		require.True(t, ok)
		require.Equal(t, item.Key, got["key"])
		require.Equal(t, string(item.State), got["state"])
	}
	require.Equal(t, string(want.TraversalOutcome), payload["traversalOutcome"])
	require.Equal(t, want.TraversalOutcome == process.TraversalOutcomeComplete, payload["scopeComplete"])
	require.Equal(t, want.Warning, payload["warning"])
	requireDryRunPreviewMissingAncestors(t, payload, want.MissingAncestors)
	require.Equal(t, false, payload["mutationSubmitted"])
}

// TestCancelProcessInstanceDryRunPreviewPayloadMapping verifies cancel dry-run plans map to the public JSON payload.
func TestCancelProcessInstanceDryRunPreviewPayloadMapping(t *testing.T) {
	want := newCancelDryRunPreviewFixture()
	preview := newProcessInstanceDryRunPreview("cancel", want.RequestedKeys, process.DryRunPIKeyExpansion{
		Roots:            want.ResolvedRoots,
		Collected:        want.AffectedFamilyKeys,
		MissingAncestors: want.MissingAncestors,
		Warning:          want.Warning,
		Outcome:          want.TraversalOutcome,
	})

	var payload map[string]any
	b, err := json.Marshal(preview)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &payload))

	requireCancelDryRunPreviewPayload(t, payload, want)
}

// TestCancelProcessInstanceDryRun_DefaultOutputHidesScopeKeysUntilVerbose verifies dry-run output keeps key lists verbose-only.
func TestCancelProcessInstanceDryRun_DefaultOutputHidesScopeKeysUntilVerbose(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	preview := newProcessInstanceDryRunPreview("cancel", typex.Keys{"child-human"}, process.DryRunPIKeyExpansion{
		Roots:     typex.Keys{"root-human"},
		Collected: typex.Keys{"root-human", "child-human", "sibling-human"},
		Outcome:   process.TraversalOutcomeComplete,
	})

	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))

	output := buf.String()
	require.Contains(t, output, "dry run: cancel process-instance")
	require.Contains(t, output, "selected process instances: 1")
	require.Contains(t, output, "process-instance trees to cancel: 1")
	require.Contains(t, output, "process instances in scope: 3")
	require.Contains(t, output, "process-instance family scope: complete (all related process instances were found)")
	require.NotContains(t, output, "selected process-instance keys")
	require.NotContains(t, output, "root process-instance tree keys")
	require.NotContains(t, output, "in-scope process-instance keys")
	require.NotContains(t, output, "root-human")
	require.NotContains(t, output, "child-human")
	require.NotContains(t, output, "sibling-human")
	require.NotContains(t, output, "no mutation submitted")

	buf.Reset()
	flagVerbose = true
	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))
	output = buf.String()
	require.Contains(t, output, "selected process-instance keys: child-human")
	require.Contains(t, output, "root process-instance tree keys: root-human")
	require.Contains(t, output, "in-scope process-instance keys: root-human, child-human, sibling-human")
	require.NotContains(t, output, "no mutation submitted")
}

// TestCancelProcessInstanceDryRun_DefaultOutputSummarizesSelectedFinalStateInstances
// verifies terminal selected instances are summarized without noisy key lists by
// default.
func TestCancelProcessInstanceDryRun_DefaultOutputSummarizesSelectedFinalStateInstances(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	preview := newProcessInstanceDryRunPreview("cancel", typex.Keys{"done-1", "active-1"}, process.DryRunPIKeyExpansion{
		Roots:              typex.Keys{"root-human"},
		Collected:          typex.Keys{"root-human", "done-1", "active-1"},
		SelectedFinalState: []process.ProcessInstance{{Key: "done-1", State: process.StateCanceled}},
		Outcome:            process.TraversalOutcomeComplete,
	})

	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))
	require.Contains(t, buf.String(), "selected process instances already in final state: 1 (states: CANCELED; not affected by cancel; use --verbose to list keys)")
	require.NotContains(t, buf.String(), "done-1=CANCELED")
	require.NotContains(t, buf.String(), "in-scope process-instance keys")

	var payload map[string]any
	b, err := json.Marshal(preview)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &payload))
	require.Equal(t, float64(1), payload["selectedFinalStateCount"])
	selectedFinalState, ok := payload["selectedFinalState"].([]any)
	require.True(t, ok)
	require.Len(t, selectedFinalState, 1)
	selectedItem, ok := selectedFinalState[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "done-1", selectedItem["key"])
	require.Equal(t, string(process.StateCanceled), selectedItem["state"])

	buf.Reset()
	flagVerbose = true
	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))
	require.Contains(t, buf.String(), "selected process instances already in final state: 1 (states: CANCELED; not affected by cancel; done-1=CANCELED)")
}

// TestCancelProcessInstanceDryRun_StructuredOutputIncludesInspectableScope
// verifies JSON dry-run output carries scope metadata.
func TestCancelProcessInstanceDryRun_StructuredOutputIncludesInspectableScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagViewAsJson = true

	want := newCancelDryRunPreviewFixture()
	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	preview := newProcessInstanceDryRunPreview("cancel", want.RequestedKeys, process.DryRunPIKeyExpansion{
		Roots:            want.ResolvedRoots,
		Collected:        want.AffectedFamilyKeys,
		MissingAncestors: want.MissingAncestors,
		Warning:          want.Warning,
		Outcome:          want.TraversalOutcome,
	})

	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))

	payload := requireDryRunEnvelopePayload(t, buf.String())
	requireCancelDryRunPreviewPayload(t, payload, want)
}

// TestCancelProcessInstanceDryRun_SearchSummaryExplainsPartialScope verifies
// aggregate output preserves partial-scope warnings.
func TestCancelProcessInstanceDryRun_SearchSummaryExplainsPartialScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	summary := newProcessInstanceDryRunSummary("cancel", []processInstanceDryRunPreview{
		newProcessInstanceDryRunPreview("cancel", typex.Keys{"101", "102"}, process.DryRunPIKeyExpansion{
			Roots:     typex.Keys{"root-a"},
			Collected: typex.Keys{"root-a", "101", "102"},
			Outcome:   process.TraversalOutcomeComplete,
		}),
		newProcessInstanceDryRunPreview("cancel", typex.Keys{"103"}, process.DryRunPIKeyExpansion{
			Roots:              typex.Keys{"root-b"},
			Collected:          typex.Keys{"root-b", "103"},
			SelectedFinalState: []process.ProcessInstance{{Key: "103", State: process.StateTerminated}},
			MissingAncestors:   []process.MissingAncestor{{Key: "missing-parent", StartKey: "103"}},
			Warning:            "one or more parent process instances were not found",
			Outcome:            process.TraversalOutcomePartial,
		}),
		newProcessInstanceDryRunPreview("cancel", typex.Keys{"104"}, process.DryRunPIKeyExpansion{
			Roots:            typex.Keys{"root-c"},
			Collected:        typex.Keys{"root-c", "104"},
			MissingAncestors: []process.MissingAncestor{{Key: "missing-parent", StartKey: "103"}},
			Warning:          "one or more parent process instances were not found",
			Outcome:          process.TraversalOutcomePartial,
		}),
	})

	require.False(t, summary.ScopeComplete)
	require.Equal(t, process.TraversalOutcomePartial, summary.TraversalOutcome)
	require.Equal(t, []processInstanceDryRunMissingAncestor{{Key: "missing-parent", StartKey: "103"}}, summary.MissingAncestors)
	require.Equal(t, []processInstanceDryRunSelectedFinalState{{Key: "103", State: process.StateTerminated}}, summary.SelectedFinalState)
	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, summary))

	output := buf.String()
	require.Contains(t, output, "dry run: cancel process-instance")
	require.Contains(t, output, "selected process instances: 4")
	require.Contains(t, output, "process-instance trees to cancel: 3")
	require.Contains(t, output, "process instances in scope: 7")
	require.Contains(t, output, "selected process instances already in final state: 1 (states: TERMINATED; not affected by cancel; use --verbose to list keys)")
	require.Contains(t, output, "scope: partial (one or more parent process instances were not found; missing ancestor keys: 1; use --verbose to list keys)")
	require.NotContains(t, output, "warning:")
	require.NotContains(t, output, "missing ancestor keys: missing-parent")
	require.NotContains(t, output, "103=TERMINATED")
	require.NotContains(t, output, "search pages processed")
	require.NotContains(t, output, "no mutation submitted")

	buf.Reset()
	flagVerbose = true
	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, summary))
	require.Contains(t, buf.String(), "selected process instances already in final state: 1 (states: TERMINATED; not affected by cancel; 103=TERMINATED)")
	require.Contains(t, buf.String(), "scope: partial (one or more parent process instances were not found; missing ancestor keys: missing-parent)")
}

// TestDeleteProcessInstanceDryRunPreviewPayloadMapping verifies delete dry-run plans map to the public JSON payload.
func TestDeleteProcessInstanceDryRunPreviewPayloadMapping(t *testing.T) {
	want := newDeleteDryRunPreviewFixture()
	preview := newProcessInstanceDryRunPreview("delete", want.RequestedKeys, process.DryRunPIKeyExpansion{
		Roots:                      want.ResolvedRoots,
		Collected:                  want.AffectedFamilyKeys,
		RequiresCancelBeforeDelete: want.RequiresCancel,
		MissingAncestors:           want.MissingAncestors,
		Warning:                    want.Warning,
		Outcome:                    want.TraversalOutcome,
	})

	var payload map[string]any
	b, err := json.Marshal(preview)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &payload))

	requireDeleteDryRunPreviewPayload(t, payload, want)
}

// TestDeleteProcessInstanceDryRun_DefaultOutputHidesScopeKeysUntilVerbose verifies dry-run output keeps key lists verbose-only.
func TestDeleteProcessInstanceDryRun_DefaultOutputHidesScopeKeysUntilVerbose(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	preview := newProcessInstanceDryRunPreview("delete", typex.Keys{"child-human"}, process.DryRunPIKeyExpansion{
		Roots:     typex.Keys{"root-human"},
		Collected: typex.Keys{"root-human", "child-human", "sibling-human"},
		Outcome:   process.TraversalOutcomeComplete,
	})

	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))

	output := buf.String()
	require.Contains(t, output, "dry run: delete process-instance")
	require.Contains(t, output, "selected process instances: 1")
	require.Contains(t, output, "process-instance trees to delete: 1")
	require.Contains(t, output, "process instances in scope: 3")
	require.Contains(t, output, "process-instance family scope: complete (all related process instances were found)")
	require.NotContains(t, output, "selected process-instance keys")
	require.NotContains(t, output, "root process-instance tree keys")
	require.NotContains(t, output, "in-scope process-instance keys")
	require.NotContains(t, output, "root-human")
	require.NotContains(t, output, "child-human")
	require.NotContains(t, output, "sibling-human")
	require.NotContains(t, output, "no mutation submitted")

	buf.Reset()
	flagVerbose = true
	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))
	output = buf.String()
	require.Contains(t, output, "selected process-instance keys: child-human")
	require.Contains(t, output, "root process-instance tree keys: root-human")
	require.Contains(t, output, "in-scope process-instance keys: root-human, child-human, sibling-human")
	require.NotContains(t, output, "no mutation submitted")
}

// TestDeleteProcessInstanceDryRun_DefaultOutputSummarizesSelectedFinalStateInstances
// verifies terminal selected instances are summarized without noisy key lists by
// default.
func TestDeleteProcessInstanceDryRun_DefaultOutputSummarizesSelectedFinalStateInstances(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	preview := newProcessInstanceDryRunPreview("delete", typex.Keys{"done-1", "active-1"}, process.DryRunPIKeyExpansion{
		Roots:                      typex.Keys{"root-human"},
		Collected:                  typex.Keys{"root-human", "done-1", "active-1"},
		SelectedFinalState:         []process.ProcessInstance{{Key: "done-1", State: process.StateTerminated}},
		RequiresCancelBeforeDelete: []process.ProcessInstance{{Key: "active-1", State: process.StateActive}},
		Outcome:                    process.TraversalOutcomeComplete,
	})

	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))
	require.Contains(t, buf.String(), "selected process instances already in final state: 1 (states: TERMINATED; use --verbose to list keys)")
	require.Contains(t, buf.String(), "process instances not in final state: 1 (states: ACTIVE; delete cannot remove them directly; use --force to cancel before delete; use --verbose to list keys)")
	require.NotContains(t, buf.String(), "done-1=TERMINATED")
	require.NotContains(t, buf.String(), "active-1=ACTIVE")
	require.NotContains(t, buf.String(), "in-scope process-instance keys")

	buf.Reset()
	flagVerbose = true
	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))
	require.Contains(t, buf.String(), "selected process instances already in final state: 1 (states: TERMINATED; done-1=TERMINATED)")
	require.Contains(t, buf.String(), "process instances not in final state: 1 (states: ACTIVE; delete cannot remove them directly; use --force to cancel before delete; active-1=ACTIVE)")

	buf.Reset()
	flagVerbose = false
	flagForce = true
	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))
	require.Contains(t, buf.String(), "process instances not in final state: 1 (states: ACTIVE; --force would cancel them before delete; use --verbose to list keys)")
}

// TestDeleteProcessInstanceDryRun_StructuredOutputIncludesInspectableScope
// verifies JSON dry-run output carries scope metadata.
func TestDeleteProcessInstanceDryRun_StructuredOutputIncludesInspectableScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagViewAsJson = true

	want := newDeleteDryRunPreviewFixture()
	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	preview := newProcessInstanceDryRunPreview("delete", want.RequestedKeys, process.DryRunPIKeyExpansion{
		Roots:                      want.ResolvedRoots,
		Collected:                  want.AffectedFamilyKeys,
		RequiresCancelBeforeDelete: want.RequiresCancel,
		MissingAncestors:           want.MissingAncestors,
		Warning:                    want.Warning,
		Outcome:                    want.TraversalOutcome,
	})

	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))

	payload := requireDryRunEnvelopePayload(t, buf.String())
	requireDeleteDryRunPreviewPayload(t, payload, want)
}

// TestDeleteProcessInstanceDryRun_SearchSummaryExplainsNonFinalScope verifies
// aggregate output explains non-final delete blockers.
func TestDeleteProcessInstanceDryRun_SearchSummaryExplainsNonFinalScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	summary := newProcessInstanceDryRunSummary("delete", []processInstanceDryRunPreview{
		newProcessInstanceDryRunPreview("delete", typex.Keys{"401", "402"}, process.DryRunPIKeyExpansion{
			Roots:                      typex.Keys{"root-a"},
			Collected:                  typex.Keys{"root-a", "401", "402"},
			SelectedFinalState:         []process.ProcessInstance{{Key: "402", State: process.StateTerminated}},
			RequiresCancelBeforeDelete: []process.ProcessInstance{{Key: "root-a", State: process.StateActive}, {Key: "401", State: process.StateActive}},
			Outcome:                    process.TraversalOutcomeComplete,
		}),
		newProcessInstanceDryRunPreview("delete", typex.Keys{"403"}, process.DryRunPIKeyExpansion{
			Roots:                      typex.Keys{"root-b"},
			Collected:                  typex.Keys{"root-b", "403"},
			RequiresCancelBeforeDelete: []process.ProcessInstance{{Key: "root-b", State: process.StateActive}},
			MissingAncestors:           []process.MissingAncestor{{Key: "missing-parent", StartKey: "403"}},
			Warning:                    "one or more parent process instances were not found",
			Outcome:                    process.TraversalOutcomePartial,
		}),
	})

	require.False(t, summary.ScopeComplete)
	require.Equal(t, process.TraversalOutcomePartial, summary.TraversalOutcome)
	require.Equal(t, 3, summary.RequiresCancelBeforeDeleteCount)
	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, summary))

	output := buf.String()
	require.Contains(t, output, "dry run: delete process-instance")
	require.Contains(t, output, "selected process instances: 3")
	require.Contains(t, output, "process-instance trees to delete: 2")
	require.Contains(t, output, "process instances in scope: 5")
	require.Contains(t, output, "selected process instances already in final state: 1 (states: TERMINATED; use --verbose to list keys)")
	require.Contains(t, output, "process instances not in final state: 3 (states: ACTIVE; delete cannot remove them directly; use --force to cancel before delete; use --verbose to list keys)")
	require.Contains(t, output, "scope: partial (one or more parent process instances were not found; missing ancestor keys: 1; use --verbose to list keys)")
	require.NotContains(t, output, "root-a=ACTIVE")
	require.NotContains(t, output, "search pages processed")
	require.NotContains(t, output, "no mutation submitted")

	buf.Reset()
	flagForce = true
	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, summary))
	require.Contains(t, buf.String(), "process instances not in final state: 3 (states: ACTIVE; --force would cancel them before delete; use --verbose to list keys)")
}
