// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
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

func TestProcessDefinitionSelectorHumanDiagnostic_SingleMissingSelectorOffersListing(t *testing.T) {
	resetProcessDefinitionSelectorPromptTestState(t)
	processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }

	cmd, _ := newProcessDefinitionSelectorValidationTestCommand()
	var prompt string
	confirmCmdOrAbortFn = func(autoConfirm bool, got string) error {
		require.False(t, autoConfirm)
		prompt = got
		return localPreconditionError(ErrCmdAborted)
	}

	err := processDefinitionSelectorValidationError(cmd, stubProcessAPI{}, processDefinitionSelectorValidationResult{
		MissingBpmnProcessIDs: []string{"missing-process"},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, ferrors.ErrLocalPrecondition))
	require.Contains(t, err.Error(), "no visible process definition matches the provided selector")
	require.Contains(t, err.Error(), "bpmnProcessId: missing-process")
	require.Contains(t, prompt, "no visible process definition matches the provided selector")
	require.Contains(t, prompt, "bpmnProcessId: missing-process")
	require.Contains(t, prompt, "List visible process definitions?")
}

func TestProcessDefinitionSelectorHumanDiagnostic_MultipleMissingSelectorsOffersListing(t *testing.T) {
	resetProcessDefinitionSelectorPromptTestState(t)
	processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }

	cmd, _ := newProcessDefinitionSelectorValidationTestCommand()
	var prompt string
	confirmCmdOrAbortFn = func(autoConfirm bool, got string) error {
		require.False(t, autoConfirm)
		prompt = got
		return localPreconditionError(ErrCmdAborted)
	}

	err := processDefinitionSelectorValidationError(cmd, stubProcessAPI{}, processDefinitionSelectorValidationResult{
		MissingBpmnProcessIDs: []string{"missing-a", "missing-b"},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "no visible process definitions match the provided selector(s)")
	require.Contains(t, err.Error(), "bpmnProcessId: missing-a")
	require.Contains(t, err.Error(), "bpmnProcessId: missing-b")
	require.Contains(t, err.Error(), "They may not exist")
	require.Contains(t, prompt, "no visible process definitions match the provided selector(s)")
	require.Contains(t, prompt, "bpmnProcessId: missing-a")
	require.Contains(t, prompt, "bpmnProcessId: missing-b")
	require.Contains(t, prompt, "List visible process definitions?")
}

func TestProcessDefinitionSelectorValidationError_MachineAndNonTTYModesDoNotPrompt(t *testing.T) {
	tests := []struct {
		name  string
		setup func(cmd *cobra.Command)
	}{
		{
			name: "json",
			setup: func(cmd *cobra.Command) {
				flagViewAsJson = true
			},
		},
		{
			name: "automation",
			setup: func(cmd *cobra.Command) {
				require.NoError(t, cmd.Flags().Set("automation", "true"))
			},
		},
		{
			name: "keys-only",
			setup: func(cmd *cobra.Command) {
				flagViewKeysOnly = true
			},
		},
		{
			name: "non-tty",
			setup: func(cmd *cobra.Command) {
				processDefinitionSelectorInteractiveTerminalFn = func() bool { return false }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessDefinitionSelectorPromptTestState(t)
			processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }
			cmd, _ := newProcessDefinitionSelectorValidationTestCommand()
			tt.setup(cmd)
			confirmCmdOrAbortFn = func(bool, string) error {
				t.Fatal("unexpected process-definition selector listing prompt")
				return nil
			}

			err := processDefinitionSelectorValidationError(cmd, stubProcessAPI{}, processDefinitionSelectorValidationResult{
				MissingBpmnProcessIDs: []string{"missing-process"},
			})

			require.Error(t, err)
			require.Contains(t, err.Error(), "no visible process definition matches the provided selector")
			require.NotContains(t, err.Error(), "List visible process definitions?")
		})
	}
}

func TestProcessDefinitionSelectorValidationError_AcceptedPromptListsVisibleDefinitions(t *testing.T) {
	resetProcessDefinitionSelectorPromptTestState(t)
	processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }

	cmd, output := newProcessDefinitionSelectorValidationTestCommand()
	var prompt string
	confirmCmdOrAbortFn = func(autoConfirm bool, got string) error {
		require.False(t, autoConfirm)
		prompt = got
		return nil
	}
	cli := stubProcessAPI{
		searchProcessDefinitions: func(_ context.Context, filter process.ProcessDefinitionFilter, opts ...options.FacadeOption) (process.ProcessDefinitions, error) {
			require.Equal(t, process.ProcessDefinitionFilter{}, filter)
			return process.ProcessDefinitions{
				Total: 2,
				Items: []process.ProcessDefinition{
					{Key: "2251799813685250", TenantId: "tenant-a", BpmnProcessId: "invoice", ProcessVersion: 2},
					{Key: "2251799813685251", TenantId: "tenant-b", BpmnProcessId: "order", ProcessVersion: 7, ProcessVersionTag: "stable"},
				},
			}, nil
		},
	}

	err := processDefinitionSelectorValidationError(cmd, cli, processDefinitionSelectorValidationResult{
		MissingBpmnProcessIDs: []string{"missing-process"},
	})

	require.Error(t, err)
	require.Contains(t, prompt, "List visible process definitions?")
	require.Contains(t, output.String(), "2251799813685250")
	require.Contains(t, output.String(), "tenant-a")
	require.Contains(t, output.String(), "invoice")
	require.Contains(t, output.String(), "v2")
	require.Contains(t, output.String(), "2251799813685251")
	require.Contains(t, output.String(), "order")
	require.Contains(t, output.String(), "v7/stable")
	require.Contains(t, output.String(), "found: 2")
}

func newProcessDefinitionSelectorValidationTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.Flags().Bool("automation", false, "")
	return cmd, buf
}

func resetProcessDefinitionSelectorPromptTestState(t *testing.T) {
	t.Helper()

	resetProcessInstanceCommandGlobals()
	processDefinitionSelectorInteractiveTerminalFn = processDefinitionSelectorInteractiveTerminal
	t.Cleanup(func() {
		resetProcessInstanceCommandGlobals()
		processDefinitionSelectorInteractiveTerminalFn = processDefinitionSelectorInteractiveTerminal
	})
}
