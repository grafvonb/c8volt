// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

func TestProcessDefinitionSelectorFromPIFlags_MapsBpmnVersionAndTag(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIBpmnProcessID = "order-process"
	flagGetPIProcessVersion = 7
	flagGetPIProcessVersionTag = "stable"

	req := newPIProcessDefinitionSelectorValidationRequest()
	filter := req.filterForBpmnProcessID("order-process")

	require.Equal(t, processDefinitionSelectorValidationRequest{
		BpmnProcessIds:    []string{"order-process"},
		ProcessVersion:    7,
		ProcessVersionTag: "stable",
	}, req)
	require.Equal(t, process.ProcessDefinitionFilter{
		BpmnProcessId:     "order-process",
		ProcessVersion:    7,
		ProcessVersionTag: "stable",
	}, filter)
}

func TestProcessDefinitionSelectorValidation_SearchesEveryDistinctBpmnProcessID(t *testing.T) {
	var gotFilters []process.ProcessDefinitionFilter
	cli := stubProcessAPI{
		searchProcessDefinitions: func(_ context.Context, filter process.ProcessDefinitionFilter, opts ...options.FacadeOption) (process.ProcessDefinitions, error) {
			gotFilters = append(gotFilters, filter)
			return process.ProcessDefinitions{
				Items: []process.ProcessDefinition{{Key: "pd-" + filter.BpmnProcessId, BpmnProcessId: filter.BpmnProcessId}},
			}, nil
		},
	}

	result, err := validateProcessDefinitionSelectors(context.Background(), cli, processDefinitionSelectorValidationRequest{
		BpmnProcessIds:    []string{"order", "order", " invoice "},
		ProcessVersion:    3,
		ProcessVersionTag: "release",
	})

	require.NoError(t, err)
	require.True(t, result.Valid())
	require.Empty(t, result.MissingBpmnProcessIDs)
	require.Equal(t, []process.ProcessDefinitionFilter{
		{BpmnProcessId: "order", ProcessVersion: 3, ProcessVersionTag: "release"},
		{BpmnProcessId: "invoice", ProcessVersion: 3, ProcessVersionTag: "release"},
	}, gotFilters)
	require.Contains(t, result.MatchesByBpmnProcessID, "order")
	require.Contains(t, result.MatchesByBpmnProcessID, "invoice")
}

func TestProcessDefinitionSelectorValidation_UsesLatestSearchWhenRequested(t *testing.T) {
	var latestCalls int
	cli := stubProcessAPI{
		searchProcessDefinitionsLatest: func(_ context.Context, filter process.ProcessDefinitionFilter, opts ...options.FacadeOption) (process.ProcessDefinitions, error) {
			latestCalls++
			require.Equal(t, process.ProcessDefinitionFilter{BpmnProcessId: "order"}, filter)
			return process.ProcessDefinitions{Total: 1}, nil
		},
	}

	result, err := validateProcessDefinitionSelectors(context.Background(), cli, processDefinitionSelectorValidationRequest{
		BpmnProcessIds: []string{"order"},
		Mode:           processDefinitionSelectorValidationLatest,
	})

	require.NoError(t, err)
	require.True(t, result.Valid())
	require.Equal(t, 1, latestCalls)
}

func TestProcessDefinitionSelectorValidation_CollectsMissingSelectors(t *testing.T) {
	cli := stubProcessAPI{
		searchProcessDefinitions: func(_ context.Context, filter process.ProcessDefinitionFilter, opts ...options.FacadeOption) (process.ProcessDefinitions, error) {
			if filter.BpmnProcessId == "visible" {
				return process.ProcessDefinitions{Total: 1}, nil
			}
			return process.ProcessDefinitions{}, nil
		},
	}

	result, err := validateProcessDefinitionSelectors(context.Background(), cli, processDefinitionSelectorValidationRequest{
		BpmnProcessIds:    []string{"missing-a", "visible", "missing-b"},
		ProcessVersionTag: "blue",
	})

	require.NoError(t, err)
	require.False(t, result.Valid())
	require.Equal(t, []string{"missing-a", "missing-b"}, result.MissingBpmnProcessIDs)
	require.Contains(t, result.MatchesByBpmnProcessID, "visible")
}

func TestProcessDefinitionSelectorMissingFormatting_IncludesSelectorContextWithoutPrompt(t *testing.T) {
	result := processDefinitionSelectorValidationResult{
		Request: processDefinitionSelectorValidationRequest{
			ProcessVersion:    5,
			ProcessVersionTag: "release",
		},
		MissingBpmnProcessIDs: []string{"missing-a", "missing-b"},
	}

	got := formatMissingProcessDefinitionSelectors(result)
	err := processDefinitionSelectorNoPromptError(result)

	require.Contains(t, got, "no visible process definitions match the provided selector(s):")
	require.Contains(t, got, "bpmnProcessId: missing-a")
	require.Contains(t, got, "bpmnProcessId: missing-b")
	require.Contains(t, got, "processVersion: 5")
	require.Contains(t, got, "processVersionTag: release")
	require.Contains(t, got, "credentials may not have access")
	require.NotContains(t, got, "List visible process definitions")
	require.Error(t, err)
	require.True(t, errors.Is(err, ferrors.ErrLocalPrecondition))
}

func TestProcessDefinitionSelectorNoPromptError_ReturnsNilForValidResult(t *testing.T) {
	require.NoError(t, processDefinitionSelectorNoPromptError(processDefinitionSelectorValidationResult{}))
}
