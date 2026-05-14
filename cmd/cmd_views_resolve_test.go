// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRenderIncidentResolutionResults_HumanOutputShowsPerTargetStatuses(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() { flagViewAsJson = prevJSON })

	cmd := &cobra.Command{Use: "incident"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := incident.ResolutionResults{
		Items: []incident.ResolutionResult{
			{IncidentKey: "2251799813685249", Status: incident.ResolutionStatusConfirmed},
			{IncidentKey: "2251799813685250", Status: incident.ResolutionStatusSubmitted},
			{IncidentKey: "2251799813685251", Status: incident.ResolutionStatusSkipped, IncidentState: "RESOLVED", Incident: &incident.ProcessInstanceIncidentDetail{CreationTime: "2026-05-06T15:43:59.260Z"}},
		},
	}

	require.NoError(t, renderIncidentResolutionResults(cmd, results))

	output := buf.String()
	require.Contains(t, output, "resolved incident 2251799813685249: confirmed")
	require.Contains(t, output, "resolved incident 2251799813685250: submitted")
	require.Contains(t, output, "incident 2251799813685251 already resolved (created 2026-05-06T15:43:59.260): skipped")
	require.Contains(t, output, "resolved: 3 (confirmed/submitted/skipped: 3, failed: 0)")
}

func TestRenderIncidentResolutionResults_JSONUsesSharedEnvelope(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() { flagViewAsJson = prevJSON })

	cmd := &cobra.Command{Use: "incident"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := incident.ResolutionResults{
		Operation: incident.ResolutionOperationIncident,
		Items: []incident.ResolutionResult{{
			IncidentKey:        "2251799813685249",
			MutationAccepted:   true,
			Status:             incident.ResolutionStatusConfirmed,
			ConfirmationStatus: "resolved",
			StatusCode:         204,
			MutationSubmitted:  true,
			ProcessInstanceKey: "2251799813685250",
		}},
		Total:             1,
		Confirmed:         1,
		MutationSubmitted: true,
	}

	require.NoError(t, renderIncidentResolutionResults(cmd, results))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "resolveIncident", payload["operation"])
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813685249", item["incidentKey"])
	require.Equal(t, "2251799813685250", item["processInstanceKey"])
	require.Equal(t, "confirmed", item["status"])
	require.Equal(t, true, item["mutationAccepted"])
	require.Equal(t, "resolved", item["confirmationStatus"])
	require.Equal(t, true, item["mutationSubmitted"])
}

func TestRenderIncidentResolutionResults_DryRunHumanOutputIsCompact(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() { flagViewAsJson = prevJSON })

	cmd := &cobra.Command{Use: "incident"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := incident.ResolutionResults{
		Operation: incident.ResolutionOperationIncident,
		Items: []incident.ResolutionResult{{
			IncidentKey:       "2251799813685249",
			Status:            incident.ResolutionStatusPlanned,
			DryRun:            true,
			MutationSubmitted: false,
			WouldResolve:      true,
		}},
		Total:   1,
		Skipped: 1,
		DryRun:  true,
	}

	require.NoError(t, renderIncidentResolutionResults(cmd, results))

	output := buf.String()
	require.Contains(t, output, "dry run: incident 2251799813685249 would be resolved")
	require.Contains(t, output, "dry run: resolve incidents: 1 target(s), 1 planned/skipped, 0 failed; no changes applied")
	require.NotContains(t, output, "resolved:")
}

func TestRenderIncidentResolutionResults_JSONDryRunPayloadIgnoresVerbose(t *testing.T) {
	prevJSON := flagViewAsJson
	prevVerbose := flagVerbose
	flagViewAsJson = true
	flagVerbose = false
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
		flagVerbose = prevVerbose
	})

	cmd := &cobra.Command{Use: "incident"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := incident.ResolutionResults{
		Operation: incident.ResolutionOperationIncident,
		Items: []incident.ResolutionResult{{
			IncidentKey:       "2251799813685249",
			Status:            incident.ResolutionStatusPlanned,
			DryRun:            true,
			MutationSubmitted: false,
			WouldResolve:      true,
		}},
		Total:   1,
		Skipped: 1,
		DryRun:  true,
	}

	require.NoError(t, renderIncidentResolutionResults(cmd, results))
	defaultOutput := buf.String()
	payload := requireDryRunEnvelopePayload(t, defaultOutput)
	require.Equal(t, "resolveIncident", payload["operation"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, false, payload["mutationSubmitted"])

	buf.Reset()
	flagVerbose = true
	require.NoError(t, renderIncidentResolutionResults(cmd, results))
	require.JSONEq(t, defaultOutput, buf.String())
}

func TestRenderProcessInstanceResolutionResults_HumanOutputShowsNoOpSuccessAndFailure(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() { flagViewAsJson = prevJSON })

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := incident.ProcessInstanceResolutionResults{
		Items: []incident.ProcessInstanceResolutionResult{
			{
				ProcessInstanceKey:    "2251799813685250",
				Status:                incident.ProcessInstanceResolutionStatusConfirmed,
				ResolvedIncidentKeys:  []string{"2251799813685249"},
				ConfirmationStatus:    "resolved",
				MutationSubmitted:     true,
				AttemptedIncidentKeys: []string{"2251799813685249"},
			},
			{
				ProcessInstanceKey: "2251799813685260",
				Status:             incident.ProcessInstanceResolutionStatusSkipped,
				ConfirmationStatus: "no_active_incidents",
			},
			{
				ProcessInstanceKey:   "2251799813685270",
				Status:               incident.ProcessInstanceResolutionStatusPartialFailed,
				ResolvedIncidentKeys: []string{"2251799813685271"},
				FailedIncidentKeys:   []string{"2251799813685272"},
				Error:                "mutation rejected",
			},
		},
	}

	require.Error(t, renderProcessInstanceResolutionResults(cmd, results))

	output := buf.String()
	require.Contains(t, output, "resolved process-instance 2251799813685250: confirmed (1 incident(s))")
	require.Contains(t, output, "resolved process-instance 2251799813685260: skipped (no_active_incidents)")
	require.Contains(t, output, "resolved process-instance 2251799813685270: partial failure (resolved: 1, failed: 1): mutation rejected")
	require.Contains(t, output, "resolved process-instances: 3 (confirmed/submitted/skipped: 2, failed: 1)")
}

func TestRenderProcessInstanceResolutionResults_JSONUsesSharedEnvelope(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() { flagViewAsJson = prevJSON })

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := incident.ProcessInstanceResolutionResults{
		Operation: incident.ResolutionOperationProcessInstance,
		Items: []incident.ProcessInstanceResolutionResult{{
			ProcessInstanceKey:    "2251799813685250",
			AttemptedIncidentKeys: []string{"2251799813685249"},
			ResolvedIncidentKeys:  []string{"2251799813685249"},
			FailedIncidentKeys:    []string{},
			Status:                incident.ProcessInstanceResolutionStatusConfirmed,
			ConfirmationStatus:    "resolved",
			MutationSubmitted:     true,
		}},
		Total:             1,
		Confirmed:         1,
		MutationSubmitted: true,
	}

	require.NoError(t, renderProcessInstanceResolutionResults(cmd, results))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "resolveProcessInstance", payload["operation"])
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813685250", item["processInstanceKey"])
	require.Equal(t, "confirmed", item["status"])
	require.Equal(t, "resolved", item["confirmationStatus"])
	require.Equal(t, true, item["mutationSubmitted"])
	require.Equal(t, []any{"2251799813685249"}, item["attemptedIncidentKeys"])
	require.Equal(t, []any{"2251799813685249"}, item["resolvedIncidentKeys"])
}

func TestRenderProcessInstanceResolutionResults_DryRunHumanOutputIsCompact(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() { flagViewAsJson = prevJSON })

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := incident.ProcessInstanceResolutionResults{
		Operation: incident.ResolutionOperationProcessInstance,
		Items: []incident.ProcessInstanceResolutionResult{{
			ProcessInstanceKey:    "2251799813685250",
			AttemptedIncidentKeys: []string{"2251799813685249", "2251799813685251"},
			Status:                incident.ProcessInstanceResolutionStatusPlanned,
			DryRun:                true,
			MutationSubmitted:     false,
		}},
		Total:   1,
		Skipped: 1,
		DryRun:  true,
	}

	require.NoError(t, renderProcessInstanceResolutionResults(cmd, results))

	output := buf.String()
	require.Contains(t, output, "dry run: process-instance 2251799813685250 would resolve 2 incident(s)")
	require.Contains(t, output, "dry run: resolve process-instances: 1 target(s), 1 planned/skipped, 0 failed; no changes applied")
}

func TestRenderProcessInstanceResolutionResults_JSONDryRunPayloadIgnoresVerbose(t *testing.T) {
	prevJSON := flagViewAsJson
	prevVerbose := flagVerbose
	flagViewAsJson = true
	flagVerbose = false
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
		flagVerbose = prevVerbose
	})

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := incident.ProcessInstanceResolutionResults{
		Operation: incident.ResolutionOperationProcessInstance,
		Items: []incident.ProcessInstanceResolutionResult{{
			ProcessInstanceKey:    "2251799813685250",
			AttemptedIncidentKeys: []string{"2251799813685249"},
			Status:                incident.ProcessInstanceResolutionStatusPlanned,
			DryRun:                true,
			MutationSubmitted:     false,
		}},
		Total:   1,
		Skipped: 1,
		DryRun:  true,
	}

	require.NoError(t, renderProcessInstanceResolutionResults(cmd, results))
	defaultOutput := buf.String()
	payload := requireDryRunEnvelopePayload(t, defaultOutput)
	require.Equal(t, "resolveProcessInstance", payload["operation"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, false, payload["mutationSubmitted"])

	buf.Reset()
	flagVerbose = true
	require.NoError(t, renderProcessInstanceResolutionResults(cmd, results))
	require.JSONEq(t, defaultOutput, buf.String())
}
