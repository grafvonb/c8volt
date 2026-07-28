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

// TestFormatOpsPreflightScopeRendersConsequencesAndConfirmationContext verifies broad selector summaries stay compact and certainty-aware.
func TestFormatOpsPreflightScopeRendersConsequencesAndConfirmationContext(t *testing.T) {
	total := int64(10000)
	pages := int64(10)

	got := formatOpsPreflightScope(ops.PreflightScope{
		Command:         "ops analyse slow-process-instances",
		SelectorSummary: "OrderProcess",
		CoreResource:    "process_instance",
		Total:           &total,
		TotalKind:       ops.TotalCertaintyLowerBound,
		PageSize:        1000,
		PageCount:       &pages,
		PageCountKind:   ops.PageCountKindEstimated,
		ConsequenceSummary: ops.ConsequenceSummary{
			WorkSummary: "discover all matches and load runtime element timelines",
			RiskSummary: "read-only, expensive",
		},
		RequiresConfirmation: true,
	})

	require.Equal(t, []string{
		"slow analysis scope: OrderProcess matched at least 10000 process instances; page size: 1000; discovery pages: at least 10",
		"slow analysis is expensive: discover all matches and load runtime element timelines",
	}, got)
}

// TestFormatOpsPreflightScopeLabelsExactLowerBoundAndUnknownTotals verifies count wording covers all US1 certainty cases.
func TestFormatOpsPreflightScopeLabelsExactLowerBoundAndUnknownTotals(t *testing.T) {
	tests := []struct {
		name      string
		total     *int64
		kind      ops.TotalCertainty
		pageCount *int64
		pageKind  ops.PageCountKind
		want      string
	}{
		{name: "exact", total: ptrInt64(2000), kind: ops.TotalCertaintyExact, pageCount: ptrInt64(2), pageKind: ops.PageCountKindExact, want: "slow analysis scope: OrderProcess matched 2000 process instances; page size: 1000; discovery pages: 2"},
		{name: "zero exact", total: ptrInt64(0), kind: ops.TotalCertaintyExact, pageKind: ops.PageCountKindUnknown, want: "slow analysis scope: OrderProcess matched no process instances; page size: 1000"},
		{name: "lower bound", total: ptrInt64(2000), kind: ops.TotalCertaintyLowerBound, pageCount: ptrInt64(2), pageKind: ops.PageCountKindEstimated, want: "slow analysis scope: OrderProcess matched at least 2000 process instances; page size: 1000; discovery pages: at least 2"},
		{name: "unknown", kind: ops.TotalCertaintyUnknown, pageKind: ops.PageCountKindUnknown, want: "slow analysis scope: OrderProcess matched an unknown number of process instances; page size: 1000"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatOpsPreflightScope(ops.PreflightScope{
				Command:         "ops analyse slow-process-instances",
				SelectorSummary: "OrderProcess",
				CoreResource:    "process_instance",
				Total:           tc.total,
				TotalKind:       tc.kind,
				PageSize:        1000,
				PageCount:       tc.pageCount,
				PageCountKind:   tc.pageKind,
			})

			require.Equal(t, tc.want, got[0])
		})
	}
}

// TestFormatOpsPreflightScopeRendersEmptyScopeWithoutBlankLines verifies zero-match output stays calm and grammatical.
func TestFormatOpsPreflightScopeRendersEmptyScopeWithoutBlankLines(t *testing.T) {
	total := int64(0)

	got := formatOpsPreflightScope(ops.PreflightScope{
		Command:         "ops analyse slow-process-instances",
		SelectorSummary: "EmptyProcess",
		CoreResource:    "process_instance",
		Total:           &total,
		TotalKind:       ops.TotalCertaintyExact,
		PageSize:        1000,
		PageCountKind:   ops.PageCountKindUnknown,
		ConsequenceSummary: ops.ConsequenceSummary{
			WorkSummary: "none; no runtime element timelines will be loaded",
		},
	})

	require.Equal(t, []string{
		"slow analysis scope: EmptyProcess matched no process instances; page size: 1000",
		"slow analysis: none; no runtime element timelines will be loaded",
	}, got)
}

// TestFormatOpsPreflightScopeNamesBasicInspectionResources verifies the shared preflight formatter has stable resource labels for the basic get rollout.
func TestFormatOpsPreflightScopeNamesBasicInspectionResources(t *testing.T) {
	total := int64(3000)
	pages := int64(3)
	tests := []struct {
		name         string
		coreResource string
		command      string
		selector     string
		want         string
	}{
		{name: "process instances", coreResource: "process_instance", command: "get process-instance", selector: "active instances", want: "process-instance search scope: active instances matched 3000 process instances; page size: 1000; discovery pages: 3"},
		{name: "incidents", coreResource: "incident", command: "get incident", selector: "active incidents", want: "incident search scope: active incidents matched 3000 incidents; page size: 1000; discovery pages: 3"},
		{name: "jobs", coreResource: "job", command: "get job", selector: "failed jobs", want: "job search scope: failed jobs matched 3000 jobs; page size: 1000; discovery pages: 3"},
		{name: "elements", coreResource: "element", command: "get element", selector: "active elements", want: "element search scope: active elements matched 3000 elements; page size: 1000; discovery pages: 3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatOpsPreflightScope(ops.PreflightScope{
				Command:         tc.command,
				SelectorSummary: tc.selector,
				CoreResource:    tc.coreResource,
				Total:           &total,
				TotalKind:       ops.TotalCertaintyExact,
				PageSize:        1000,
				PageCount:       &pages,
				PageCountKind:   ops.PageCountKindExact,
			})

			require.Equal(t, tc.want, got[0])
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
