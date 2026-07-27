// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/stretchr/testify/require"
)

// TestFormatOpsTotalCertaintyLabelsCountSemantics verifies count wording never implies lower-bound or unknown totals are exact.
func TestFormatOpsTotalCertaintyLabelsCountSemantics(t *testing.T) {
	total := int64(10000)

	require.Equal(t, "10000 process instance(s)", formatOpsTotalCertainty(&total, ops.TotalCertaintyExact, "process instance(s)"))
	require.Equal(t, "10000+ process instance(s)", formatOpsTotalCertainty(&total, ops.TotalCertaintyLowerBound, "process instance(s)"))
	require.Equal(t, "about 10000 process instance(s)", formatOpsTotalCertainty(&total, ops.TotalCertaintyEstimated, "process instance(s)"))
	require.Equal(t, "unknown process instance(s)", formatOpsTotalCertainty(nil, ops.TotalCertaintyUnknown, "process instance(s)"))
}

// TestFormatOpsPageCountUsesExactEstimatedAndUnknownWording verifies page progress does not invent certainty.
func TestFormatOpsPageCountUsesExactEstimatedAndUnknownWording(t *testing.T) {
	pages := int64(10)

	require.Equal(t, "page 4/10", formatOpsPageCount(4, &pages, ops.PageCountKindExact))
	require.Equal(t, "page 4/~10", formatOpsPageCount(4, &pages, ops.PageCountKindEstimated))
	require.Equal(t, "page 4", formatOpsPageCount(4, nil, ops.PageCountKindUnknown))
	require.Empty(t, formatOpsPageCount(0, &pages, ops.PageCountKindExact))
}

// TestFormatOpsFrozenScopeProgressGatesETA verifies exact counters can show elapsed/rate while ETA waits for a remaining estimate.
func TestFormatOpsFrozenScopeProgressGatesETA(t *testing.T) {
	rate := 4.25
	remaining := 95 * time.Second

	require.Equal(t, "loading runtime elements, 48/800 process instance(s), 2m0s elapsed, ~4.2/s", formatOpsFrozenScopeProgress(ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         48,
		Total:        800,
		Elapsed:      2 * time.Minute,
		Rate:         &rate,
	}))
	require.Equal(t, "loading runtime elements, 48/800 process instance(s), 2m0s elapsed, ~4.2/s, ~1m35s remaining", formatOpsFrozenScopeProgress(ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         48,
		Total:        800,
		Elapsed:      2 * time.Minute,
		Rate:         &rate,
		ETA:          &remaining,
	}))
}

// TestOpsProgressChannelForModeProtectsMachineOutput verifies progress is never routed to stdout for script-safe modes.
func TestOpsProgressChannelForModeProtectsMachineOutput(t *testing.T) {
	require.Equal(t, ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}, opsProgressChannelForMode(opsProgressModeInput{RenderMode: RenderModeOneLine}))
	require.Equal(t, ops.ProgressChannel{Mode: ops.ProgressModeVerbose, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}, opsProgressChannelForMode(opsProgressModeInput{RenderMode: RenderModeOneLine, Verbose: true}))
	require.Equal(t, ops.ProgressChannel{Mode: ops.ProgressModeJSON}, opsProgressChannelForMode(opsProgressModeInput{RenderMode: RenderModeJSON}))
	require.Equal(t, ops.ProgressChannel{Mode: ops.ProgressModeKeysOnly}, opsProgressChannelForMode(opsProgressModeInput{RenderMode: RenderModeKeysOnly}))
	require.Equal(t, ops.ProgressChannel{Mode: ops.ProgressModeQuiet}, opsProgressChannelForMode(opsProgressModeInput{RenderMode: RenderModeOneLine, Quiet: true}))
	require.Equal(t, ops.ProgressChannel{Mode: ops.ProgressModeAutomation, StructuredReportAllowed: true}, opsProgressChannelForMode(opsProgressModeInput{RenderMode: RenderModeOneLine, Automation: true}))
}

// TestOpsETAAllowedRequiresSamplesExactTotalAndRemaining verifies approximate remaining time needs a complete timing window.
func TestOpsETAAllowedRequiresSamplesExactTotalAndRemaining(t *testing.T) {
	remaining := 30 * time.Second

	require.False(t, opsETAAllowed(ops.ETASampleWindow{Total: 800, CompletedSamples: 2, Elapsed: time.Minute, Remaining: &remaining}))
	require.False(t, opsETAAllowed(ops.ETASampleWindow{MinimumSamplesMet: true, CompletedSamples: 2, Elapsed: time.Minute, Remaining: &remaining}))
	require.False(t, opsETAAllowed(ops.ETASampleWindow{MinimumSamplesMet: true, Total: 800, Elapsed: time.Minute, Remaining: &remaining}))
	require.False(t, opsETAAllowed(ops.ETASampleWindow{MinimumSamplesMet: true, Total: 800, CompletedSamples: 2, Remaining: &remaining}))
	require.False(t, opsETAAllowed(ops.ETASampleWindow{MinimumSamplesMet: true, Total: 800, CompletedSamples: 2, Elapsed: time.Minute}))
	require.True(t, opsETAAllowed(ops.ETASampleWindow{MinimumSamplesMet: true, Total: 800, CompletedSamples: 2, Elapsed: time.Minute, Remaining: &remaining}))
}
