// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const cancelDeleteRelativeDayNow = "2026-04-10T12:00:00Z"

type cancelDryRunPreviewFixture struct {
	RequestedKeys      typex.Keys
	ResolvedRoots      typex.Keys
	AffectedFamilyKeys typex.Keys
	TraversalOutcome   process.TraversalOutcome
	Warning            string
	MissingAncestors   []process.MissingAncestor
}

func newCancelDryRunPreviewFixture() cancelDryRunPreviewFixture {
	return cancelDryRunPreviewFixture{
		RequestedKeys:      typex.Keys{"2251799813711967"},
		ResolvedRoots:      typex.Keys{"2251799813711900"},
		AffectedFamilyKeys: typex.Keys{"2251799813711900", "2251799813711967"},
		TraversalOutcome:   process.TraversalOutcomePartial,
		Warning:            "one or more parent process instances were not found",
		MissingAncestors:   []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
	}
}

func requireCancelDryRunPreviewPayload(t *testing.T, payload map[string]any, want cancelDryRunPreviewFixture) {
	t.Helper()

	require.Equal(t, "cancel", payload["operation"])
	requireDryRunPreviewStringSlice(t, payload, "requestedKeys", want.RequestedKeys)
	requireDryRunPreviewStringSlice(t, payload, "resolvedRoots", want.ResolvedRoots)
	requireDryRunPreviewStringSlice(t, payload, "affectedFamilyKeys", want.AffectedFamilyKeys)
	require.Equal(t, float64(len(want.RequestedKeys)), payload["requestedCount"])
	require.Equal(t, float64(len(want.ResolvedRoots)), payload["resolvedRootCount"])
	require.Equal(t, float64(len(want.AffectedFamilyKeys)), payload["affectedCount"])
	require.Equal(t, string(want.TraversalOutcome), payload["traversalOutcome"])
	require.Equal(t, want.TraversalOutcome == process.TraversalOutcomeComplete, payload["scopeComplete"])
	require.Equal(t, want.Warning, payload["warning"])
	requireDryRunPreviewMissingAncestors(t, payload, want.MissingAncestors)
	require.Equal(t, false, payload["mutationSubmitted"])
}

func TestCancelProcessInstanceDryRunPreviewPayloadMapping(t *testing.T) {
	want := newCancelDryRunPreviewFixture()
	preview := newProcessInstanceDryRunPreview("cancel", want.RequestedKeys, process.DryRunPIKeyExpansion{
		Roots:            want.ResolvedRoots,
		Collected:        want.AffectedFamilyKeys,
		MissingAncestors: want.MissingAncestors,
		Warning:          want.Warning,
		Outcome:          want.TraversalOutcome,
	})

	var payload map[string]any
	b, err := json.Marshal(preview)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &payload))

	requireCancelDryRunPreviewPayload(t, payload, want)
}

func TestCancelProcessInstanceDryRun_HumanOutputIncludesInspectableScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	preview := newProcessInstanceDryRunPreview("cancel", typex.Keys{"child-human"}, process.DryRunPIKeyExpansion{
		Roots:     typex.Keys{"root-human"},
		Collected: typex.Keys{"root-human", "child-human", "sibling-human"},
		Outcome:   process.TraversalOutcomeComplete,
	})

	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))

	output := buf.String()
	require.Contains(t, output, "dry run: cancel process-instance")
	require.Contains(t, output, "requested process instances: 1")
	require.Contains(t, output, "resolved root process instances: 1")
	require.Contains(t, output, "affected process instances: 3")
	require.Contains(t, output, "scope: complete")
	require.Contains(t, output, "requested keys: child-human")
	require.Contains(t, output, "resolved root keys: root-human")
	require.Contains(t, output, "affected family keys: root-human, child-human, sibling-human")
	require.Contains(t, output, "no mutation submitted: cancel was not submitted")
}

func TestCancelProcessInstanceDryRun_StructuredOutputIncludesInspectableScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagViewAsJson = true

	want := newCancelDryRunPreviewFixture()
	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	preview := newProcessInstanceDryRunPreview("cancel", want.RequestedKeys, process.DryRunPIKeyExpansion{
		Roots:            want.ResolvedRoots,
		Collected:        want.AffectedFamilyKeys,
		MissingAncestors: want.MissingAncestors,
		Warning:          want.Warning,
		Outcome:          want.TraversalOutcome,
	})

	require.NoError(t, renderProcessInstanceDryRunPreview(cmd, preview))

	payload := requireDryRunEnvelopePayload(t, buf.String())
	requireCancelDryRunPreviewPayload(t, payload, want)
}

func TestCancelProcessInstanceDryRun_KeyedChildEscalatesToRootWithoutMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during cancel dry run")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"child-1"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-1"},
				Collected: typex.Keys{"root-1", "child-1"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	got, err := cancelProcessInstancesWithPlan(cmd, cli, typex.Keys{"child-1"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, typex.Keys{"root-1"}, typex.Keys(got.DryRunPreview.ResolvedRoots))
	require.Equal(t, typex.Keys{"root-1", "child-1"}, typex.Keys(got.DryRunPreview.AffectedFamilyKeys))
	require.Contains(t, buf.String(), "dry run: cancel process-instance")
	require.Contains(t, buf.String(), "requested keys: child-1")
	require.Contains(t, buf.String(), "resolved root keys: root-1")
	require.Contains(t, buf.String(), "affected family keys: root-1, child-1")
	require.Contains(t, buf.String(), "no mutation submitted: cancel was not submitted")
}

func TestCancelProcessInstanceDryRun_KeyedRootReportsFullFamilyWithoutMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during cancel dry run")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"root-1"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-1"},
				Collected: typex.Keys{"root-1", "child-1", "child-2"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	got, err := cancelProcessInstancesWithPlan(cmd, cli, typex.Keys{"root-1"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 3, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, typex.Keys{"root-1"}, typex.Keys(got.DryRunPreview.ResolvedRoots))
	require.Equal(t, typex.Keys{"root-1", "child-1", "child-2"}, typex.Keys(got.DryRunPreview.AffectedFamilyKeys))
	require.Contains(t, buf.String(), "affected process instances: 3")
	require.Contains(t, buf.String(), "scope: complete")
	require.Contains(t, buf.String(), "affected family keys: root-1, child-1, child-2")
	require.Contains(t, buf.String(), "no mutation submitted: cancel was not submitted")
}

func TestCancelProcessInstanceDryRun_PartialOrphanParentRendersWarningAndMissingAncestor(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during cancel dry-run orphan preview")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"child-orphan"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:            typex.Keys{"root-partial"},
				Collected:        typex.Keys{"root-partial", "child-orphan"},
				MissingAncestors: []process.MissingAncestor{{Key: "missing-parent", StartKey: "child-orphan"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	got, err := cancelProcessInstancesWithPlan(cmd, cli, typex.Keys{"child-orphan"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, process.TraversalOutcomePartial, got.DryRunPreview.TraversalOutcome)
	require.False(t, got.DryRunPreview.ScopeComplete)
	require.Equal(t, []processInstanceDryRunMissingAncestor{{Key: "missing-parent", StartKey: "child-orphan"}}, got.DryRunPreview.MissingAncestors)
	require.Contains(t, buf.String(), "scope: partial")
	require.Contains(t, buf.String(), "warning: one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: missing-parent")
	require.Contains(t, buf.String(), "no mutation submitted: cancel was not submitted")
}

func TestCancelProcessInstanceDryRun_UnresolvedOrphanFailsWithoutMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during unresolved cancel dry run")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"unresolved-child"}, keys)
			return process.DryRunPIKeyExpansion{}, fmt.Errorf("%w: no process instances resolved during dependency expansion", services.ErrOrphanedInstance)
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	got, err := cancelProcessInstancesWithPlan(cmd, cli, typex.Keys{"unresolved-child"}, true)

	require.Error(t, err)
	require.ErrorContains(t, err, "cancel validation")
	require.ErrorContains(t, err, "no process instances resolved during dependency expansion")
	require.Equal(t, processInstancePageActionResult{}, got)
	require.Empty(t, buf.String())
}

func TestCancelProcessInstanceDryRun_SearchPagesAggregateStructuredOutput(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true
	flagViewAsJson = true
	flagGetPISize = 2

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "2"))
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during cancel dry-run search")
		return nil
	}

	var planned []typex.Keys
	var searchedFrom []int32
	cli := stubProcessAPI{
		searchProcessInstancesPage: func(_ context.Context, _ process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			searchedFrom = append(searchedFrom, req.From)
			require.EqualValues(t, 2, req.Size)
			switch req.From {
			case 0:
				return process.ProcessInstancePage{
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateHasMore,
					Items: []process.ProcessInstance{
						{Key: "101", State: process.StateActive},
						{Key: "102", State: process.StateActive},
					},
				}, nil
			case 2:
				return process.ProcessInstancePage{
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateNoMore,
					Items: []process.ProcessInstance{
						{Key: "103", State: process.StateActive},
					},
				}, nil
			default:
				t.Fatalf("unexpected search page offset %d", req.From)
				return process.ProcessInstancePage{}, nil
			}
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			planned = append(planned, append(typex.Keys(nil), keys...))
			switch strings.Join(keys, ",") {
			case "101,102":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"root-a"},
					Collected: typex.Keys{"root-a", "101", "102"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			case "103":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"root-b"},
					Collected: typex.Keys{"root-b"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected dry-run plan keys %v", keys)
				return process.DryRunPIKeyExpansion{}, nil
			}
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	results, err := processPISearchPagesWithAction(cmd, cli, nil, process.ProcessInstanceFilter{}, func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error) {
		keys := make(typex.Keys, 0, len(page.Items))
		for _, pi := range page.Items {
			keys = append(keys, pi.Key)
		}
		return cancelProcessInstancesWithPlanAndRender(cmd, cli, keys, firstPage, false)
	})
	require.NoError(t, err)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 2)
	require.Equal(t, []int32{0, 2}, searchedFrom)
	require.Equal(t, []typex.Keys{{"101", "102"}, {"103"}}, planned)

	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, newProcessInstanceDryRunSummary("cancel", results.DryRunPreviews)))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	previews := requireDryRunSummaryPayload(t, payload, "cancel", 3, 2, 4, 2)

	firstPreview, ok := previews[0].(map[string]any)
	require.True(t, ok)
	requireDryRunPreviewStringSlice(t, firstPreview, "requestedKeys", typex.Keys{"101", "102"})
	requireDryRunPreviewStringSlice(t, firstPreview, "resolvedRoots", typex.Keys{"root-a"})
	requireDryRunPreviewStringSlice(t, firstPreview, "affectedFamilyKeys", typex.Keys{"root-a", "101", "102"})

	secondPreview, ok := previews[1].(map[string]any)
	require.True(t, ok)
	requireDryRunPreviewStringSlice(t, secondPreview, "requestedKeys", typex.Keys{"103"})
	requireDryRunPreviewStringSlice(t, secondPreview, "resolvedRoots", typex.Keys{"root-b"})
	requireDryRunPreviewStringSlice(t, secondPreview, "affectedFamilyKeys", typex.Keys{"root-b"})
}

func TestCancelProcessInstanceDryRun_SearchBatchSizeLimitUsesLimitedPage(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true
	flagGetPISize = 4
	flagGetPILimit = 2

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "4"))
	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during cancel dry-run limited search")
		return nil
	}

	var planned typex.Keys
	var searchRequests []process.ProcessInstancePageRequest
	cli := stubProcessAPI{
		searchProcessInstancesPage: func(_ context.Context, _ process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			searchRequests = append(searchRequests, req)
			require.EqualValues(t, 4, req.Size)
			return process.ProcessInstancePage{
				Request:       req,
				OverflowState: process.ProcessInstanceOverflowStateHasMore,
				Items: []process.ProcessInstance{
					{Key: "201", State: process.StateActive},
					{Key: "202", State: process.StateActive},
					{Key: "203", State: process.StateActive},
					{Key: "204", State: process.StateActive},
				},
			}, nil
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			planned = append(typex.Keys(nil), keys...)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-limit"},
				Collected: typex.Keys{"root-limit", "201", "202"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	results, err := processPISearchPagesWithAction(cmd, cli, nil, process.ProcessInstanceFilter{}, func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error) {
		keys := make(typex.Keys, 0, len(page.Items))
		for _, pi := range page.Items {
			keys = append(keys, pi.Key)
		}
		return cancelProcessInstancesWithPlanAndRender(cmd, cli, keys, firstPage, false)
	})
	require.NoError(t, err)
	require.Len(t, searchRequests, 1)
	require.EqualValues(t, 0, searchRequests[0].From)
	require.Equal(t, typex.Keys{"201", "202"}, planned)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 1)
	require.Equal(t, 2, results.DryRunPreviews[0].RequestedCount)
}

func TestCancelCommand_CommandLocalBackoffTimeoutEnvOverridesProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "27s")

	cfg := resolveCommandConfigForTest(t, cancelCmd, writeBackoffPrecedenceConfig(t), nil)

	require.Equal(t, 27*time.Second, cfg.App.Backoff.Timeout)
}

func TestCancelHelp_DocumentsConfirmationAndNoWaitSemantics(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"cancel"}, []string{
		"Cancel running process instances",
		"--auto-confirm",
		"waits for the observed cancellation",
		"./c8volt cancel pi --state active --batch-size 200 --auto-confirm",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"cancel", "process-instance"}, []string{
		"validates the affected root and descendant instances",
		"Use --force when a selected child must be escalated",
		"Use --auto-confirm for unattended destructive runs",
		"verify later with `get pi` or `expect pi`",
		"number of process instances to process per page",
		"maximum number of matching process instances to process across all pages",
		"./c8volt expect pi --key <process-instance-key> --state canceled",
		"./c8volt cancel pi --state active --batch-size 250 --limit 25",
	}, []string{"--count"})
	require.Contains(t, output, "--force")
	require.Contains(t, output, "--batch-size int32")
	require.Contains(t, output, "--limit int32")
}

// Verifies search-mode cancellation builds the expected date-filtered search request and no-ops cleanly on empty matches.
func TestCancelProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceSearchScaffoldHelper", cfgPath)

	filter := decodeCapturedPISearchFilter(t, requests)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

// Verifies date-filtered search selection cancels matched instances and keeps descendant lookup behavior intact.
func TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatches(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))

			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			if parentKey == "2251799813711967" {
				_, _ = w.Write([]byte(`{"items":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/2251799813711967/cancellation":
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.Equal(t, []string{"/v2/process-instances/2251799813711967/cancellation"}, cancelled.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

// Verifies relative-day search selection derives canonical start-date bounds before cancelling matches.
func TestCancelProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndCancelsMatches(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))

			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			if parentKey == "2251799813711967" {
				_, _ = w.Write([]byte(`{"items":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-11T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-11T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/2251799813711967/cancellation":
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndCancelsMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.Equal(t, []string{"/v2/process-instances/2251799813711967/cancellation"}, cancelled.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-03-11T00:00:00Z")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

// Verifies cancel no-ops successfully when a date-filtered search returns no process instances.
func TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatches(t *testing.T) {
	var requests []string

	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

// Verifies a relative-day-only filter is sufficient to trigger search mode.
func TestCancelProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficient(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.NotContains(t, output, "either at least one --key is required, or sufficient filtering options")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

func TestCancelProcessInstanceCommand_SearchPagingPromptFlow(t *testing.T) {
	var requests safeSlice[string]
	var cancelled safeSlice[string]
	searchPage := 0
	var searchMu sync.Mutex

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			searchMu.Lock()
			defer searchMu.Unlock()
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"101","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"102","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"103","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/101/cancellation" || r.URL.Path == "/v2/process-instances/102/cancellation" || r.URL.Path == "/v2/process-instances/103/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	var prompts []string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		prompts = append(prompts, prompt)
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 0, pages[0]["from"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.ElementsMatch(t, []string{
		"/v2/process-instances/101/cancellation",
		"/v2/process-instances/102/cancellation",
		"/v2/process-instances/103/cancellation",
	}, cancelled.Snapshot())
	require.Len(t, prompts, 2)
	require.Contains(t, prompts[0], "You are about to cancel 2 process instance(s)")
	require.Contains(t, prompts[1], "Processed 2 process instance(s) on this page (2 requested so far, 2 including dependencies). More matching process instances remain. Continue?")
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
	require.NotContains(t, output, "next step: auto-continue")
}

func TestCancelProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotals(t *testing.T) {
	var requests safeSlice[string]
	var cancelled safeSlice[string]
	searchPage := 0
	var searchMu sync.Mutex

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if parentKey, ok := filter["parentKey"]; ok && parentKey != nil {
					parent := int64(parentKey.(float64))
					if parent != 701 && parent != 702 {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"items":[]}`))
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"key":%d,"parentKey":%d,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}]}`, parent+1000, parent)))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			searchMu.Lock()
			defer searchMu.Unlock()
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"key":701,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"},{"key":702,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"key":701,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"},{"key":702,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"},{"key":703,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/701/cancellation" || r.URL.Path == "/v2/process-instances/702/cancellation" || r.URL.Path == "/v2/process-instances/703/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")
	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper", cfgPath)

	sizes := decodeCapturedTopLevelPISearchSizes(t, requests.Snapshot())
	require.Equal(t, exitcode.Error, code)
	require.Equal(t, []float64{2}, sizes)
	require.Empty(t, cancelled.Snapshot())
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance direct lookup by key is not tenant-safe in Camunda 8.7")
}

func TestCancelProcessInstancesWithPlan_PrintsOrphanWarningForKeyedPreflight(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	var prompt string
	confirmCmdOrAbortFn = func(_ bool, got string) error {
		prompt = got
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"2251799813711967"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:            typex.Keys{"2251799813711900"},
				Collected:        typex.Keys{"2251799813711900", "2251799813711967"},
				MissingAncestors: []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		cancelProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.CancelReports, error) {
			require.Equal(t, typex.Keys{"2251799813711900"}, keys)
			require.Zero(t, wantedWorkers)
			require.Equal(t, 2, options.ApplyFacadeOptions(opts).AffectedProcessInstanceCount)
			return process.CancelReports{Items: []process.CancelReport{{Key: "2251799813711900", Ok: true}}}, nil
		},
	}

	got, err := cancelProcessInstancesWithPlan(cmd, cli, typex.Keys{"2251799813711967"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.Contains(t, prompt, "requested to cancel 1 process instance(s)")
	require.Contains(t, prompt, "a total of 2 instance(s) with 1 root instance(s) will be canceled")
	require.Contains(t, buf.String(), "warning: one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: 1 (use --verbose to list keys)")
	require.NotContains(t, buf.String(), "missing ancestor keys: 2251799813711999")
}

func TestCancelProcessInstancePage_PrintsOrphanWarningForPagedPreflight(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"2251799813711967"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:            typex.Keys{"2251799813711967"},
				Collected:        typex.Keys{"2251799813711967"},
				MissingAncestors: []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		cancelProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.CancelReports, error) {
			require.Equal(t, typex.Keys{"2251799813711967"}, keys)
			require.Zero(t, wantedWorkers)
			require.Equal(t, 1, options.ApplyFacadeOptions(opts).AffectedProcessInstanceCount)
			return process.CancelReports{Items: []process.CancelReport{{Key: "2251799813711967", Ok: true}}}, nil
		},
	}

	got, err := cancelProcessInstancePage(cmd, cli, typex.Keys{"2251799813711967"}, false)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 1, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.Contains(t, buf.String(), "warning: one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: 2251799813711999")
}

func TestCancelProcessInstanceCommand_SearchPagingAutoConfirmFlow(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"201","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"202","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"203","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/201/cancellation" || r.URL.Path == "/v2/process-instances/202/cancellation" || r.URL.Path == "/v2/process-instances/203/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--auto-confirm",
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/201/cancellation",
		"/v2/process-instances/202/cancellation",
		"/v2/process-instances/203/cancellation",
	}, cancelled.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

func TestCancelProcessInstanceCommand_SearchPagingLimitFlow(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"201","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"202","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"203","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"204","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/201/cancellation" || r.URL.Path == "/v2/process-instances/202/cancellation" || r.URL.Path == "/v2/process-instances/203/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--auto-confirm",
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
		"--limit", "3",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/201/cancellation",
		"/v2/process-instances/202/cancellation",
		"/v2/process-instances/203/cancellation",
	}, cancelled.Snapshot())
	require.NotContains(t, strings.Join(cancelled.Snapshot(), "\n"), "204")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: yes, next step: limit-reached")
}

func TestCancelProcessInstanceCommand_SearchPagingBatchSizeLimitFlow(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"401","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"402","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"403","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"404","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":6,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/401/cancellation" || r.URL.Path == "/v2/process-instances/402/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--auto-confirm",
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "4",
		"--limit", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 1)
	require.EqualValues(t, 4, pages[0]["limit"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/401/cancellation",
		"/v2/process-instances/402/cancellation",
	}, cancelled.Snapshot())
	require.NotContains(t, strings.Join(cancelled.Snapshot(), "\n"), "403")
	require.Contains(t, output, "page size: 4, current page: 2, total so far: 2, more matches: yes, next step: limit-reached")
}

func TestCancelProcessInstanceCommand_SearchPagingAutomationFlow(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"301","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"302","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"303","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/301/cancellation" || r.URL.Path == "/v2/process-instances/302/cancellation" || r.URL.Path == "/v2/process-instances/303/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--automation",
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/301/cancellation",
		"/v2/process-instances/302/cancellation",
		"/v2/process-instances/303/cancellation",
	}, cancelled.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

func TestCancelProcessInstanceCommand_SearchPagingPartialCompletionSummary(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"211","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"212","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/211/cancellation" || r.URL.Path == "/v2/process-instances/212/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	callCount := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		callCount++
		if callCount == 1 {
			return nil
		}
		return ErrCmdAborted
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/211/cancellation",
		"/v2/process-instances/212/cancellation",
	}, cancelled.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
	require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
}

func TestCancelProcessInstanceCommand_SearchPagingWarningStopSummary(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"221","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"222","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{}}`))
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/221/cancellation" || r.URL.Path == "/v2/process-instances/222/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error { return nil }
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/221/cancellation",
		"/v2/process-instances/222/cancellation",
	}, cancelled.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
	require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
}

func TestCancelProcessInstanceCommand_DirectKeyBypassesTopLevelSearchPaging(t *testing.T) {
	var requests []string
	var cancelled safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))
			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			w.Header().Set("Content-Type", "application/json")
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
			}
			_, _ = w.Write([]byte(`{"items":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/301/cancellation":
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/301":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"301","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error { return nil }
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--tenant", "tenant",
		"--json",
		"cancel", "process-instance",
		"--key", "301",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Empty(t, pages)
	require.Equal(t, []string{"/v2/process-instances/301/cancellation"}, cancelled.Snapshot())
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Equal(t, "cancel process-instance", got["command"])
	require.Contains(t, stderr, "INFO")
}

func TestCancelProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetail(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/301", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"resource not found"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetailHelper", cfgPath)

	require.Equal(t, exitcode.NotFound, code)
	require.Contains(t, output, "resource not found")
	require.Contains(t, output, "cancel validation")
	require.Contains(t, output, "ancestry")
	require.NotContains(t, output, "validating process instance keys for cancellation")
	require.NotContains(t, output, "ancestry get")
	require.Contains(t, output, "get process instance")
	require.Less(t, strings.Index(output, "cancel validation"), strings.Index(output, "ancestry"))
	require.Less(t, strings.Index(output, "ancestry"), strings.Index(output, "get process instance"))
	require.NotContains(t, output, "fetching process instance with key")
}

// Verifies invalid --state values are rejected through the shared invalid-args error path.
func TestCancelProcessInstanceCommand_RejectsInvalidSearchState(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidSearchStateHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "invalid value for --state")
}

// Verifies invalid date literals for date flags are rejected with a clear YYYY-MM-DD validation error.
func TestCancelProcessInstanceCommand_RejectsInvalidDateFilter(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidDateFilterHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
}

// Verifies reversed date ranges are rejected when the after-bound is later than the before-bound.
func TestCancelProcessInstanceCommand_RejectsInvalidDateRange(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidDateRangeHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid range for --end-date-after and --end-date-before: "2026-02-01" is later than "2026-01-31"`)
}

// Verifies date filters cannot be combined with direct key lookup mode.
func TestCancelProcessInstanceCommand_RejectsKeyAndDateFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsKeyAndDateFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
}

// Verifies relative-day filters cannot be combined with direct key lookup mode.
func TestCancelProcessInstanceCommand_RejectsKeyAndRelativeDayFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsKeyAndRelativeDayFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
}

func TestCancelProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "removed count flag is rejected",
			helper: "TestCancelProcessInstanceCommand_RejectsRemovedCountFlagHelper",
			want:   "unknown flag: --count",
		},
		{
			name:   "non-positive limit is rejected",
			helper: "TestCancelProcessInstanceCommand_RejectsInvalidLimitHelper",
			want:   "--limit must be positive integer",
		},
		{
			name:   "limit cannot be combined with key",
			helper: "TestCancelProcessInstanceCommand_RejectsLimitWithKeyHelper",
			want:   "--limit cannot be combined with --key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeCancelProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, tt.want)
		})
	}
}

// Verifies process-instance date filters are rejected for Camunda 8.7 where the capability is unsupported.
func TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Verifies relative-day process-instance filters are also rejected for Camunda 8.7.
func TestCancelProcessInstanceCommand_RejectsRelativeDayFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

func executeCancelProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
	t.Helper()

	output, err := testx.RunCmdSubprocess(t, helperName, map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	return string(output), exitErr.ExitCode()
}

func executeCancelProcessInstanceSuccessHelper(t *testing.T, helperName string, cfgPath string) (string, error) {
	t.Helper()

	output, err := testx.RunCmdSubprocess(t, helperName, map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	})
	out := string(output)
	if err != nil {
		return out, err
	}
	return out, nil
}

func TestCancelProcessInstanceCommand_RejectsRemovedCountFlagHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--count", "2"}

	Execute()
}

func TestCancelProcessInstanceCommand_RejectsInvalidLimitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--limit", "-1"}

	Execute()
}

func TestCancelProcessInstanceCommand_RejectsLimitWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--key", "123", "--limit", "1"}

	Execute()
}

// Helper-process entrypoint for the search scaffold failure test.
func TestCancelProcessInstanceSearchScaffoldHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

func TestCancelProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "--verbose", "cancel", "process-instance", "--state", "active", "--no-wait", "--batch-size", "2"}

	Execute()
}

// Helper-process entrypoint for the successful search-select-and-cancel flow test.
func TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the successful relative-day search-select-and-cancel flow test.
func TestCancelProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndCancelsMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-newer-days", "30", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the no-matches failure test.
func TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for relative-day-only sufficiency validation.
func TestCancelProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--start-date-older-days", "72"}

	Execute()
}

// Helper-process entrypoint for invalid --state validation.
func TestCancelProcessInstanceCommand_RejectsInvalidSearchStateHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "broken", "--bpmn-process-id", "order-process"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestCancelProcessInstanceCommand_RejectsInvalidDateFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for invalid date range validation.
func TestCancelProcessInstanceCommand_RejectsInvalidDateRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--end-date-after", "2026-02-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for key-and-date-filter exclusivity validation.
func TestCancelProcessInstanceCommand_RejectsKeyAndDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}

// Helper-process entrypoint for key-and-relative-day-filter exclusivity validation.
func TestCancelProcessInstanceCommand_RejectsKeyAndRelativeDayFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--key", "2251799813711967", "--start-date-newer-days", "30"}

	Execute()
}

// Helper-process entrypoint for version capability validation on Camunda 8.7.
func TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01"}

	Execute()
}

// Helper-process entrypoint for relative-day version capability validation on Camunda 8.7.
func TestCancelProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-newer-days", "30"}

	Execute()
}

func TestCancelProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetailHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "cancel", "process-instance", "--key", "301", "--no-wait"}

	Execute()
}
