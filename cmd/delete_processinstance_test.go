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
	"strings"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestDeleteProcessInstanceDryRun_KeyedChildEscalatesToRootWithoutMutation
// verifies child selection previews root escalation without deleting anything.
func TestDeleteProcessInstanceDryRun_KeyedChildEscalatesToRootWithoutMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	sink := &activitysink.Sink{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during delete dry run")
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
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"child-1"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, typex.Keys{"root-1"}, typex.Keys(got.DryRunPreview.ResolvedRoots))
	require.Equal(t, typex.Keys{"root-1", "child-1"}, typex.Keys(got.DryRunPreview.AffectedFamilyKeys))
	require.Contains(t, buf.String(), "dry run: delete process-instance")
	require.NotContains(t, buf.String(), "selected process-instance keys")
	require.NotContains(t, buf.String(), "root process-instance tree keys")
	require.NotContains(t, buf.String(), "in-scope process-instance keys")
	require.NotContains(t, buf.String(), "no mutation submitted")
	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"preparing delete dry-run scope for 1 process instance(s)"}, msgs)
}

// TestDeleteProcessInstanceDryRun_KeyedRootReportsFullFamilyWithoutMutation
// verifies root selection previews the full family without mutation.
func TestDeleteProcessInstanceDryRun_KeyedRootReportsFullFamilyWithoutMutation(t *testing.T) {
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
		t.Fatal("unexpected confirmation prompt during delete dry run")
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
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"root-1"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 3, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, typex.Keys{"root-1"}, typex.Keys(got.DryRunPreview.ResolvedRoots))
	require.Equal(t, typex.Keys{"root-1", "child-1", "child-2"}, typex.Keys(got.DryRunPreview.AffectedFamilyKeys))
	require.Contains(t, buf.String(), "process instances in scope: 3")
	require.Contains(t, buf.String(), "process-instance family scope: complete (all related process instances were found)")
	require.NotContains(t, buf.String(), "in-scope process-instance keys")
	require.NotContains(t, buf.String(), "no mutation submitted")
}

// TestDeleteProcessInstanceCommand_DuplicateStdinKeysDeduplicateBeforePlanning
// verifies duplicate flag/stdin inputs are collapsed before command-local dry-run
// planning counts are rendered.
func TestDeleteProcessInstanceCommand_DuplicateStdinKeysDeduplicateBeforePlanning(t *testing.T) {
	const (
		firstKey  = "2251799813711967"
		secondKey = "2251799813711968"
	)

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			switch key {
			case firstKey, secondKey:
				_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
			default:
				t.Fatalf("unexpected process-instance lookup key %s", key)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = io.Copy(io.Discard, r.Body)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output := executeRootForProcessInstanceTestWithStdin(t,
		firstKey+"\n"+secondKey+"\n"+secondKey+"\n",
		"--config", cfgPath,
		"delete", "pi",
		"--key", firstKey,
		"-",
		"--dry-run",
		"--no-state-check",
	)

	require.Contains(t, output, "selected process instances: 2")
	require.NotContains(t, output, "selected process instances: 4")
}

func TestDeleteProcessInstanceStdinPipelineKeysSkipBpmnSelectorValidation(t *testing.T) {
	const key = "2251799813711967"
	var requests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+key:
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = io.Copy(io.Discard, r.Body)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case r.URL.Path == "/v2/process-definitions/search":
			t.Fatal("downstream stdin-key delete must not validate a BPMN selector")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output := executeRootForProcessInstanceTestWithStdin(t,
		key+"\n",
		"--config", cfgPath,
		"delete", "pi",
		"-",
		"--dry-run",
		"--no-state-check",
	)

	require.NotContains(t, requests, "POST /v2/process-definitions/search")
	require.Contains(t, requests, "POST /v2/process-instances/search")
	require.Contains(t, output, "selected process instances: 1")
}

// TestDeleteProcessInstanceDryRun_PartialOrphanParentRendersWarningAndMissingAncestor
// verifies partial ancestry details are surfaced in output.
func TestDeleteProcessInstanceDryRun_PartialOrphanParentRendersWarningAndMissingAncestor(t *testing.T) {
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
		t.Fatal("unexpected confirmation prompt during delete dry-run orphan preview")
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
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"child-orphan"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, process.TraversalOutcomePartial, got.DryRunPreview.TraversalOutcome)
	require.False(t, got.DryRunPreview.ScopeComplete)
	require.Equal(t, []processInstanceDryRunMissingAncestor{{Key: "missing-parent", StartKey: "child-orphan"}}, got.DryRunPreview.MissingAncestors)
	require.Contains(t, buf.String(), "scope: partial (one or more parent process instances were not found; missing ancestor keys: 1; use --verbose to list keys)")
	require.NotContains(t, buf.String(), "warning:")
	require.NotContains(t, buf.String(), "missing ancestor keys: missing-parent")
	require.NotContains(t, buf.String(), "no mutation submitted")
}

// TestDeleteProcessInstanceDryRun_UnresolvedOrphanFailsWithoutMutation verifies
// unresolved dry-run planning stops before mutation.
func TestDeleteProcessInstanceDryRun_UnresolvedOrphanFailsWithoutMutation(t *testing.T) {
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
		t.Fatal("unexpected confirmation prompt during unresolved delete dry run")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"unresolved-child"}, keys)
			return process.DryRunPIKeyExpansion{}, fmt.Errorf("%w: no process instances resolved during dependency expansion", services.ErrOrphanedInstance)
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"unresolved-child"}, true)

	require.Error(t, err)
	require.ErrorContains(t, err, "delete validation")
	require.ErrorContains(t, err, "no process instances resolved during dependency expansion")
	require.Equal(t, processInstancePageActionResult{}, got)
	require.Empty(t, buf.String())
}

// TestDeleteProcessInstanceDryRun_KeyTenantMismatchUsesAdminScope keeps direct
// delete previews backend-authorized while preserving dependency planning.
func TestDeleteProcessInstanceDryRun_KeyTenantMismatchUsesAdminScope(t *testing.T) {
	var requests testx.SafeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		requests.Append(r.Method + " " + r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+tenantAdminKeysProcessInstanceKey:
			_, _ = w.Write([]byte(tenantAdminKeysMismatchProcessInstanceJSON()))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, ok := searchBody["filter"].(map[string]any)
			require.True(t, ok, "expected search request filter object")
			require.Equal(t, tenantAdminKeysProcessInstanceKey, filter["parentProcessInstanceKey"])
			require.NotContains(t, filter, "tenantId")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", tenantAdminKeysSelectedTenant,
		"--json",
		"delete", "process-instance",
		"--key", tenantAdminKeysProcessInstanceKey,
		"--dry-run",
	)

	require.Contains(t, requests.Snapshot(), "GET /v2/process-instances/"+tenantAdminKeysProcessInstanceKey)
	require.Contains(t, requests.Snapshot(), "POST /v2/process-instances/search")
	require.Contains(t, output, `"requestedCount": 1`)
	require.Contains(t, output, tenantAdminKeysProcessInstanceKey)
}

// Verifies date filters cannot be combined with direct key lookup mode.
func TestDeleteProcessInstanceCommand_RejectsKeyAndDateFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsKeyAndDateFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
}

// Verifies relative-day filters cannot be combined with direct key lookup mode.
func TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
}

// TestDeleteProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags verifies paging flag validation errors.
func TestDeleteProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "removed count flag is rejected",
			helper: "TestDeleteProcessInstanceCommand_RejectsRemovedCountFlagHelper",
			want:   "unknown flag: --count",
		},
		{
			name:   "non-positive limit is rejected",
			helper: "TestDeleteProcessInstanceCommand_RejectsInvalidLimitHelper",
			want:   "--limit must be positive integer",
		},
		{
			name:   "limit cannot be combined with key",
			helper: "TestDeleteProcessInstanceCommand_RejectsLimitWithKeyHelper",
			want:   "--limit cannot be combined with --key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeDeleteProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestDeleteProcessInstanceCommand_V89DeletesViaCamundaProcessInstanceAPI verifies v8.9 deletion uses the native API.
func TestDeleteProcessInstanceCommand_V89DeletesViaCamundaProcessInstanceAPI(t *testing.T) {
	var requests []string
	var deleted []string

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
			switch {
			case parentKey == "2251799813711967":
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/2251799813711967/deletion":
			deleted = append(deleted, r.URL.Path)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"delete", "process-instance",
		"--key", "2251799813711967",
		"--no-wait",
	)

	require.GreaterOrEqual(t, len(requests), 1)
	require.Equal(t, []string{"/v2/process-instances/2251799813711967/deletion"}, deleted)
	require.Contains(t, requests[len(requests)-1], `"parentProcessInstanceKey":"2251799813711967"`)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Equal(t, "delete process-instance", got["command"])
	require.NotContains(t, stderr, "INFO")
	require.NotContains(t, stderr, "delete requested")
	require.NotContains(t, stderr, "deleting process instances")
}

// TestDeleteProcessInstancesWithPlan_PrintsOrphanWarningForKeyedImpactCheck verifies keyed impact-check warnings are printed.
func TestDeleteProcessInstancesWithPlan_PrintsOrphanWarningForKeyedImpactCheck(t *testing.T) {
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
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			require.Equal(t, typex.Keys{"2251799813711900"}, keys)
			require.Zero(t, wantedWorkers)
			cfg := options.ApplyFacadeOptions(opts)
			require.Equal(t, 2, cfg.AffectedProcessInstanceCount)
			require.True(t, cfg.SuppressWorkflowDetailLogs)
			require.True(t, cfg.SuppressProcessInstanceDetailLogs)
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "2251799813711900", Ok: true}}}, nil
		},
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"2251799813711967"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.Contains(t, prompt, "requested to delete 1 process instance(s)")
	require.Contains(t, prompt, "a total of 2 instance(s) with 1 root instance(s) will be deleted")
	require.Contains(t, buf.String(), "one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: 1 (use --verbose to list keys)")
	require.Contains(t, buf.String(), "deletion: deleted 1/1 process-instance tree(s); affected process instances: 2")
	require.NotContains(t, buf.String(), "missing ancestor keys: 2251799813711999")
}

// TestDeleteProcessInstancesWithPlan_SubmitsResolvedRootsOnlyForKeyedHierarchy
// protects direct-key delete planning from regressing to deleting selected child
// keys directly.
func TestDeleteProcessInstancesWithPlan_SubmitsResolvedRootsOnlyForKeyedHierarchy(t *testing.T) {
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
			require.Equal(t, typex.Keys{"child-a", "child-b"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-a"},
				Collected: typex.Keys{"root-a", "child-a", "child-b"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			require.Equal(t, typex.Keys{"root-a"}, keys)
			require.Zero(t, wantedWorkers)
			cfg := options.ApplyFacadeOptions(opts)
			require.Equal(t, 3, cfg.AffectedProcessInstanceCount)
			require.True(t, cfg.SuppressWorkflowDetailLogs)
			require.True(t, cfg.SuppressProcessInstanceDetailLogs)
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "root-a", Ok: true}}}, nil
		},
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"child-a", "child-b"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 2, Affected: 3, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, typex.Keys{"root-a"}, typex.Keys(got.DryRunPreview.ResolvedRoots))
	require.Equal(t, typex.Keys{"root-a", "child-a", "child-b"}, typex.Keys(got.DryRunPreview.AffectedFamilyKeys))
	require.Contains(t, prompt, "requested to delete 2 process instance(s)")
	require.Contains(t, prompt, "a total of 3 instance(s) with 1 root instance(s) will be deleted")
	require.Contains(t, buf.String(), "deletion: deleted 1/1 process-instance tree(s); affected process instances: 3")
}

// TestDeleteProcessInstancesWithPlan_RegressionForceNoWaitAndWorkerControls
// protects delete process-instance hierarchy planning and execution controls while incident
// purge delegates to this path.
func TestDeleteProcessInstancesWithPlan_RegressionForceNoWaitAndWorkerControls(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagForce = true
	flagNoWait = true
	flagFailFast = true
	flagNoWorkerLimit = true
	flagWorkers = 3

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.True(t, autoConfirm)
		require.Contains(t, prompt, "requested to delete 2 process instance(s)")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"child-a", "child-b"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-a"},
				Collected: typex.Keys{"root-a", "child-a", "child-b"},
				RequiresCancelBeforeDelete: []process.ProcessInstance{
					{Key: "child-b", State: process.StateActive},
				},
				Outcome: process.TraversalOutcomeComplete,
			}, nil
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			require.Equal(t, typex.Keys{"root-a"}, keys)
			require.Equal(t, 3, wantedWorkers)
			applied := options.ApplyFacadeOptions(opts)
			require.True(t, applied.Force)
			require.True(t, applied.NoWait)
			require.True(t, applied.FailFast)
			require.True(t, applied.NoWorkerLimit)
			require.Equal(t, 3, applied.AffectedProcessInstanceCount)
			require.True(t, applied.SuppressWorkflowDetailLogs)
			require.True(t, applied.SuppressProcessInstanceDetailLogs)
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "root-a", Ok: true}}}, nil
		},
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"child-a", "child-b"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 2, Affected: 3, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, typex.Keys{"root-a"}, typex.Keys(got.DryRunPreview.ResolvedRoots))
}

// TestDeleteProcessInstancesWithPlan_RequiresForceBeforeAnyMutation verifies
// delete is all-or-nothing when the expanded scope includes non-final instances.
func TestDeleteProcessInstancesWithPlan_RequiresForceBeforeAnyMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(_ bool, _ string) error {
		t.Fatal("unexpected confirmation prompt before force impact-check failure")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"6755399442315265"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"6755399442312837"},
				Collected: typex.Keys{"6755399442312837", "6755399442312860", "6755399442313287", "6755399442315265"},
				RequiresCancelBeforeDelete: []process.ProcessInstance{
					{Key: "6755399442312837", State: process.StateActive},
					{Key: "6755399442312860", State: process.StateActive},
					{Key: "6755399442315265", State: process.StateActive},
				},
				Outcome: process.TraversalOutcomeComplete,
			}, nil
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"6755399442315265"}, true)

	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to delete process-instance scope")
	require.Contains(t, err.Error(), "no delete request was submitted")
	require.Empty(t, got.Reports)
	require.NotContains(t, buf.String(), "waiting for process instance")
}

// Verifies direct --key deletion bypasses top-level search pagination logic.
func TestDeleteProcessInstanceCommand_DirectKeyBypassesTopLevelSearchPaging(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))
			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			w.Header().Set("Content-Type", "application/json")
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
			}
			_, _ = w.Write([]byte(`{"items":[]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/601":
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/601":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"601","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
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
		"delete", "process-instance",
		"--key", "601",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Empty(t, pages)
	require.Equal(t, []string{"/v1/process-instances/601"}, deleted.Snapshot())
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.NotContains(t, stderr, "INFO")
	require.NotContains(t, stderr, "delete requested")
	require.NotContains(t, stderr, "deleting process instances")
}

// TestDeleteProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetail verifies direct-key failures keep root detail.
func TestDeleteProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetail(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/601", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"resource not found"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetailHelper", cfgPath)

	require.Equal(t, exitcode.NotFound, code)
	require.Contains(t, output, "resource not found")
	require.Contains(t, output, "delete validation")
	require.Contains(t, output, "ancestry")
	require.NotContains(t, output, "validating process instance keys for cancellation")
	require.NotContains(t, output, "ancestry get")
	require.Contains(t, output, "get process instance")
	require.Less(t, strings.Index(output, "delete validation"), strings.Index(output, "ancestry"))
	require.Less(t, strings.Index(output, "ancestry"), strings.Index(output, "get process instance"))
	require.NotContains(t, output, "fetching process instance with key")
}

// TestDeleteProcessInstanceCommand_RejectsRemovedCountFlagHelper is the helper-process entrypoint for removed --count validation.
func TestDeleteProcessInstanceCommand_RejectsRemovedCountFlagHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--count", "2"}

	Execute()
}

// TestDeleteProcessInstanceCommand_RejectsInvalidLimitHelper is the helper-process entrypoint for invalid --limit validation.
func TestDeleteProcessInstanceCommand_RejectsInvalidLimitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--limit", "0"}

	Execute()
}

// TestDeleteProcessInstanceCommand_RejectsLimitWithKeyHelper is the helper-process entrypoint for --limit with --key validation.
func TestDeleteProcessInstanceCommand_RejectsLimitWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "123", "--limit", "1"}

	Execute()
}

// TestDeleteProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetailHelper is the helper-process entrypoint for direct-key failure detail.
func TestDeleteProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetailHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "delete", "process-instance", "--key", "601", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for key-and-date-filter exclusivity validation.
func TestDeleteProcessInstanceCommand_RejectsKeyAndDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}

// Helper-process entrypoint for key-and-relative-day-filter exclusivity validation.
func TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "2251799813711967", "--end-date-newer-days", "7"}

	Execute()
}
