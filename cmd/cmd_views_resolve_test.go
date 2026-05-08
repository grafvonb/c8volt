// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
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
	results := process.IncidentResolutionResults{
		Items: []process.IncidentResolutionResult{
			{IncidentKey: "2251799813685249", Status: process.IncidentResolutionStatusConfirmed},
			{IncidentKey: "2251799813685250", Status: process.IncidentResolutionStatusSubmitted},
		},
	}

	require.NoError(t, renderIncidentResolutionResults(cmd, results))

	output := buf.String()
	require.Contains(t, output, "resolved incident 2251799813685249: confirmed")
	require.Contains(t, output, "resolved incident 2251799813685250: submitted")
	require.Contains(t, output, "resolved: 2 (confirmed/submitted/skipped: 2, failed: 0)")
}

func TestRenderIncidentResolutionResults_JSONUsesSharedEnvelope(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() { flagViewAsJson = prevJSON })

	cmd := &cobra.Command{Use: "incident"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := process.IncidentResolutionResults{
		Items: []process.IncidentResolutionResult{{
			IncidentKey:        "2251799813685249",
			MutationAccepted:   true,
			Status:             process.IncidentResolutionStatusConfirmed,
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
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813685249", item["incidentKey"])
	require.Equal(t, "2251799813685250", item["processInstanceKey"])
	require.Equal(t, "confirmed", item["status"])
	require.Equal(t, true, item["mutationAccepted"])
	require.Equal(t, "resolved", item["confirmationStatus"])
	require.Equal(t, true, item["mutationSubmitted"])
}

func TestRenderProcessInstanceResolutionResults_HumanOutputShowsNoOpSuccessAndFailure(t *testing.T) {
	prevJSON := flagViewAsJson
	flagViewAsJson = false
	t.Cleanup(func() { flagViewAsJson = prevJSON })

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	results := process.ProcessInstanceResolutionResults{
		Items: []process.ProcessInstanceResolutionResult{
			{
				ProcessInstanceKey:    "2251799813685250",
				Status:                process.ProcessInstanceResolutionStatusConfirmed,
				ResolvedIncidentKeys:  []string{"2251799813685249"},
				ConfirmationStatus:    "resolved",
				MutationSubmitted:     true,
				AttemptedIncidentKeys: []string{"2251799813685249"},
			},
			{
				ProcessInstanceKey: "2251799813685260",
				Status:             process.ProcessInstanceResolutionStatusSkipped,
				ConfirmationStatus: "no_active_incidents",
			},
			{
				ProcessInstanceKey:   "2251799813685270",
				Status:               process.ProcessInstanceResolutionStatusPartialFailed,
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
	results := process.ProcessInstanceResolutionResults{
		Items: []process.ProcessInstanceResolutionResult{{
			ProcessInstanceKey:    "2251799813685250",
			AttemptedIncidentKeys: []string{"2251799813685249"},
			ResolvedIncidentKeys:  []string{"2251799813685249"},
			FailedIncidentKeys:    []string{},
			Status:                process.ProcessInstanceResolutionStatusConfirmed,
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
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813685250", item["processInstanceKey"])
	require.Equal(t, "confirmed", item["status"])
	require.Equal(t, "resolved", item["confirmationStatus"])
	require.Equal(t, true, item["mutationSubmitted"])
	require.Equal(t, []any{"2251799813685249"}, item["attemptedIncidentKeys"])
	require.Equal(t, []any{"2251799813685249"}, item["resolvedIncidentKeys"])
}
