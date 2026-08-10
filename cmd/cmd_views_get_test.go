// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestGetViewFilesAvoidBackendOwnership keeps renderer files free of backend
// orchestration, with the known dry-run planning exception tracked for US3.
func TestGetViewFilesAvoidBackendOwnership(t *testing.T) {
	t.Parallel()

	violations := collectViewBackendOwnershipViolations(t)

	require.Empty(t, violations, "renderer files should not call public facades or import internal services")
}

// Protects the shared flat-list contract: align from observed values, but preserve every character.
func TestFormatFlatRows_AlignsColumnsWithoutTruncating(t *testing.T) {
	got := formatFlatRows([]flatRow{
		{"1", "tenant", "Short", "v1"},
		{"22", "t", "MuchLongerProcess", "v12"},
	})

	require.Equal(t, []string{
		"1  tenant Short             v1",
		"22 t      MuchLongerProcess v12",
	}, got)
}

var allowedViewFacadeCalls = map[string]string{
	"cmd_views_processinstance_dryrun.go:planProcessInstanceDryRunPreviewWithOptions:cli.DryRunCancelOrDeletePlan": "US3 T041 moves dry-run planning out of renderer ownership",
}

// collectViewBackendOwnershipViolations scans renderer source without loading
// the package so ownership regressions are caught even before type checking.
func collectViewBackendOwnershipViolations(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob("cmd_views_*.go")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	var violations []string
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		violations = append(violations, viewBackendOwnershipViolationsForFile(t, path)...)
	}
	sort.Strings(violations)
	return violations
}

// viewBackendOwnershipViolationsForFile returns backend ownership markers from
// one renderer file while preserving the existing dry-run planning baseline.
func viewBackendOwnershipViolationsForFile(t *testing.T, path string) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	require.NoError(t, err)

	imports := viewFileImports(file)
	publicFacadeAliases := publicFacadeImportAliases(imports)
	var violations []string
	for _, importPath := range imports {
		if strings.HasPrefix(importPath, "github.com/grafvonb/c8volt/internal/services") {
			violations = append(violations, path+":import:"+importPath)
		}
	}

	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		facadeParamNames := facadeAPIParamNames(function, publicFacadeAliases)
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				if fun.Name == "NewCli" {
					violations = appendViolationIfNotAllowed(violations, path, function.Name.Name, fun.Name)
				}
			case *ast.SelectorExpr:
				receiver, ok := fun.X.(*ast.Ident)
				if ok {
					if _, isFacadeParam := facadeParamNames[receiver.Name]; isFacadeParam {
						violations = appendViolationIfNotAllowed(violations, path, function.Name.Name, receiver.Name+"."+fun.Sel.Name)
					}
				}
			}
			return true
		})
	}

	return violations
}

// appendViolationIfNotAllowed keeps the known dry-run planning violation from
// blocking foundational checks while still failing on any new renderer call.
func appendViolationIfNotAllowed(violations []string, path, functionName, callName string) []string {
	violation := path + ":" + functionName + ":" + callName
	if _, ok := allowedViewFacadeCalls[violation]; ok {
		return violations
	}
	return append(violations, violation)
}

// viewFileImports maps the effective package name to the imported path.
func viewFileImports(file *ast.File) map[string]string {
	imports := make(map[string]string, len(file.Imports))
	for _, imported := range file.Imports {
		importPath := strings.Trim(imported.Path.Value, `"`)
		name := filepath.Base(importPath)
		if imported.Name != nil {
			name = imported.Name.Name
		}
		imports[name] = importPath
	}
	return imports
}

// publicFacadeImportAliases returns c8volt facade packages that may expose an API.
func publicFacadeImportAliases(imports map[string]string) map[string]struct{} {
	aliases := make(map[string]struct{})
	for name, importPath := range imports {
		if strings.HasPrefix(importPath, "github.com/grafvonb/c8volt/c8volt/") &&
			importPath != "github.com/grafvonb/c8volt/c8volt/ferrors" &&
			importPath != "github.com/grafvonb/c8volt/c8volt/foptions" {
			aliases[name] = struct{}{}
		}
	}
	return aliases
}

// facadeAPIParamNames identifies function parameters typed as public facade APIs.
func facadeAPIParamNames(function *ast.FuncDecl, publicFacadeAliases map[string]struct{}) map[string]struct{} {
	names := make(map[string]struct{})
	if function.Type.Params == nil {
		return names
	}
	for _, field := range function.Type.Params.List {
		selector, ok := field.Type.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "API" {
			continue
		}
		alias, ok := selector.X.(*ast.Ident)
		if !ok {
			continue
		}
		if _, isPublicFacade := publicFacadeAliases[alias.Name]; !isPublicFacade {
			continue
		}
		for _, name := range field.Names {
			names[name.Name] = struct{}{}
		}
	}
	return names
}

// Verifies process-definition scan output uses the same dynamic alignment as process-instance lists.
func TestListProcessDefinitionsView_AlignsFlatRowsDynamically(t *testing.T) {
	cmd := &cobra.Command{Use: "process-definition"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := listProcessDefinitionsView(cmd, process.ProcessDefinitions{
		Items: []process.ProcessDefinition{
			{
				Key:            "1",
				TenantId:       "tenant",
				BpmnProcessId:  "Short",
				ProcessVersion: 1,
			},
			{
				Key:               "22",
				TenantId:          "tenant",
				BpmnProcessId:     "MuchLongerDefinition",
				ProcessVersion:    12,
				ProcessVersionTag: "stable",
				Statistics: &process.ProcessDefinitionStatistics{
					Active:                 4,
					Completed:              9,
					Canceled:               2,
					Incidents:              3,
					IncidentCountSupported: true,
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, ""+
		"1  tenant Short                v1\n"+
		"22 tenant MuchLongerDefinition v12/stable [ac:4 cp:9 cx:2 inc:3]\n"+
		"found: 2\n", buf.String())
}

func TestIncidentHumanLineWithMessageLimit_RendersAlignedIncidentListFieldsAndAge(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	lines := formatIncidentListRows([]incident.ProcessInstanceIncidentDetail{
		{
			IncidentKey:            "2251799813685249",
			TenantId:               "tenant-a",
			State:                  "ACTIVE",
			ErrorType:              "JOB_NO_RETRIES",
			ErrorMessage:           "No retries left for a long-running job",
			CreationTime:           "2026-05-05T10:15:00Z",
			ProcessInstanceKey:     "2251799813711967",
			RootProcessInstanceKey: "2251799813711960",
			ProcessDefinitionKey:   "2251799813685200",
			ProcessDefinitionId:    "demo-process",
			ElementId:              "task-a",
			ElementInstanceKey:     "2251799813685300",
		},
		{
			IncidentKey:         "9",
			TenantId:            "<default>",
			State:               "RESOLVED",
			ErrorType:           "IO_MAPPING_ERROR",
			ErrorMessage:        "short",
			CreationTime:        "2026-05-08T10:15:00Z",
			ProcessInstanceKey:  "1",
			ProcessDefinitionId: "tiny-demo",
		},
	}, 15, false)

	require.Len(t, lines, 2)
	require.Contains(t, lines[0], "2251799813685249 tenant-a  JOB_NO_RETRIES   ACTIVE   j:n/a")
	require.Contains(t, lines[0], "j:n/a 2026-05-05T10:15:00.000 (4 days ago) demo-process pi:2251799813711967 root:2251799813711960")
	require.Contains(t, lines[0], "e:task-a ei:2251799813685300 m:No retries left...")
	require.NotContains(t, lines[0], "fn:")
	require.NotContains(t, lines[0], "fni:")
	require.Contains(t, lines[1], "9                <default> IO_MAPPING_ERROR RESOLVED j:n/a")
	require.Contains(t, lines[1], "j:n/a 2026-05-08T10:15:00.000 (1 days ago) tiny-demo    pi:1")
	require.Contains(t, lines[1], "m:short")
	require.Less(t, strings.Index(lines[0], "ACTIVE"), strings.Index(lines[0], "j:n/a"))
	require.Less(t, strings.Index(lines[0], "j:n/a"), strings.Index(lines[0], "2026-05-05T10:15:00.000"))
	require.Less(t, strings.Index(lines[0], "2026-05-05T10:15:00.000"), strings.Index(lines[0], "demo-process"))
	require.Less(t, strings.Index(lines[0], "demo-process"), strings.Index(lines[0], "pi:2251799813711967"))
	require.Less(t, strings.Index(lines[0], "pi:2251799813711967"), strings.Index(lines[0], "root:2251799813711960"))
	require.Less(t, strings.Index(lines[0], "root:2251799813711960"), strings.Index(lines[0], "e:task-a"))
	require.Less(t, strings.Index(lines[0], "e:task-a"), strings.Index(lines[0], "ei:2251799813685300"))
	require.Less(t, strings.Index(lines[0], "ei:2251799813685300"), strings.Index(lines[0], "m:No"))
	require.Equal(t, strings.Index(lines[0], "2026-05-05T10:15:00.000"), strings.Index(lines[1], "2026-05-08T10:15:00.000"))
	require.NotContains(t, lines[0], "2251799813685200")
	require.NotContains(t, lines[0], "err:")
	require.Contains(t, lines[0], "m:No")
	require.NotContains(t, lines[0], "m: ")
}

func TestIncidentHumanLineWithMessageLimit_SkipsAgeForMissingOrInvalidCreationTime(t *testing.T) {
	line := incidentListHumanLineWithMessageLimit(incident.ProcessInstanceIncidentDetail{
		IncidentKey:  "2251799813685249",
		CreationTime: "not-a-date",
		ErrorMessage: "failed",
	}, 0)

	require.Contains(t, line, "not-a-date")
	require.NotContains(t, line, "days ago")
	require.NotContains(t, line, "(today)")
}

func TestTruncateIncidentHumanMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		limit   int
		want    string
	}{
		{
			name:    "unlimited",
			message: "No retries left",
			limit:   0,
			want:    "No retries left",
		},
		{
			name:    "exact limit",
			message: "No retries left",
			limit:   15,
			want:    "No retries left",
		},
		{
			name:    "truncated",
			message: "No retries left",
			limit:   2,
			want:    "No...",
		},
		{
			name:    "multi-byte",
			message: "äöü failed",
			limit:   2,
			want:    "äö...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, truncateIncidentHumanMessage(tt.message, tt.limit))
		})
	}
}

func TestListIncidentsView_HumanJSONAndKeysOnly(t *testing.T) {
	resp := incident.Incidents{
		Total: 2,
		Items: []incident.ProcessInstanceIncidentDetail{
			{
				IncidentKey:        "incident-123",
				CreationTime:       "2026-05-06T09:29:42.711Z",
				ProcessInstanceKey: "pi-123",
				TenantId:           "tenant-a",
				State:              "ACTIVE",
				ErrorType:          "JOB_NO_RETRIES",
				ErrorMessage:       "No retries left",
				ElementId:          "task-a",
				ElementInstanceKey: "element-123",
				JobKey:             "job-123",
			},
			{
				IncidentKey:  "incident-124",
				State:        "RESOLVED",
				ErrorMessage: "Mapping failed",
			},
		},
	}

	t.Run("human", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("incident")

		require.NoError(t, listIncidentsView(cmd, resp, 10, false))
		output := cmd.OutOrStdout().(*bytes.Buffer).String()

		require.Contains(t, output, "incident-123")
		require.Contains(t, output, "2026-05-06T09:29:42.711")
		require.Contains(t, output, "e:task-a")
		require.Contains(t, output, "JOB_NO_RETRIES")
		require.Contains(t, output, "j:job-123")
		require.Contains(t, output, "m:No retries...")
		require.Contains(t, output, "incident-124")
		require.Contains(t, output, "j:n/a")
		require.Contains(t, output, "found: 2")
	})

	t.Run("default without messages", func(t *testing.T) {
		resetViewModeFlags(t)
		cmd := newGetViewTestCommand("incident")

		require.NoError(t, listIncidentsView(cmd, resp, 10, true))
		output := cmd.OutOrStdout().(*bytes.Buffer).String()

		require.Contains(t, output, "incident-123")
		require.NotContains(t, output, "m:")
		require.NotContains(t, output, "No retries")
		require.Contains(t, output, "found: 2")
	})

	t.Run("json", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewAsJson = true
		cmd := newGetViewTestCommand("incident")
		setContractSupport(cmd, ContractSupportFull)

		require.NoError(t, listIncidentsView(cmd, resp, 4, false))
		output := cmd.OutOrStdout().(*bytes.Buffer).String()

		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &envelope))
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		payload := requireJSONObject(t, envelope["payload"])
		items, ok := payload["items"].([]any)
		require.True(t, ok)
		first := requireJSONObject(t, items[0])
		require.Equal(t, "No retries left", first["errorMessage"])
		require.Equal(t, "2026-05-06T09:29:42.711Z", first["creationTime"])
	})

	t.Run("keys only", func(t *testing.T) {
		resetViewModeFlags(t)
		flagViewKeysOnly = true
		cmd := newGetViewTestCommand("incident")

		require.NoError(t, listIncidentsView(cmd, resp, 0, false))

		require.Equal(t, "incident-123\nincident-124\n", cmd.OutOrStdout().(*bytes.Buffer).String())
	})
}

func TestListIncidentsView_PIKeysOnlySkipsMissingProcessInstanceKeys(t *testing.T) {
	resetViewModeFlags(t)
	flagGetIncidentPIKeysOnly = true
	cmd := newGetViewTestCommand("incident")

	err := listIncidentsView(cmd, incident.Incidents{
		Items: []incident.ProcessInstanceIncidentDetail{
			{IncidentKey: "incident-123", ProcessInstanceKey: "pi-123"},
			{IncidentKey: "incident-124"},
		},
	}, 0, false)

	require.NoError(t, err)
	require.Equal(t, "pi-123\n", cmd.OutOrStdout().(*bytes.Buffer).String())
}

func TestRenderIncidentProcessInstanceKeys_PreservesDuplicatesAndSkipsMissing(t *testing.T) {
	resetViewModeFlags(t)
	cmd := newGetViewTestCommand("incident")

	err := renderIncidentProcessInstanceKeys(cmd, []incident.ProcessInstanceIncidentDetail{
		{IncidentKey: "incident-123", ProcessInstanceKey: "pi-123"},
		{IncidentKey: "incident-124", ProcessInstanceKey: "pi-123"},
		{IncidentKey: "incident-125"},
		{IncidentKey: "incident-126", ProcessInstanceKey: "pi-126"},
	})

	require.NoError(t, err)
	require.Equal(t, "pi-123\npi-123\npi-126\n", cmd.OutOrStdout().(*bytes.Buffer).String())
}

func newGetViewTestCommand(use string) *cobra.Command {
	cmd := &cobra.Command{Use: use}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	parent := &cobra.Command{Use: "get"}
	parent.AddCommand(cmd)
	return cmd
}

func resetViewModeFlags(t *testing.T) {
	t.Helper()
	prevJSON := flagViewAsJson
	prevKeysOnly := flagViewKeysOnly
	prevPIKeysOnly := flagGetIncidentPIKeysOnly
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
		flagGetIncidentPIKeysOnly = prevPIKeysOnly
	})
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagGetIncidentPIKeysOnly = false
}
