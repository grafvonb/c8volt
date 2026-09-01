// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"time"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// runProcessInstanceSemanticProgressNow is overridden by command tests to
// exercise durable milestone pacing without real sleeps.
var runProcessInstanceSemanticProgressNow = time.Now

// appendExplicitLargeWorkProgressOption installs stdout-safe progress routing for explicit-count or explicit-key workflows.
func appendExplicitLargeWorkProgressOption(cmd *cobra.Command, opts []processOptions.FacadeOption) []processOptions.FacadeOption {
	out := append([]processOptions.FacadeOption{}, opts...)
	return append(out, processOptions.WithProgress(func(event processOptions.ProgressEvent) {
		if event.Kind != processOptions.ProgressEventKindFrozenScope || event.FrozenScope == nil {
			return
		}
		progress := opsFrozenScopeProgressFromProcessOption(*event.FrozenScope)
		printExplicitLargeWorkProgressEvent(cmd, ops.ProgressEvent{
			Kind:        ops.ProgressEventKindFrozenScope,
			FrozenScope: &progress,
		})
	}))
}

// newRunProcessInstanceSemanticProgressReporter owns the finite bulk-start
// completion scope; plain walk/search callers continue using transient frozen
// progress through appendExplicitLargeWorkProgressOption.
func newRunProcessInstanceSemanticProgressReporter(cmd *cobra.Command, total int) *opsSemanticProgressReporter {
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	return newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			Phase:                     "create",
			ActivityLabel:             "starting process instances",
			CoreResource:              "process instance(s)",
			Total:                     total,
			AffectedResource:          "process instances",
			AffectedCoverageAvailable: true,
			SubmittedVerb:             "submitted",
			ConfirmedVerb:             "started",
			FailedVerb:                "failed",
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(channel),
		Now:    runProcessInstanceSemanticProgressNow,
	})
}

// appendRunProcessInstanceProgressOption preserves the existing frozen-scope
// progress stream while routing bulk-start completion facts to the semantic
// reporter used for exact workflow activity.
func appendRunProcessInstanceProgressOption(cmd *cobra.Command, opts []processOptions.FacadeOption, reporter *opsSemanticProgressReporter) []processOptions.FacadeOption {
	out := append([]processOptions.FacadeOption{}, opts...)
	return append(out, processOptions.WithProgress(func(event processOptions.ProgressEvent) {
		switch event.Kind {
		case processOptions.ProgressEventKindFrozenScope:
			if event.FrozenScope == nil {
				return
			}
			progress := opsFrozenScopeProgressFromProcessOption(*event.FrozenScope)
			printExplicitLargeWorkDurableFrozenProgress(cmd, progress)
		case processOptions.ProgressEventKindCompletion:
			if event.Completion == nil || reporter == nil {
				return
			}
			reporter.Report(ops.ProgressEvent{
				Kind: ops.ProgressEventKindCompletion,
				Completion: &ops.CompletionProgress{
					Phase:            event.Completion.Phase,
					CoreResource:     event.Completion.CoreResource,
					Total:            event.Completion.Total,
					Identity:         event.Completion.Identity,
					Disposition:      ops.CompletionDisposition(event.Completion.Disposition),
					FailureDetail:    event.Completion.FailureDetail,
					AffectedResource: event.Completion.AffectedResource,
					AffectedCount:    event.Completion.AffectedCount,
				},
			})
		}
	}))
}

// printExplicitLargeWorkDurableFrozenProgress preserves legacy verbose frozen
// counters without letting them replace a semantic workflow activity update.
func printExplicitLargeWorkDurableFrozenProgress(cmd *cobra.Command, progress ops.FrozenScopeProgress) {
	line := formatProcessInstanceMutationFrozenProgress(progress)
	if line == "" {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	if !opsSlowProcessAnalysisDurableProgressAllowed(channel) {
		return
	}
	printOpsDurableLine(cmd, line, false)
}

// printExplicitLargeWorkProgressEvent renders exact explicit-work counters using the shared command progress gate.
func printExplicitLargeWorkProgressEvent(cmd *cobra.Command, event ops.ProgressEvent) {
	if event.Kind != ops.ProgressEventKindFrozenScope || event.FrozenScope == nil {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	printOpsSlowProcessAnalysisProgress(cmd, formatProcessInstanceMutationFrozenProgress(*event.FrozenScope), channel)
}
