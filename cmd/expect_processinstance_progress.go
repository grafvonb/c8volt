// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

const expectProcessInstanceCompletionPhase = "expect process instances"

func newExpectProcessInstanceSemanticProgress(cmd *cobra.Command, total int) *opsSemanticProgressReporter {
	if total <= 1 {
		return nil
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	return newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			Phase:         expectProcessInstanceCompletionPhase,
			ActivityLabel: "waiting for process-instance expectations",
			CoreResource:  "process instance(s)",
			Total:         total,
			ConfirmedVerb: "satisfied",
			FailedVerb:    "failed",
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(channel),
	})
}

func appendExpectProcessInstanceProgressOption(opts []processOptions.FacadeOption, reporter *opsSemanticProgressReporter) []processOptions.FacadeOption {
	if reporter == nil {
		return opts
	}
	out := append([]processOptions.FacadeOption{}, opts...)
	return append(out, processOptions.WithProgress(func(event processOptions.ProgressEvent) {
		if event.Kind != processOptions.ProgressEventKindCompletion || event.Completion == nil {
			return
		}
		if strings.TrimSpace(event.Completion.Phase) != expectProcessInstanceCompletionPhase {
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
	}))
}
