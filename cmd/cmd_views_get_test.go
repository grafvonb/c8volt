// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"

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
