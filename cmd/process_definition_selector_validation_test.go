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

// This protects the preflight contract that a mixed visible/missing selector reports all misses without hiding valid matches.
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

// Version/tag misses get a second broad lookup so interactive recovery can show nearby definitions.
func TestProcessDefinitionSelectorValidation_SearchesByIDOnlyWhenVersionSelectorMisses(t *testing.T) {
	var gotFilters []process.ProcessDefinitionFilter
	cli := stubProcessAPI{
		searchProcessDefinitions: func(_ context.Context, filter process.ProcessDefinitionFilter, opts ...options.FacadeOption) (process.ProcessDefinitions, error) {
			gotFilters = append(gotFilters, filter)
			if filter.BpmnProcessId == "order" && filter.ProcessVersion == 0 && filter.ProcessVersionTag == "" {
				require.True(t, options.ApplyFacadeOptions(opts).IgnoreTenant)
				return process.ProcessDefinitions{
					Total: 1,
					Items: []process.ProcessDefinition{
						{Key: "pd-order-v2", TenantId: "<default>", BpmnProcessId: "order", ProcessVersion: 2, ProcessVersionTag: "stable"},
					},
				}, nil
			}
			require.False(t, options.ApplyFacadeOptions(opts).IgnoreTenant)
			return process.ProcessDefinitions{}, nil
		},
	}

	result, err := validateProcessDefinitionSelectors(context.Background(), cli, processDefinitionSelectorValidationRequest{
		BpmnProcessIds:    []string{"order"},
		ProcessVersion:    99,
		ProcessVersionTag: "missing",
	})

	require.NoError(t, err)
	require.False(t, result.Valid())
	require.Equal(t, []string{"order"}, result.MissingBpmnProcessIDs)
	require.Equal(t, []process.ProcessDefinitionFilter{
		{BpmnProcessId: "order", ProcessVersion: 99, ProcessVersionTag: "missing"},
		{BpmnProcessId: "order"},
	}, gotFilters)
	require.True(t, result.HasNearMatches())
	require.Equal(t, "pd-order-v2", result.NearMatchesByBpmnProcessID["order"].Items[0].Key)
}

// Machine diagnostics intentionally stay compact and single-line for stderr and JSON envelopes.
func TestProcessDefinitionSelectorMissingFormatting_UsesSingleLineBracketedSelectors(t *testing.T) {
	result := processDefinitionSelectorValidationResult{
		Request: processDefinitionSelectorValidationRequest{
			ProcessVersion:    5,
			ProcessVersionTag: "release",
		},
		MissingBpmnProcessIDs: []string{"missing-a", "missing-b"},
	}

	got := formatMissingProcessDefinitionSelectors(result)
	err := processDefinitionSelectorNoPromptError(result)

	require.Equal(t, "no visible process definitions match the provided selector(s): [missing-a v5/release], [missing-b v5/release]", got)
	require.NotContains(t, got, "bpmnProcessId:")
	require.NotContains(t, got, "processVersion:")
	require.NotContains(t, got, "processVersionTag:")
	require.NotContains(t, got, "credentials may not have access")
	require.NotContains(t, got, "List visible process definitions")
	require.Error(t, err)
	require.Equal(t, got, err.Error())
	require.True(t, errors.Is(err, ferrors.ErrLocalPrecondition))
}

func TestProcessDefinitionSelectorMissingFormatting_UsesCompactVersionSelectors(t *testing.T) {
	tests := []struct {
		name string
		req  processDefinitionSelectorValidationRequest
		want string
	}{
		{
			name: "version only",
			req:  processDefinitionSelectorValidationRequest{ProcessVersion: 3},
			want: "missing-process v3",
		},
		{
			name: "tag only",
			req:  processDefinitionSelectorValidationRequest{ProcessVersionTag: "stable"},
			want: "missing-process */stable",
		},
		{
			name: "no version selector",
			req:  processDefinitionSelectorValidationRequest{},
			want: "missing-process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMissingProcessDefinitionSelectors(processDefinitionSelectorValidationResult{
				Request:               tt.req,
				MissingBpmnProcessIDs: []string{"missing-process"},
			})

			require.Contains(t, got, "["+tt.want+"]")
			require.NotContains(t, got, "bpmnProcessId:")
		})
	}
}

func TestProcessDefinitionSelectorNoPromptError_ReturnsNilForValidResult(t *testing.T) {
	require.NoError(t, processDefinitionSelectorNoPromptError(processDefinitionSelectorValidationResult{}))
}

// Interactive diagnostics may offer recovery, but the primary error must remain terse and parseable.
func TestProcessDefinitionSelectorHumanDiagnostic_SingleMissingSelectorOffersListing(t *testing.T) {
	resetProcessDefinitionSelectorPromptTestState(t)
	processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }

	cmd, _ := newProcessDefinitionSelectorValidationTestCommand()
	var prompt string
	confirmProcessDefinitionSelectorListVisibleFn = func(autoConfirm bool, got string) error {
		require.False(t, autoConfirm)
		prompt = got
		return localPreconditionError(ErrCmdAborted)
	}

	result := processDefinitionSelectorValidationResult{
		MissingBpmnProcessIDs: []string{"missing-process"},
	}
	err := processDefinitionSelectorValidationError(result)
	recoveryErr := processDefinitionSelectorRecovery(cmd, stubProcessAPI{}, result)

	require.Error(t, err)
	require.NoError(t, recoveryErr)
	require.True(t, errors.Is(err, ferrors.ErrLocalPrecondition))
	require.EqualError(t, err, "no visible process definition matches the provided selector: [missing-process]")
	require.NotContains(t, err.Error(), "bpmnProcessId: missing-process")
	require.Equal(t, "List visible process definitions?", prompt)
	require.NotContains(t, prompt, "credentials may not have access")
	require.NotContains(t, prompt, "\n\n")
}

// Multiple misses use one prompt and one compact error instead of printing a help-style explanation per selector.
func TestProcessDefinitionSelectorHumanDiagnostic_MultipleMissingSelectorsOffersListing(t *testing.T) {
	resetProcessDefinitionSelectorPromptTestState(t)
	processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }

	cmd, _ := newProcessDefinitionSelectorValidationTestCommand()
	var prompt string
	confirmProcessDefinitionSelectorListVisibleFn = func(autoConfirm bool, got string) error {
		require.False(t, autoConfirm)
		prompt = got
		return localPreconditionError(ErrCmdAborted)
	}

	result := processDefinitionSelectorValidationResult{
		MissingBpmnProcessIDs: []string{"missing-a", "missing-b"},
	}
	err := processDefinitionSelectorValidationError(result)
	recoveryErr := processDefinitionSelectorRecovery(cmd, stubProcessAPI{}, result)

	require.Error(t, err)
	require.NoError(t, recoveryErr)
	require.EqualError(t, err, "no visible process definitions match the provided selector(s): [missing-a], [missing-b]")
	require.NotContains(t, err.Error(), "bpmnProcessId: missing-a")
	require.NotContains(t, err.Error(), "bpmnProcessId: missing-b")
	require.Equal(t, "List visible process definitions?", prompt)
	require.NotContains(t, prompt, "credentials may not have access")
	require.NotContains(t, prompt, "\n\n")
}

// Non-human modes must fail without launching the recovery listing path.
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
			confirmProcessDefinitionSelectorListVisibleFn = func(bool, string) error {
				t.Fatal("unexpected process-definition selector listing prompt")
				return nil
			}

			result := processDefinitionSelectorValidationResult{
				MissingBpmnProcessIDs: []string{"missing-process"},
			}
			err := processDefinitionSelectorValidationError(result)

			require.Error(t, err)
			require.False(t, processDefinitionSelectorPromptAllowed(cmd))
			require.Contains(t, err.Error(), "no visible process definition matches the provided selector")
			require.NotContains(t, err.Error(), "List visible process definitions?")
		})
	}
}

// Accepted recovery prompts reuse the normal process-definition table so users see familiar rows.
func TestProcessDefinitionSelectorValidationError_AcceptedPromptListsVisibleDefinitions(t *testing.T) {
	resetProcessDefinitionSelectorPromptTestState(t)
	processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }

	cmd, output := newProcessDefinitionSelectorValidationTestCommand()
	var prompt string
	confirmProcessDefinitionSelectorListVisibleFn = func(autoConfirm bool, got string) error {
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

	result := processDefinitionSelectorValidationResult{
		MissingBpmnProcessIDs: []string{"missing-process"},
	}
	err := processDefinitionSelectorValidationError(result)
	recoveryErr := processDefinitionSelectorRecovery(cmd, cli, result)

	require.Error(t, err)
	require.NoError(t, recoveryErr)
	require.EqualError(t, err, "no visible process definition matches the provided selector: [missing-process]")
	require.Equal(t, "List visible process definitions?", prompt)
	require.NotContains(t, prompt, "credentials may not have access")
	require.NotContains(t, prompt, "\n\n")
	require.Contains(t, output.String(), "2251799813685250")
	require.Contains(t, output.String(), "tenant-a")
	require.Contains(t, output.String(), "invoice")
	require.Contains(t, output.String(), "v2")
	require.Contains(t, output.String(), "2251799813685251")
	require.Contains(t, output.String(), "order")
	require.Contains(t, output.String(), "v7/stable")
	require.Contains(t, output.String(), "found: 2")
}

// Near-match recovery uses the validation result instead of issuing another broad listing request.
func TestProcessDefinitionSelectorValidationError_AcceptedPromptListsNearMatches(t *testing.T) {
	resetProcessDefinitionSelectorPromptTestState(t)
	processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }

	cmd, output := newProcessDefinitionSelectorValidationTestCommand()
	var prompt string
	confirmProcessDefinitionSelectorListVisibleFn = func(autoConfirm bool, got string) error {
		require.False(t, autoConfirm)
		prompt = got
		return nil
	}
	cli := stubProcessAPI{
		searchProcessDefinitions: func(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
			t.Fatal("near-match listing should use already discovered process definitions")
			return process.ProcessDefinitions{}, nil
		},
	}

	result := processDefinitionSelectorValidationResult{
		Request: processDefinitionSelectorValidationRequest{
			ProcessVersion:    99,
			ProcessVersionTag: "missing",
		},
		NearMatchesByBpmnProcessID: map[string]process.ProcessDefinitions{
			"order": {
				Total: 1,
				Items: []process.ProcessDefinition{
					{Key: "pd-order-v2", TenantId: "<default>", BpmnProcessId: "order", ProcessVersion: 2, ProcessVersionTag: "stable"},
				},
			},
		},
		MissingBpmnProcessIDs: []string{"order"},
	}
	err := processDefinitionSelectorValidationError(result)
	recoveryErr := processDefinitionSelectorRecovery(cmd, cli, result)

	require.Error(t, err)
	require.NoError(t, recoveryErr)
	require.EqualError(t, err, "no visible process definition matches the provided selector: [order v99/missing]")
	require.Equal(t, "List matching process definitions?", prompt)
	require.NotContains(t, prompt, "credentials may not have access")
	require.Contains(t, output.String(), "pd-order-v2")
	require.Contains(t, output.String(), "<default>")
	require.Contains(t, output.String(), "order")
	require.Contains(t, output.String(), "v2/stable")
	require.Contains(t, output.String(), "found: 1")
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
	confirmProcessDefinitionSelectorListVisibleFn = confirmCmdOrAbortDefaultYes
	t.Cleanup(func() {
		resetProcessInstanceCommandGlobals()
		processDefinitionSelectorInteractiveTerminalFn = processDefinitionSelectorInteractiveTerminal
		confirmProcessDefinitionSelectorListVisibleFn = confirmCmdOrAbortDefaultYes
	})
}
