// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
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

// TestFormatOpsPageProgressLabelsPageCountCertainty verifies discovery activity text keeps exact, lower-bound, and unknown page counts distinct.
func TestFormatOpsPageProgressLabelsPageCountCertainty(t *testing.T) {
	tests := []struct {
		name string
		in   ops.PageProgress
		want string
	}{
		{
			name: "known",
			in:   ops.PageProgress{Phase: "discovering process instances", CurrentPage: 4, PageCount: ptrInt64(10), PageCountKind: ops.PageCountKindExact, Seen: 3812, Selected: 3800},
			want: "discovering process instances, page 4/10, 3812 seen, 3800 selected",
		},
		{
			name: "lower bound",
			in:   ops.PageProgress{Phase: "discovering process instances", CurrentPage: 4, PageCount: ptrInt64(10), PageCountKind: ops.PageCountKindEstimated, Seen: 3812},
			want: "discovering process instances, page 4/~10, 3812 seen",
		},
		{
			name: "unknown",
			in:   ops.PageProgress{Phase: "discovering process instances", CurrentPage: 4, PageCountKind: ops.PageCountKindUnknown, Seen: 3812},
			want: "discovering process instances, page 4, 3812 seen",
		},
		{
			name: "user limited",
			in:   ops.PageProgress{Phase: "discovering process instances", CurrentPage: 2, Seen: 1500, Selected: 1000, LimitReached: true},
			want: "discovering process instances, page 2, 1500 seen, 1000 selected, user-limited",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, formatOpsPageProgress(tc.in, "process instance(s)"))
		})
	}
}

// TestFormatOpsFrozenScopeProgressGatesETA verifies exact counters can show elapsed, percent, and rate while ETA waits for a remaining estimate.
func TestFormatOpsFrozenScopeProgressGatesETA(t *testing.T) {
	rate := 4.25
	remaining := 95 * time.Second

	require.Equal(t, "loading runtime elements, 48/800 process instance(s), 6.0%, 2m0s elapsed, ~4.2/s", formatOpsFrozenScopeProgress(ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         48,
		Total:        800,
		Elapsed:      2 * time.Minute,
		Rate:         &rate,
	}))
	require.Equal(t, "loading runtime elements, 48/800 process instance(s), 6.0%, 2m0s elapsed, ~4.2/s, ~1m35s remaining", formatOpsFrozenScopeProgress(ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         48,
		Total:        800,
		Elapsed:      2 * time.Minute,
		Rate:         &rate,
		ETA:          &remaining,
	}))
}

// TestFormatOpsFrozenScopeProgressOmitsPercentForUnknownTotal verifies percent complete is never rendered without an exact total.
func TestFormatOpsFrozenScopeProgressOmitsPercentForUnknownTotal(t *testing.T) {
	require.Equal(t, "loading runtime elements, 3/0 process instance(s), 5s elapsed", formatOpsFrozenScopeProgress(ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         3,
		Elapsed:      5 * time.Second,
	}))
}

// ptrInt64 returns a stable pointer for compact progress formatter fixtures.
func ptrInt64(value int64) *int64 {
	return &value
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

// TestPrintOpsPreflightScopeRendersTenantContextBeforeScope verifies ops
// preflight emits the shared tenant block before expensive-work scope lines.
func TestPrintOpsPreflightScopeRendersTenantContextBeforeScope(t *testing.T) {
	cmd := &cobra.Command{}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	ctx := withTenantContextEvidence(newDiscoveryTenantContext(""), []string{"tenant-b", "tenant-a"}, 1)

	printOpsPreflightScope(cmd, ops.PreflightScope{
		Command:         "ops purge process-instances-with-incidents",
		CoreResource:    "process_instance",
		SelectorSummary: "incident filters",
		Total:           ptrInt64(2),
		TotalKind:       ops.TotalCertaintyExact,
		TenantContext:   &ctx,
	}, ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true})

	got := stderr.String()
	require.Contains(t, got, "Tenant filter: none")
	require.Contains(t, got, "incident purge scope")
	require.Less(t, strings.Index(got, "Tenant filter: none"), strings.Index(got, "incident purge scope"))
	require.Contains(t, got, "Resource tenants: tenant-a, tenant-b")
	require.Contains(t, got, "WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b")
	require.Contains(t, got, "WARNING: tenant metadata is unknown for 1 target")
}

// TestPrintOpsPreflightScopeSuppressesTenantContextForProtectedModes verifies
// quiet and keys-only progress channels keep stdout-safe contracts silent.
func TestPrintOpsPreflightScopeSuppressesTenantContextForProtectedModes(t *testing.T) {
	ctx := withTenantContextEvidence(newDiscoveryTenantContext("tenant-a"), []string{"tenant-a"}, 0)
	for _, channel := range []ops.ProgressChannel{
		{Mode: ops.ProgressModeKeysOnly},
		{Mode: ops.ProgressModeQuiet},
	} {
		cmd := &cobra.Command{}
		var stderr bytes.Buffer
		cmd.SetErr(&stderr)

		printOpsPreflightScope(cmd, ops.PreflightScope{TenantContext: &ctx}, channel)

		require.Empty(t, stderr.String())
		_, ok := attachedTenantContext(cmd)
		require.False(t, ok)
	}
}

// TestOpsProgressDurableMilestoneRequiresElapsedTimeAndPageProgress verifies default-human milestones wait for silence and observed discovery movement.
func TestOpsProgressDurableMilestoneRequiresElapsedTimeAndPageProgress(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}
	event := ops.ProgressEvent{
		Kind: ops.ProgressEventKindPage,
		Page: &ops.PageProgress{Phase: "discovering process instances", CurrentPage: 1, Seen: 1000, Selected: 900},
	}

	require.False(t, pacer.AllowDurableMilestone(event, channel))

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	event.Page.CurrentPage = 2
	event.Page.Seen = 2000
	event.Page.Selected = 1800

	require.True(t, pacer.AllowDurableMilestone(event, channel))
}

// TestOpsProgressDurableMilestoneRequiresForwardProgress verifies elapsed time alone does not repeat the same milestone.
func TestOpsProgressDurableMilestoneRequiresForwardProgress(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}
	event := ops.ProgressEvent{
		Kind: ops.ProgressEventKindPage,
		Page: &ops.PageProgress{Phase: "discovering process instances", CurrentPage: 1, Seen: 1000, Selected: 900},
	}

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	require.True(t, pacer.AllowDurableMilestone(event, channel))

	now = now.Add(opsDurableMilestoneMinimumElapsed)

	require.False(t, pacer.AllowDurableMilestone(event, channel))
}

// TestOpsProgressDurableMilestoneAllowsFrozenScopeProgress verifies frozen-scope counters can drive sparse default-human milestones.
func TestOpsProgressDurableMilestoneAllowsFrozenScopeProgress(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}
	event := ops.ProgressEvent{
		Kind:        ops.ProgressEventKindFrozenScope,
		FrozenScope: &ops.FrozenScopeProgress{Phase: "loading runtime elements", CoreResource: "process instance(s)", Done: 0, Total: 100},
	}

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	require.False(t, pacer.AllowDurableMilestone(event, channel))

	event.FrozenScope.Done = 25

	require.True(t, pacer.AllowDurableMilestone(event, channel))
}

// TestOpsProgressDurableMilestoneSuppressesTimerOnlyETA verifies timing-only ETA updates do not create duplicate durable milestones.
func TestOpsProgressDurableMilestoneSuppressesTimerOnlyETA(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}
	remaining := 5 * time.Minute
	event := ops.ProgressEvent{
		Kind: ops.ProgressEventKindETA,
		ETA: &ops.ETASampleWindow{
			Phase:             "loading runtime elements",
			CompletedSamples:  3,
			Total:             100,
			Elapsed:           30 * time.Second,
			MinimumSamplesMet: true,
			Remaining:         &remaining,
		},
	}

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	require.True(t, pacer.AllowDurableMilestone(event, channel))

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	event.ETA.Elapsed += opsDurableMilestoneMinimumElapsed
	remaining -= time.Minute

	require.False(t, pacer.AllowDurableMilestone(event, channel))
}

// TestOpsProgressDurableMilestoneChannelGatingProtectsMachineModes verifies sparse durable milestones stay out of script-safe modes.
func TestOpsProgressDurableMilestoneChannelGatingProtectsMachineModes(t *testing.T) {
	startedAt := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	now := startedAt
	event := ops.ProgressEvent{
		Kind: ops.ProgressEventKindPage,
		Page: &ops.PageProgress{Phase: "discovering process instances", CurrentPage: 2, Seen: 2000, Selected: 1800},
	}

	tests := []struct {
		name    string
		channel ops.ProgressChannel
		want    bool
	}{
		{name: "default human", channel: ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}, want: true},
		{name: "json", channel: ops.ProgressChannel{Mode: ops.ProgressModeJSON}},
		{name: "keys only", channel: ops.ProgressChannel{Mode: ops.ProgressModeKeysOnly}},
		{name: "quiet", channel: ops.ProgressChannel{Mode: ops.ProgressModeQuiet}},
		{name: "automation", channel: ops.ProgressChannel{Mode: ops.ProgressModeAutomation, StructuredReportAllowed: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
			now = startedAt.Add(opsDurableMilestoneMinimumElapsed)

			require.Equal(t, tc.want, pacer.AllowDurableMilestone(event, tc.channel))
			now = startedAt
		})
	}
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

// TestFormatOpsETASampleWindowUsesApproximateWording verifies standalone ETA messages label throughput and remaining time as approximate.
func TestFormatOpsETASampleWindowUsesApproximateWording(t *testing.T) {
	rate := 2.5
	remaining := 2*time.Minute + 4*time.Second

	require.Empty(t, formatOpsETASampleWindow(ops.ETASampleWindow{Phase: "loading runtime elements", CompletedSamples: 2, Total: 10, Elapsed: time.Second, Rate: &rate, Remaining: &remaining}))
	require.Equal(t, "loading runtime elements, 3/10 sample(s), 2s elapsed, ~2.5/s, ~2m4s remaining", formatOpsETASampleWindow(ops.ETASampleWindow{
		Phase:             "loading runtime elements",
		CompletedSamples:  3,
		Total:             10,
		Elapsed:           2 * time.Second,
		MinimumSamplesMet: true,
		Rate:              &rate,
		Remaining:         &remaining,
	}))
}
