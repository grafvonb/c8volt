// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type processInstanceMutationTenantEmitter struct {
	name string
	emit func(*cobra.Command, tenant.Context)
}

type processInstanceMutationTenantLogRecord struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

var processInstanceMutationTenantEmitters = []processInstanceMutationTenantEmitter{
	{
		name: "durable progress",
		emit: func(cmd *cobra.Command, ctx tenant.Context) {
			attachTenantContext(cmd, ctx)
			printProcessInstanceMutationTenantContext(cmd, ops.ProgressChannel{
				Mode:           ops.ProgressModeHuman,
				DurableAllowed: true,
				StderrAllowed:  true,
			})
		},
	},
	{
		name: "confirmation context",
		emit: renderProcessInstanceMutationTenantContextStderr,
	},
}

var processInstanceMutationTenantExpectedRecords = []processInstanceMutationTenantLogRecord{
	{Level: "INFO", Msg: "configured tenant: tenant-a"},
	{Level: "WARN", Msg: `--tenant "" overrides the configured tenant filter; selection is unfiltered`},
	{Level: "INFO", Msg: "selection scope: unfiltered across accessible tenants"},
	{Level: "WARN", Msg: "affected tenants: tenant-a, tenant-b"},
	{Level: "WARN", Msg: "tenant metadata is unknown for 1 target"},
}

// TestProcessInstanceMutationTenantSeverity verifies both mutation tenant
// emitters honor the standard plain/JSON formats and INFO/WARN/ERROR levels.
func TestProcessInstanceMutationTenantSeverity(t *testing.T) {
	formats := []struct {
		name  string
		value string
	}{
		{name: "plain", value: "plain"},
		{name: "json", value: "json"},
	}
	levels := []struct {
		name string
		want []processInstanceMutationTenantLogRecord
	}{
		{name: "info", want: processInstanceMutationTenantExpectedRecords},
		{name: "warn", want: []processInstanceMutationTenantLogRecord{
			processInstanceMutationTenantExpectedRecords[1],
			processInstanceMutationTenantExpectedRecords[3],
			processInstanceMutationTenantExpectedRecords[4],
		}},
		{name: "error"},
	}

	for _, emitter := range processInstanceMutationTenantEmitters {
		for _, format := range formats {
			for _, level := range levels {
				t.Run(emitter.name+"/"+format.name+"/"+level.name, func(t *testing.T) {
					resetProcessInstanceCommandGlobals()
					t.Cleanup(resetProcessInstanceCommandGlobals)

					stdout := &bytes.Buffer{}
					stderr := &bytes.Buffer{}
					cmd := &cobra.Command{}
					cmd.SetOut(stdout)
					cmd.SetErr(stderr)
					cmd.SetContext(tenantOverrideProvenance{
						ConfiguredTenantID: "tenant-a",
						ExplicitTenantID:   "",
						Explicit:           true,
					}.ToContext(logging.ToContext(context.Background(), logging.New(logging.LoggerConfig{
						Level:  level.name,
						Format: format.value,
						Writer: stderr,
					}))))
					ctx := withTenantContextEvidence(newDiscoveryTenantContext(""), []string{"tenant-b", "tenant-a"}, 1)

					emitter.emit(cmd, ctx)

					require.Empty(t, stdout.String())
					if format.value == "json" {
						requireProcessInstanceMutationTenantJSONRecords(t, stderr.String(), level.want)
						return
					}
					requireProcessInstanceMutationTenantPlainRecords(t, stderr.String(), level.want)
				})
			}
		}
	}
}

func requireProcessInstanceMutationTenantPlainRecords(t *testing.T, output string, expected []processInstanceMutationTenantLogRecord) {
	t.Helper()
	if len(expected) == 0 {
		require.Empty(t, output)
		return
	}

	plainLine := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}[+-]\d{2}:\d{2}) (INFO|WARN) (.*)$`)
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	require.Len(t, lines, len(expected))
	for i, want := range expected {
		matches := plainLine.FindStringSubmatch(lines[i])
		require.Len(t, matches, 4, "line %d must use the standard timestamped plain format: %q", i, lines[i])
		_, err := time.Parse(logging.PlainTimestampLayout, matches[1])
		require.NoError(t, err, "line %d must begin with a valid timestamp", i)
		require.Equal(t, want.Level, matches[2])
		require.Equal(t, want.Msg, matches[3])
	}
}

func requireProcessInstanceMutationTenantJSONRecords(t *testing.T, output string, expected []processInstanceMutationTenantLogRecord) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	for i, want := range expected {
		var got processInstanceMutationTenantLogRecord
		require.NoError(t, decoder.Decode(&got), "record %d must be valid JSON", i)
		_, err := time.Parse(time.RFC3339Nano, got.Time)
		require.NoError(t, err, "record %d must contain a valid time", i)
		require.Equal(t, want.Level, got.Level)
		require.Equal(t, want.Msg, got.Msg)
	}
	var extra any
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
}

// TestProcessInstanceMutationTenantFilteredFirstReport verifies a filtered
// emission marks tenant context rendered and never falls back or replays later.
func TestProcessInstanceMutationTenantFilteredFirstReport(t *testing.T) {
	for _, emitter := range processInstanceMutationTenantEmitters {
		for _, format := range []string{"plain", "json"} {
			t.Run(emitter.name+"/"+format, func(t *testing.T) {
				resetProcessInstanceCommandGlobals()
				t.Cleanup(resetProcessInstanceCommandGlobals)

				stdout := &bytes.Buffer{}
				fallback := &bytes.Buffer{}
				filtered := &bytes.Buffer{}
				permissive := &bytes.Buffer{}
				cmd := &cobra.Command{}
				cmd.SetOut(stdout)
				cmd.SetErr(fallback)
				cmd.SetContext(logging.ToContext(context.Background(), logging.New(logging.LoggerConfig{
					Level:  "error",
					Format: format,
					Writer: filtered,
				})))
				ctx := withTenantContextEvidence(newDiscoveryTenantContext(""), []string{"tenant-a", "tenant-b"}, 1)

				emitter.emit(cmd, ctx)
				require.True(t, tenantContextHumanRendered(cmd))
				require.Empty(t, stdout.String())
				require.Empty(t, fallback.String(), "filtering must not use raw stderr fallback")
				require.Empty(t, filtered.String())

				cmd.SetContext(logging.ToContext(cmd.Context(), logging.New(logging.LoggerConfig{
					Level:  "info",
					Format: format,
					Writer: permissive,
				})))
				emitter.emit(cmd, ctx)

				require.Empty(t, stdout.String())
				require.Empty(t, fallback.String())
				require.Empty(t, permissive.String(), "a permissive logger must not replay a filtered first report")
			})
		}
	}
}

// TestProcessInstanceMutationTenantFallbackAndDeduplication verifies raw
// configured-stderr fallback and rendered-state suppression across both paths.
func TestProcessInstanceMutationTenantFallbackAndDeduplication(t *testing.T) {
	ctx := withTenantContextEvidence(newDiscoveryTenantContext("tenant-a"), []string{"tenant-a"}, 0)
	want := "selection scope: tenant-a only\naffected tenants: tenant-a\n"
	channel := ops.ProgressChannel{
		Mode:           ops.ProgressModeHuman,
		DurableAllowed: true,
		StderrAllowed:  true,
	}

	tests := []struct {
		name string
		emit func(*cobra.Command)
		want string
	}{
		{
			name: "progress same path",
			emit: func(cmd *cobra.Command) {
				attachTenantContext(cmd, ctx)
				printProcessInstanceMutationTenantContext(cmd, channel)
				printProcessInstanceMutationTenantContext(cmd, channel)
			},
			want: want,
		},
		{
			name: "confirmation same path",
			emit: func(cmd *cobra.Command) {
				renderProcessInstanceMutationTenantContextStderr(cmd, ctx)
				renderProcessInstanceMutationTenantContextStderr(cmd, ctx)
			},
			want: want,
		},
		{
			name: "progress then confirmation",
			emit: func(cmd *cobra.Command) {
				attachTenantContext(cmd, ctx)
				printProcessInstanceMutationTenantContext(cmd, channel)
				renderProcessInstanceMutationTenantContextStderr(cmd, ctx)
			},
			want: want,
		},
		{
			name: "confirmation then progress",
			emit: func(cmd *cobra.Command) {
				attachTenantContext(cmd, ctx)
				renderProcessInstanceMutationTenantContextStderr(cmd, ctx)
				printProcessInstanceMutationTenantContext(cmd, channel)
			},
			want: want,
		},
		{
			name: "absent progress context",
			emit: func(cmd *cobra.Command) {
				printProcessInstanceMutationTenantContext(cmd, channel)
			},
		},
		{
			name: "zero confirmation context",
			emit: func(cmd *cobra.Command) {
				renderProcessInstanceMutationTenantContextStderr(cmd, tenant.Context{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			t.Cleanup(resetProcessInstanceCommandGlobals)

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd := &cobra.Command{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)

			tt.emit(cmd)

			require.Empty(t, stdout.String())
			require.Equal(t, tt.want, stderr.String())
		})
	}
}

// pendingProcessInstanceMutationProgressT064 marks the historical progress
// contract gate while the concrete tests define the preserved behavior.
func pendingProcessInstanceMutationProgressT064(t *testing.T) {
	t.Helper()
}

// TestCancelProcessInstanceSearchProgressContractPendingT064 defines the shared
// destructive progress contract for search-selected cancel.
func TestCancelProcessInstanceSearchProgressContractPendingT064(t *testing.T) {
	pendingProcessInstanceMutationProgressT064(t)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true
	flagGetPISize = 1

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "1"))
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
		require.True(t, autoConfirm)
		require.Contains(t, prompt, "cancel")
		return nil
	}

	cli := stubProcessAPI{
		planProcessInstanceMutationPages: func(_ context.Context, request process.ProcessInstanceMutationPlanRequest, visitor process.ProcessInstanceMutationPlanVisitor, opts ...options.FacadeOption) (process.ProcessInstanceMutationPlanPagesResult, error) {
			require.Equal(t, int32(1), request.SearchRequest.Page.Size)
			require.NotNil(t, options.ApplyFacadeOptions(opts).Progress)
			page := process.ProcessInstancePage{
				Items:         []process.ProcessInstance{{Key: "401", State: process.StateActive}},
				Request:       process.ProcessInstancePageRequest{From: 0, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateHasMore,
				ReportedTotal: &process.ProcessInstanceReportedTotal{Count: 2, Kind: process.ProcessInstanceReportedTotalKindLowerBound},
			}
			plan := process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-401"},
				Collected: typex.Keys{"root-401", "401"},
				Outcome:   process.TraversalOutcomeComplete,
			}
			action, err := visitor(process.ProcessInstanceMutationPlanStep{
				Page:             page,
				RequestedKeys:    []string{"401"},
				Plan:             plan,
				CumulativeCount:  1,
				CumulativeImpact: 2,
			})
			require.NoError(t, err)
			require.Equal(t, process.ProcessInstanceSearchPageActionContinue, action)
			return process.ProcessInstanceMutationPlanPagesResult{
				Plans:            []process.ProcessInstanceMutationPlanStep{{Page: page, RequestedKeys: []string{"401"}, Plan: plan, CumulativeCount: 1, CumulativeImpact: 2}},
				Pages:            1,
				RequestedCount:   1,
				CumulativeImpact: 2,
			}, nil
		},
		cancelProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.CancelReports, error) {
			reportProcessInstanceMutationCompletionForTest(t, "cancel", "root-401", 2, keys, opts...)
			return process.CancelReports{Items: []process.CancelReport{{Key: "root-401", Ok: true}}}, nil
		},
	}

	got, err := cancelProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{State: process.StateActive})

	require.NoError(t, err)
	require.Len(t, got.Reports, 1)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "process-instance cancel scope: cancel process-instance matched at least 2 process instances; page size: 1; discovery pages: at least 2")
	require.Contains(t, stderr.String(), "planning process-instance cancel scope 1/1 process instance(s)")
	require.Contains(t, stderr.String(), "root-401 canceled (cancellation process-instance trees, 1/1 process-instance tree(s), affected process instances: 2)")
	require.Contains(t, stderr.String(), "cancellation: canceled 1/1 process-instance tree(s); affected process instances: 2")
	require.NotContains(t, stderr.String(), "/v2/")
	require.NotContains(t, stderr.String(), "cursor")
}

// TestCancelProcessInstanceSearchQuietAndAutomationSuppressProgress verifies machine modes keep mutation progress transient-only.
func TestCancelProcessInstanceSearchQuietAndAutomationSuppressProgress(t *testing.T) {
	pendingProcessInstanceMutationProgressT064(t)

	for _, mode := range []struct {
		name  string
		setup func()
	}{
		{name: "json", setup: func() { flagViewAsJson = true }},
		{name: "quiet", setup: func() { flagQuiet = true }},
		{name: "automation", setup: func() { flagCmdAutomation = true }},
	} {
		t.Run(mode.name, func(t *testing.T) {
			stdout, stderr := exerciseProcessInstanceMutationProgressOutput(t, "cancel", mode.setup)
			require.NotContains(t, stdout, "process-instance cancel scope:")
			require.NotContains(t, stdout, "planning process-instance cancel scope")
			require.NotContains(t, stdout, "cancelling process instances")
			require.NotContains(t, stderr, "process-instance cancel scope:")
			require.NotContains(t, stderr, "planning process-instance cancel scope")
			require.NotContains(t, stderr, "cancelling process instances")
		})
	}
}

// TestProcessInstanceMutationProgress_AttachedDiscoveryTenantContextPrecedesVerbosePreflight
// verifies shared process-instance progress can render the attached discovery
// context before verbose mutation preflight scope.
func TestProcessInstanceMutationProgress_AttachedDiscoveryTenantContextPrecedesVerbosePreflight(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagVerbose = true

	cmd := &cobra.Command{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	attachTenantContext(cmd, newDiscoveryTenantContext("tenant-a"))

	progress := newProcessInstanceMutationProgressReporter(cmd, "cancel")
	progress(processInstanceMutationTestPreflightEvent("cancel"))

	require.Empty(t, stdout.String())
	output := stderr.String()
	tenantLine := "selection scope: tenant-a only\n"
	scopeLine := "process-instance cancel scope:"
	require.Contains(t, output, tenantLine)
	require.Contains(t, output, scopeLine)
	require.Less(t, strings.Index(output, tenantLine), strings.Index(output, scopeLine))
}

// TestProcessInstanceMutationProgress_RendersTenantOverrideBeforeScope verifies
// durable PI progress includes explicit tenant broadening provenance before the
// normal discovery scope line.
func TestProcessInstanceMutationProgress_RendersTenantOverrideBeforeScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagVerbose = true

	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)
	cmd.SetContext(tenantOverrideProvenance{
		ConfiguredTenantID: "tenant-a",
		ExplicitTenantID:   "",
		Explicit:           true,
	}.ToContext(context.Background()))
	attachTenantContext(cmd, newDiscoveryTenantContext(""))

	progress := newProcessInstanceMutationProgressReporter(cmd, "cancel")
	progress(processInstanceMutationTestPreflightEvent("cancel"))

	output := stderr.String()
	configuredLine := "configured tenant: tenant-a\n"
	overrideWarning := "--tenant \"\" overrides the configured tenant filter; selection is unfiltered\n"
	tenantLine := "selection scope: unfiltered across accessible tenants\n"
	scopeLine := "process-instance cancel scope:"
	require.Contains(t, output, configuredLine)
	require.Contains(t, output, overrideWarning)
	require.Contains(t, output, tenantLine)
	require.Less(t, strings.Index(output, configuredLine), strings.Index(output, overrideWarning))
	require.Less(t, strings.Index(output, overrideWarning), strings.Index(output, tenantLine))
	require.Less(t, strings.Index(output, tenantLine), strings.Index(output, scopeLine))
}

// TestProcessInstanceMutationProgress_RendersAllTenantsOverrideOnceBeforeScope
// verifies durable mutation progress reports the all-tenants broadening warning
// once before the effective unfiltered discovery scope.
func TestProcessInstanceMutationProgress_RendersAllTenantsOverrideOnceBeforeScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagVerbose = true

	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)
	cmd.SetContext(tenantOverrideProvenance{
		ConfiguredTenantID: "tenant-a",
		AllTenants:         true,
	}.ToContext(context.Background()))
	attachTenantContext(cmd, newDiscoveryTenantContext(""))

	progress := newProcessInstanceMutationProgressReporter(cmd, "cancel")
	progress(processInstanceMutationTestPreflightEvent("cancel"))
	progress(processInstanceMutationTestPreflightEvent("cancel"))

	output := stderr.String()
	configuredLine := "configured tenant: tenant-a\n"
	overrideWarning := "--all-tenants overrides the configured tenant filter; selection is unfiltered\n"
	tenantLine := "selection scope: unfiltered across accessible tenants\n"
	scopeLine := "process-instance cancel scope:"
	require.Contains(t, output, configuredLine)
	require.Contains(t, output, overrideWarning)
	require.Contains(t, output, tenantLine)
	require.Equal(t, 1, strings.Count(output, overrideWarning))
	require.Equal(t, 1, strings.Count(output, tenantLine))
	require.Less(t, strings.Index(output, configuredLine), strings.Index(output, overrideWarning))
	require.Less(t, strings.Index(output, overrideWarning), strings.Index(output, tenantLine))
	require.Less(t, strings.Index(output, tenantLine), strings.Index(output, scopeLine))
}

// TestProcessInstanceMutationProgress_ProtectedModesSuppressAttachedDiscoveryTenantContext
// verifies quiet and keys-only progress modes do not leak tenant context to
// stdout or stderr.
func TestProcessInstanceMutationProgress_ProtectedModesSuppressAttachedDiscoveryTenantContext(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
	}{
		{name: "quiet", setup: func() { flagQuiet = true }},
		{name: "keys only", setup: func() { flagViewKeysOnly = true }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			prevQuiet := flagQuiet
			prevKeysOnly := flagViewKeysOnly
			t.Cleanup(resetProcessInstanceCommandGlobals)
			t.Cleanup(func() {
				flagQuiet = prevQuiet
				flagViewKeysOnly = prevKeysOnly
			})
			tt.setup()

			cmd := &cobra.Command{}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			attachTenantContext(cmd, newDiscoveryTenantContext(""))

			progress := newProcessInstanceMutationProgressReporter(cmd, "cancel")
			progress(processInstanceMutationTestPreflightEvent("cancel"))

			require.Empty(t, stdout.String())
			require.NotContains(t, stderr.String(), "selection scope:")
			require.NotContains(t, stderr.String(), "affected tenants:")
		})
	}
}

// TestProcessInstanceMutationProgress_AllTenantsProtectedModesSuppressProvenance
// verifies protected mutation-progress modes do not leak command-line
// all-tenants provenance to stdout or stderr.
func TestProcessInstanceMutationProgress_AllTenantsProtectedModesSuppressProvenance(t *testing.T) {
	for _, mode := range []struct {
		name  string
		setup func()
	}{
		{name: "json", setup: func() { flagViewAsJson = true }},
		{name: "quiet", setup: func() { flagQuiet = true }},
		{name: "keys only", setup: func() { flagViewKeysOnly = true }},
	} {
		t.Run(mode.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			t.Cleanup(resetProcessInstanceCommandGlobals)
			mode.setup()

			cmd := &cobra.Command{}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetContext(tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				AllTenants:         true,
			}.ToContext(context.Background()))
			attachTenantContext(cmd, newDiscoveryTenantContext(""))

			progress := newProcessInstanceMutationProgressReporter(cmd, "cancel")
			progress(processInstanceMutationTestPreflightEvent("cancel"))

			require.Empty(t, stdout.String())
			require.NotContains(t, stderr.String(), "--all-tenants overrides")
			require.NotContains(t, stderr.String(), "configured tenant:")
			require.NotContains(t, stderr.String(), "selection scope:")
		})
	}
}

// TestCancelProcessInstanceSearchDryRun_RendersMergedTenantWarnings verifies
// search dry-run summaries use aggregate page evidence for resource tenant
// warnings instead of only the base discovery filter line.
func TestCancelProcessInstanceSearchDryRun_RendersMergedTenantWarnings(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true
	flagGetPISize = 1

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "1"))
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cli := stubProcessAPI{
		planProcessInstanceMutationPages: func(_ context.Context, _ process.ProcessInstanceMutationPlanRequest, visitor process.ProcessInstanceMutationPlanVisitor, _ ...options.FacadeOption) (process.ProcessInstanceMutationPlanPagesResult, error) {
			page := process.ProcessInstancePage{
				Items:         []process.ProcessInstance{{Key: "401", State: process.StateActive}},
				Request:       process.ProcessInstancePageRequest{From: 0, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
			}
			plan := process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-401"},
				Collected: typex.Keys{"root-401", "401", "unknown-401"},
				TenantEvidence: process.TenantEvidence{
					ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
					UnknownTargetCount: 1,
					TargetCount:        3,
				},
				Outcome: process.TraversalOutcomeComplete,
			}
			action, err := visitor(process.ProcessInstanceMutationPlanStep{
				Page:             page,
				RequestedKeys:    []string{"401"},
				Plan:             plan,
				CumulativeCount:  1,
				CumulativeImpact: 3,
			})
			require.NoError(t, err)
			require.Equal(t, process.ProcessInstanceSearchPageActionStop, action)
			return process.ProcessInstanceMutationPlanPagesResult{
				Plans:            []process.ProcessInstanceMutationPlanStep{{Page: page, RequestedKeys: []string{"401"}, Plan: plan, CumulativeCount: 1, CumulativeImpact: 3}},
				Pages:            1,
				RequestedCount:   1,
				CumulativeImpact: 3,
				TenantEvidence: process.TenantEvidence{
					ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
					UnknownTargetCount: 1,
					TargetCount:        3,
				},
			}, nil
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	results, err := cancelProcessInstanceSearchPages(cmd, cli, &config.Config{}, process.ProcessInstanceFilter{State: process.StateActive})
	require.NoError(t, err)
	require.Len(t, results.DryRunPreviews, 1)
	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, newProcessInstanceDryRunSummary("cancel", results.DryRunPreviews)))

	output := buf.String()
	tenantLine := "selection scope: unfiltered across accessible tenants\n"
	resourceLine := "affected tenants: tenant-a, tenant-b\n"
	unknownWarning := "tenant metadata is unknown for 1 target\n"
	summaryLine := "dry run: cancel process-instance\n"
	require.Contains(t, output, tenantLine)
	require.Contains(t, output, resourceLine)
	require.Contains(t, output, unknownWarning)
	require.Contains(t, output, summaryLine)
	require.Equal(t, 1, strings.Count(output, resourceLine))
	require.Less(t, strings.Index(output, tenantLine), strings.Index(output, resourceLine))
	require.Less(t, strings.Index(output, resourceLine), strings.Index(output, unknownWarning))
	require.Less(t, strings.Index(output, unknownWarning), strings.Index(output, summaryLine))
}

// exerciseProcessInstanceMutationProgressOutput captures stdout and stderr for
// progress mode gating without running a full destructive command.
func exerciseProcessInstanceMutationProgressOutput(t *testing.T, operation string, setup func()) (string, string) {
	t.Helper()
	resetProcessInstanceCommandGlobals()
	prevQuiet := flagQuiet
	prevAutomation := flagCmdAutomation
	t.Cleanup(func() {
		resetProcessInstanceCommandGlobals()
		flagQuiet = prevQuiet
		flagCmdAutomation = prevAutomation
	})
	if setup != nil {
		setup()
	}

	cmd := &cobra.Command{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	progress := newProcessInstanceMutationProgressReporter(cmd, operation)
	progress(processInstanceMutationTestPreflightEvent(operation))
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "planning process-instance mutation scope",
			CoreResource: "process instance(s)",
			Done:         1,
			Total:        1,
		},
	})
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        processInstanceMutationTestMutationPhase(operation),
			CoreResource: "process instance(s)",
			Done:         1,
			Total:        1,
		},
	})
	return stdout.String(), stderr.String()
}

// processInstanceMutationTestPreflightEvent returns a reusable destructive
// preflight event for progress renderer contract tests.
func processInstanceMutationTestPreflightEvent(operation string) options.ProgressEvent {
	total := int64(1)
	pageCount := int64(1)
	return options.ProgressEvent{
		Kind: options.ProgressEventKindPreflight,
		Preflight: &options.PreflightScope{
			CoreResource:    "process_instance",
			SelectorSummary: operation + " process-instance",
			Total:           &total,
			TotalKind:       options.TotalCertaintyExact,
			PageSize:        1,
			PageCount:       &pageCount,
			PageCountKind:   options.PageCountKindExact,
			ConsequenceSummary: options.ConsequenceSummary{
				WorkSummary: "plan process-instance " + operation + " scope",
				RiskSummary: "destructive mutation",
			},
			RequiresConfirmation: true,
		},
	}
}

// processInstanceMutationTestMutationPhase returns the service phase text the
// mutation progress reporter should route for each destructive operation.
func processInstanceMutationTestMutationPhase(operation string) string {
	switch operation {
	case "cancel":
		return "cancelling process instances"
	case "delete":
		return "deleting process instances"
	default:
		return operation + " process instances"
	}
}

// TestCancelProcessInstanceProgressUsesWorkflowImportance verifies cancel progress stays above nested service waits and requests.
func TestCancelProcessInstanceProgressUsesWorkflowImportance(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	progress := newProcessInstanceMutationProgressReporter(cmd, "cancel")
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "cancelling process instances",
			CoreResource: "process instance(s)",
			Done:         3,
			Total:        10,
		},
	})

	require.Equal(t, []activitysink.Update{{
		Message:    "cancelling process instances 3/10 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}

// TestDeleteProcessInstanceProgressUsesWorkflowImportance verifies delete progress stays above nested service waits and requests.
func TestDeleteProcessInstanceProgressUsesWorkflowImportance(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	progress := newProcessInstanceMutationProgressReporter(cmd, "delete")
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "deleting process instances",
			CoreResource: "process instance(s)",
			Done:         4,
			Total:        12,
		},
	})

	require.Equal(t, []activitysink.Update{{
		Message:    "deleting process instances 4/12 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}

// TestProcessInstanceMutationDirectAndStdinKeysUseSemanticCompletionActivity
// verifies explicit key and stdin-key paths install the same post-confirmation
// semantic reporter for process-instance cancel and delete.
func TestProcessInstanceMutationDirectAndStdinKeysUseSemanticCompletionActivity(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		inputKeys typex.Keys
		run       func(*cobra.Command, process.API, typex.Keys) (processInstancePageActionResult, error)
	}{
		{
			name:      "cancel direct key",
			operation: "cancel",
			inputKeys: typex.Keys{"direct-child"},
			run:       runCancelProcessInstanceDirect,
		},
		{
			name:      "cancel stdin key",
			operation: "cancel",
			inputKeys: typex.Keys{"stdin-child"},
			run:       runCancelProcessInstanceDirect,
		},
		{
			name:      "delete direct key",
			operation: "delete",
			inputKeys: typex.Keys{"direct-child"},
			run:       runDeleteProcessInstanceDirect,
		},
		{
			name:      "delete stdin key",
			operation: "delete",
			inputKeys: typex.Keys{"stdin-child"},
			run:       runDeleteProcessInstanceDirect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			t.Cleanup(resetProcessInstanceCommandGlobals)
			flagCmdAutoConfirm = true

			sink := &activitysink.Sink{}
			cmd := &cobra.Command{}
			cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

			prevConfirm := confirmCmdOrAbortFn
			t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
			confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
				require.True(t, autoConfirm)
				require.Contains(t, prompt, tt.operation)
				return nil
			}

			cli := stubProcessAPI{
				dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, opts ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
					require.Equal(t, tt.inputKeys, keys)
					require.True(t, options.ApplyFacadeOptions(opts).IgnoreTenant)
					return process.DryRunPIKeyExpansion{
						Roots:     typex.Keys{"root-1"},
						Collected: typex.Keys{"root-1", tt.inputKeys[0]},
						Outcome:   process.TraversalOutcomeComplete,
					}, nil
				},
				cancelProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.CancelReports, error) {
					reportProcessInstanceMutationCompletionForTest(t, tt.operation, "root-1", 2, keys, opts...)
					return process.CancelReports{Items: []process.CancelReport{{Key: "root-1", Ok: true}}}, nil
				},
				deleteProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.DeleteReports, error) {
					reportProcessInstanceMutationCompletionForTest(t, tt.operation, "root-1", 2, keys, opts...)
					return process.DeleteReports{Items: []process.DeleteReport{{Key: "root-1", Ok: true}}}, nil
				},
			}

			got, err := tt.run(cmd, cli, tt.inputKeys)

			require.NoError(t, err)
			require.Len(t, got.Reports, 1)
			requireProcessInstanceMutationSemanticActivity(t, sink, tt.operation, "root-1")
		})
	}
}

// TestProcessInstanceMutationDirectAndStdinKeysShareLifecycleWording verifies
// direct-key and stdin-key-equivalent paths render the same submitted and
// confirmed lifecycle semantics for cancel and delete completions.
func TestProcessInstanceMutationDirectAndStdinKeysShareLifecycleWording(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		inputKeys   typex.Keys
		noWait      bool
		disposition options.CompletionDisposition
		wantItem    string
		wantSummary string
		run         func(*cobra.Command, process.API, typex.Keys) (processInstancePageActionResult, error)
	}{
		{
			name:        "cancel direct waited",
			operation:   "cancel",
			inputKeys:   typex.Keys{"direct-cancel-child"},
			disposition: options.CompletionDispositionConfirmed,
			wantItem:    "root-1 canceled (cancellation process-instance trees, 1/1 process-instance tree(s), affected process instances: 2)",
			wantSummary: "cancellation: canceled 1/1 process-instance tree(s); affected process instances: 2",
			run:         runCancelProcessInstanceDirect,
		},
		{
			name:        "cancel stdin no wait",
			operation:   "cancel",
			inputKeys:   typex.Keys{"stdin-cancel-child"},
			noWait:      true,
			disposition: options.CompletionDispositionSubmitted,
			wantItem:    "root-1 submitted (cancellation process-instance trees, 1/1 process-instance tree(s), affected process instances: 2)",
			wantSummary: "cancellation: submitted 1/1 process-instance tree(s); affected process instances: 2",
			run:         runCancelProcessInstanceDirect,
		},
		{
			name:        "delete direct waited",
			operation:   "delete",
			inputKeys:   typex.Keys{"direct-delete-child"},
			disposition: options.CompletionDispositionConfirmed,
			wantItem:    "root-1 deleted (deletion process-instance trees, 1/1 process-instance tree(s), affected process instances: 2)",
			wantSummary: "deletion: deleted 1/1 process-instance tree(s); affected process instances: 2",
			run:         runDeleteProcessInstanceDirect,
		},
		{
			name:        "delete stdin no wait",
			operation:   "delete",
			inputKeys:   typex.Keys{"stdin-delete-child"},
			noWait:      true,
			disposition: options.CompletionDispositionSubmitted,
			wantItem:    "root-1 submitted (deletion process-instance trees, 1/1 process-instance tree(s), affected process instances: 2)",
			wantSummary: "deletion: submitted 1/1 process-instance tree(s); affected process instances: 2",
			run:         runDeleteProcessInstanceDirect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			t.Cleanup(resetProcessInstanceCommandGlobals)
			flagCmdAutoConfirm = true
			flagVerbose = true
			flagNoWait = tt.noWait

			cmd := &cobra.Command{}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)

			prevConfirm := confirmCmdOrAbortFn
			t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
			confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
				require.True(t, autoConfirm)
				require.Contains(t, prompt, tt.operation)
				return nil
			}

			cli := stubProcessAPI{
				dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, opts ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
					require.Equal(t, tt.inputKeys, keys)
					require.True(t, options.ApplyFacadeOptions(opts).IgnoreTenant)
					return process.DryRunPIKeyExpansion{
						Roots:     typex.Keys{"root-1"},
						Collected: typex.Keys{"root-1", tt.inputKeys[0]},
						Outcome:   process.TraversalOutcomeComplete,
					}, nil
				},
				cancelProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.CancelReports, error) {
					reportProcessInstanceMutationCompletionForTestWithDisposition(t, tt.operation, "root-1", 2, keys, tt.disposition, ptrInt(2), opts...)
					return process.CancelReports{Items: []process.CancelReport{{Key: "root-1", Ok: true}}}, nil
				},
				deleteProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.DeleteReports, error) {
					reportProcessInstanceMutationCompletionForTestWithDisposition(t, tt.operation, "root-1", 2, keys, tt.disposition, ptrInt(2), opts...)
					return process.DeleteReports{Items: []process.DeleteReport{{Key: "root-1", Ok: true}}}, nil
				},
			}

			got, err := tt.run(cmd, cli, tt.inputKeys)

			require.NoError(t, err)
			require.Len(t, got.Reports, 1)
			require.NotContains(t, stdout.String(), "process-instance trees")
			require.Contains(t, stderr.String(), tt.wantItem)
			require.Contains(t, stderr.String(), tt.wantSummary)
		})
	}
}

// TestProcessInstanceMutationSemanticProgressScopeMapsLifecycleVocabulary
// verifies cancel/delete scopes use submitted plus operation-specific confirmed
// wording without relying on service-layer prose.
func TestProcessInstanceMutationSemanticProgressScopeMapsLifecycleVocabulary(t *testing.T) {
	tests := []struct {
		operation string
		label     string
		submitted string
		confirmed string
		failed    string
	}{
		{
			operation: "cancel",
			label:     "cancellation process-instance trees",
			submitted: "submitted",
			confirmed: "canceled",
			failed:    "failed",
		},
		{
			operation: "delete",
			label:     "deletion process-instance trees",
			submitted: "submitted",
			confirmed: "deleted",
			failed:    "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			scope := processInstanceMutationSemanticProgressScope(tt.operation, 2, true)

			require.Equal(t, tt.operation, scope.Phase)
			require.Equal(t, tt.label, scope.ActivityLabel)
			require.Equal(t, tt.submitted, scope.SubmittedVerb)
			require.Equal(t, tt.confirmed, scope.ConfirmedVerb)
			require.Equal(t, tt.failed, scope.FailedVerb)
		})
	}
}

// TestProcessInstanceMutationSemanticProgressFailureAndUnknownAffected verifies
// failed process-instance mutation facts warn immediately and permanently omit
// affected counts when the scope cannot prove per-root affected coverage.
func TestProcessInstanceMutationSemanticProgressFailureAndUnknownAffected(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagVerbose = true

	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)
	reporter := newProcessInstanceMutationSemanticReporter(cmd, "delete", processInstancePageImpact{Requested: 2, Affected: 3, Roots: 2})
	callback := processInstanceMutationSemanticProgressCallback(reporter)

	reportProcessInstanceMutationCompletionEvent(callback, "delete", "root-1", 2, options.CompletionDispositionFailed, "delete rejected", nil)
	reportProcessInstanceMutationCompletionEvent(callback, "delete", "root-2", 2, options.CompletionDispositionConfirmed, "", ptrInt(1))
	reporter.Close()

	output := stderr.String()
	require.Contains(t, output, "root-1 failed: delete rejected (deletion process-instance trees, 1/2 process-instance tree(s), 1 failed)")
	require.Contains(t, output, "root-2 deleted (deletion process-instance trees, 2/2 process-instance tree(s), 1 failed)")
	require.NotContains(t, output, "affected process instances:")
}

// reportProcessInstanceMutationCompletionForTest emits a confirmed completion
// fact through the facade progress callback and checks mutation options shared
// by direct and search command paths.
func reportProcessInstanceMutationCompletionForTest(t *testing.T, operation string, expectedRoot string, affectedCount int, keys typex.Keys, opts ...options.FacadeOption) {
	t.Helper()
	reportProcessInstanceMutationCompletionForTestWithDisposition(t, operation, expectedRoot, affectedCount, keys, options.CompletionDispositionConfirmed, ptrInt(affectedCount), opts...)
}

// reportProcessInstanceMutationCompletionForTestWithDisposition keeps command
// path tests explicit about submitted, confirmed, failed, and unknown affected
// completion facts while preserving common option assertions.
func reportProcessInstanceMutationCompletionForTestWithDisposition(t *testing.T, operation string, expectedRoot string, affectedCount int, keys typex.Keys, disposition options.CompletionDisposition, affected *int, opts ...options.FacadeOption) {
	t.Helper()
	require.Equal(t, typex.Keys{expectedRoot}, keys)
	cfg := options.ApplyFacadeOptions(opts)
	require.Equal(t, affectedCount, cfg.AffectedProcessInstanceCount)
	require.NotNil(t, cfg.Progress)
	wantWorkflowSuppressed := !flagVerbose || flagQuiet || flagCmdAutomation || pickMode() != RenderModeOneLine
	require.Equal(t, wantWorkflowSuppressed, cfg.SuppressWorkflowDetailLogs)
	require.True(t, cfg.SuppressProcessInstanceDetailLogs)
	cfg.Progress(options.ProgressEvent{
		Kind: options.ProgressEventKindCompletion,
		Completion: &options.CompletionProgress{
			Phase:            operation,
			CoreResource:     "process-instance tree(s)",
			Total:            1,
			Identity:         expectedRoot,
			Disposition:      disposition,
			AffectedResource: "affected process instances",
			AffectedCount:    affected,
		},
	})
}

// requireProcessInstanceMutationSemanticActivity asserts that semantic
// completion updates own workflow-priority aggregate activity instead of
// exposing per-key lifecycle wording in the transient message.
func requireProcessInstanceMutationSemanticActivity(t *testing.T, sink *activitysink.Sink, operation string, identity string) {
	t.Helper()
	label, verb := processInstanceMutationResultWords(operation, false)
	start := activitysink.Start{
		Message:    label + " process-instance trees, 0/1 process-instance tree(s), affected process instances: 0",
		Importance: logging.ActivityImportanceWorkflow,
	}
	update := activitysink.Update{
		Message:    label + " process-instance trees, 1/1 process-instance tree(s), affected process instances: 2",
		Importance: logging.ActivityImportanceWorkflow,
	}
	require.Contains(t, sink.Starts(), start)
	require.Contains(t, sink.PriorityUpdates(), update)
	require.Contains(t, update.Message, label)
	require.NotContains(t, update.Message, identity+" "+verb)
	require.GreaterOrEqual(t, sink.Stopped(), 1)
}

// requireProcessInstanceMutationPlanningStoppedBeforePrompt verifies search
// planning activity is balanced before destructive confirmation can prompt.
func requireProcessInstanceMutationPlanningStoppedBeforePrompt(t *testing.T, sink *activitysink.Sink, operation string) {
	t.Helper()
	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "planning process-instance " + operation + " scope",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Equal(t, 1, sink.Stopped())
}

// TestProcessInstanceMutationSemanticProgressVerboseItemsSuppressAggregateMilestones
// verifies process-instance mutation adapters use verbose per-root completion
// lines instead of paced aggregate milestone duplicates.
func TestProcessInstanceMutationSemanticProgressVerboseItemsSuppressAggregateMilestones(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagVerbose = true
	now := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	processInstanceMutationSemanticProgressNow = func() time.Time { return now }
	t.Cleanup(func() { processInstanceMutationSemanticProgressNow = time.Now })

	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)
	reporter := newProcessInstanceMutationSemanticReporter(cmd, "delete", processInstancePageImpact{Requested: 2, Affected: 2, Roots: 2})
	callback := processInstanceMutationSemanticProgressCallback(reporter)

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportProcessInstanceMutationCompletionEvent(callback, "delete", "root-1", 2, options.CompletionDispositionConfirmed, "", ptrInt(1))
	reportProcessInstanceMutationCompletionEvent(callback, "delete", "root-2", 2, options.CompletionDispositionConfirmed, "", ptrInt(1))
	reporter.Close()

	output := stderr.String()
	require.Contains(t, output, "root-1 deleted (deletion process-instance trees, 1/2 process-instance tree(s), affected process instances: 1)")
	require.Contains(t, output, "root-2 deleted (deletion process-instance trees, 2/2 process-instance tree(s), affected process instances: 2)")
	require.Equal(t, 2, strings.Count(output, "deletion process-instance trees"))
	require.NotContains(t, output, "\ndeletion process-instance trees, 1/2")
	require.NotContains(t, output, "\ndeletion process-instance trees, 2/2")
}

// TestProcessInstanceMutationSemanticProgressFailureWarnsImmediatelyAndFlushes
// verifies a failed process-instance completion produces an immediate warning
// and Close records later unreported aggregate progress exactly once.
func TestProcessInstanceMutationSemanticProgressFailureWarnsImmediatelyAndFlushes(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	now := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	processInstanceMutationSemanticProgressNow = func() time.Time { return now }
	t.Cleanup(func() { processInstanceMutationSemanticProgressNow = time.Now })

	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)
	reporter := newProcessInstanceMutationSemanticReporter(cmd, "cancel", processInstancePageImpact{Requested: 2, Affected: 2, Roots: 2})
	callback := processInstanceMutationSemanticProgressCallback(reporter)

	reportProcessInstanceMutationCompletionEvent(callback, "cancel", "root-1", 2, options.CompletionDispositionFailed, "operation timed out", ptrInt(1))
	reportProcessInstanceMutationCompletionEvent(callback, "cancel", "root-2", 2, options.CompletionDispositionConfirmed, "", ptrInt(1))
	reporter.Close()
	reporter.Close()

	output := stderr.String()
	require.Contains(t, output, "root-1 failed: operation timed out (cancellation process-instance trees, 1/2 process-instance tree(s), 1 failed, affected process instances: 1)")
	require.Equal(t, 1, strings.Count(output, "root-1 failed: operation timed out"))
	require.Equal(t, 1, strings.Count(output, "cancellation process-instance trees, 2/2 process-instance tree(s), 1 failed, affected process instances: 2"))
}

// reportProcessInstanceMutationCompletionEvent sends one facade-level
// completion fact through the process-instance semantic callback under test.
func reportProcessInstanceMutationCompletionEvent(callback func(options.ProgressEvent), phase string, identity string, total int, disposition options.CompletionDisposition, detail string, affected *int) {
	callback(options.ProgressEvent{
		Kind: options.ProgressEventKindCompletion,
		Completion: &options.CompletionProgress{
			Phase:            phase,
			CoreResource:     "process-instance tree(s)",
			Total:            total,
			Identity:         identity,
			Disposition:      disposition,
			FailureDetail:    detail,
			AffectedResource: "affected process instances",
			AffectedCount:    affected,
		},
	})
}

// ptrInt keeps process-instance progress test facts concise while preserving
// nil-versus-zero affected-count semantics in call sites.
func ptrInt(value int) *int {
	return &value
}

// TestDeleteProcessInstanceSearchProgressContractPendingT064 defines the shared
// destructive progress contract for search-selected delete.
func TestDeleteProcessInstanceSearchProgressContractPendingT064(t *testing.T) {
	pendingProcessInstanceMutationProgressT064(t)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true
	flagGetPISize = 1

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "1"))
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
		require.True(t, autoConfirm)
		require.Contains(t, prompt, "delete")
		return nil
	}

	cli := stubProcessAPI{
		planProcessInstanceMutationPages: func(_ context.Context, request process.ProcessInstanceMutationPlanRequest, visitor process.ProcessInstanceMutationPlanVisitor, opts ...options.FacadeOption) (process.ProcessInstanceMutationPlanPagesResult, error) {
			require.Equal(t, int32(1), request.SearchRequest.Page.Size)
			require.NotNil(t, options.ApplyFacadeOptions(opts).Progress)
			page := process.ProcessInstancePage{
				Items:         []process.ProcessInstance{{Key: "401", State: process.StateCompleted}},
				Request:       process.ProcessInstancePageRequest{From: 0, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
				ReportedTotal: &process.ProcessInstanceReportedTotal{Count: 1, Kind: process.ProcessInstanceReportedTotalKindExact},
			}
			plan := process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-401"},
				Collected: typex.Keys{"root-401", "401"},
				Outcome:   process.TraversalOutcomeComplete,
			}
			action, err := visitor(process.ProcessInstanceMutationPlanStep{
				Page:             page,
				RequestedKeys:    []string{"401"},
				Plan:             plan,
				CumulativeCount:  1,
				CumulativeImpact: 2,
			})
			require.NoError(t, err)
			require.Equal(t, process.ProcessInstanceSearchPageActionStop, action)
			return process.ProcessInstanceMutationPlanPagesResult{
				Plans:            []process.ProcessInstanceMutationPlanStep{{Page: page, RequestedKeys: []string{"401"}, Plan: plan, CumulativeCount: 1, CumulativeImpact: 2}},
				Pages:            1,
				RequestedCount:   1,
				CumulativeImpact: 2,
			}, nil
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			reportProcessInstanceMutationCompletionForTest(t, "delete", "root-401", 2, keys, opts...)
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "root-401", Ok: true}}}, nil
		},
	}

	got, err := deleteProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{State: process.StateCompleted})

	require.NoError(t, err)
	require.Len(t, got.Reports, 1)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "process-instance delete scope: delete process-instance matched 1 process instance; page size: 1; discovery pages: 1")
	require.Contains(t, stderr.String(), "planning process-instance delete scope 1/1 process instance(s)")
	require.Contains(t, stderr.String(), "root-401 deleted (deletion process-instance trees, 1/1 process-instance tree(s), affected process instances: 2)")
	require.Contains(t, stderr.String(), "deletion: deleted 1/1 process-instance tree(s); affected process instances: 2")
	require.NotContains(t, stderr.String(), "/v2/")
	require.NotContains(t, stderr.String(), "cursor")
}

// TestDeleteProcessInstanceSearchQuietAndAutomationSuppressProgress verifies machine modes keep mutation progress transient-only.
func TestDeleteProcessInstanceSearchQuietAndAutomationSuppressProgress(t *testing.T) {
	pendingProcessInstanceMutationProgressT064(t)

	for _, mode := range []struct {
		name  string
		setup func()
	}{
		{name: "json", setup: func() { flagViewAsJson = true }},
		{name: "quiet", setup: func() { flagQuiet = true }},
		{name: "automation", setup: func() { flagCmdAutomation = true }},
	} {
		t.Run(mode.name, func(t *testing.T) {
			stdout, stderr := exerciseProcessInstanceMutationProgressOutput(t, "delete", mode.setup)
			require.NotContains(t, stdout, "process-instance delete scope:")
			require.NotContains(t, stdout, "planning process-instance delete scope")
			require.NotContains(t, stdout, "deleting process instances")
			require.NotContains(t, stderr, "process-instance delete scope:")
			require.NotContains(t, stderr, "planning process-instance delete scope")
			require.NotContains(t, stderr, "deleting process instances")
		})
	}
}
