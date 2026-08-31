// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// opsProgressModeInput captures the command state that decides where progress may be written.
type opsProgressModeInput struct {
	RenderMode RenderMode
	Verbose    bool
	Quiet      bool
	Automation bool
	Debug      bool
}

// opsSemanticProgressOutputPolicy captures command-owned visibility decisions
// for completion-driven progress without exposing renderer details to services.
type opsSemanticProgressOutputPolicy struct {
	Channel           ops.ProgressChannel
	TransientActivity bool
	PacedAggregate    bool
	VerboseItems      bool
	FailureWarnings   bool
	Stdout            bool
	Stderr            bool
}

// opsProgressModeForCommand keeps progress gating tied to existing root flags and render mode.
func opsProgressModeForCommand(cmd *cobra.Command, mode RenderMode) opsProgressModeInput {
	return opsProgressModeInput{
		RenderMode: mode,
		Verbose:    flagVerbose,
		Quiet:      flagQuiet,
		Automation: automationModeEnabled(cmd),
		Debug:      flagDebug,
	}
}

// opsProgressChannelForMode applies the shared stdout-safe progress channel contract.
func opsProgressChannelForMode(input opsProgressModeInput) ops.ProgressChannel {
	switch {
	case input.Quiet:
		return ops.ProgressChannel{Mode: ops.ProgressModeQuiet}
	case input.Automation:
		return ops.ProgressChannel{Mode: ops.ProgressModeAutomation, StructuredReportAllowed: true}
	case input.RenderMode == RenderModeJSON:
		return ops.ProgressChannel{Mode: ops.ProgressModeJSON}
	case input.RenderMode == RenderModeKeysOnly:
		return ops.ProgressChannel{Mode: ops.ProgressModeKeysOnly}
	case input.Debug:
		return ops.ProgressChannel{Mode: ops.ProgressModeDebug, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}
	case input.Verbose:
		return ops.ProgressChannel{Mode: ops.ProgressModeVerbose, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}
	default:
		return ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}
	}
}

// opsSemanticProgressOutputPolicyForChannel translates the shared channel
// contract into semantic reporter behavior for completion facts.
func opsSemanticProgressOutputPolicyForChannel(channel ops.ProgressChannel) opsSemanticProgressOutputPolicy {
	policy := opsSemanticProgressOutputPolicy{
		Channel: channel,
		Stdout:  channel.StdoutAllowed,
		Stderr:  channel.StderrAllowed,
	}
	switch channel.Mode {
	case ops.ProgressModeHuman:
		policy.TransientActivity = channel.TransientAllowed
		policy.PacedAggregate = channel.DurableAllowed && channel.StderrAllowed && !channel.StdoutAllowed
		policy.FailureWarnings = channel.StderrAllowed && !channel.StdoutAllowed
	case ops.ProgressModeVerbose, ops.ProgressModeDebug:
		policy.TransientActivity = channel.TransientAllowed
		policy.VerboseItems = channel.DurableAllowed && channel.StderrAllowed && !channel.StdoutAllowed
		policy.FailureWarnings = channel.StderrAllowed && !channel.StdoutAllowed
	case ops.ProgressModeQuiet:
		policy.FailureWarnings = true
	}
	return policy
}
