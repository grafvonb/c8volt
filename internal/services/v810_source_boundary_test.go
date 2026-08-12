// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package services_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var v810ServiceFamilies = []string{
	"batchoperation",
	"cluster",
	"element",
	"incident",
	"job",
	"processdefinition",
	"processinstance",
	"resource",
	"tenant",
	"usertask",
	"variable",
}

// TestV810AdapterSourceBoundary rejects cross-version generated clients,
// older service adapters, and removed component clients as soon as a v810
// adapter package appears.
func TestV810AdapterSourceBoundary(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	var scanned []string
	var violations []string
	for _, family := range v810ServiceFamilies {
		dir := filepath.Join(root, "internal", "services", family, "v810")
		if _, err := os.Stat(dir); err != nil {
			require.ErrorIs(t, err, os.ErrNotExist)
			continue
		}

		files := goSourceFiles(t, dir)
		require.NotEmptyf(t, files, "v810 adapter directory exists without Go source: %s", dir)
		for _, file := range files {
			scanned = append(scanned, relativePath(t, root, file))
			violations = append(violations, v810AdapterImportViolations(t, root, file)...)
		}
	}

	sort.Strings(scanned)
	sort.Strings(violations)
	require.Empty(t, violations, "v810 adapter imports must stay native to v810 unified clients")
}

// TestCommandAndFacadeSourceBoundaryForGeneratedClients keeps generated
// Camunda clients and versioned service implementations behind public facades
// and internal factories.
func TestCommandAndFacadeSourceBoundaryForGeneratedClients(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	scanRoots := []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "c8volt"),
	}

	var violations []string
	for _, scanRoot := range scanRoots {
		for _, file := range goSourceFiles(t, scanRoot) {
			violations = append(violations, commandFacadeImportViolations(t, root, file)...)
		}
	}

	sort.Strings(violations)
	require.Empty(t, violations, "cmd and public facades must not import generated clients or versioned services")
}

func v810AdapterImportViolations(t *testing.T, root, file string) []string {
	t.Helper()

	imports := parseImports(t, file)
	var violations []string
	for _, importPath := range imports {
		if strings.Contains(importPath, "/internal/clients/camunda/") &&
			importPath != "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda" {
			violations = append(violations, relativePath(t, root, file)+":import:"+importPath)
			continue
		}
		if strings.Contains(importPath, "/internal/services/") && strings.Contains(importPath, "/v8") && !allowedV810ServiceImport(root, file, importPath) {
			violations = append(violations, relativePath(t, root, file)+":import:"+importPath)
		}
	}
	return violations
}

func allowedV810ServiceImport(root, file, importPath string) bool {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return false
	}
	return filepath.ToSlash(rel) == "internal/services/processinstance/v810/service.go" &&
		importPath == "github.com/grafvonb/c8volt/internal/services/variable/v810"
}

func commandFacadeImportViolations(t *testing.T, root, file string) []string {
	t.Helper()

	imports := parseImports(t, file)
	var violations []string
	for _, importPath := range imports {
		switch {
		case strings.Contains(importPath, "/internal/clients/camunda/"):
			violations = append(violations, relativePath(t, root, file)+":import:"+importPath)
		case strings.Contains(importPath, "/internal/services/") && strings.Contains(importPath, "/v8"):
			violations = append(violations, relativePath(t, root, file)+":import:"+importPath)
		}
	}
	return violations
}

func parseImports(t *testing.T, path string) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	require.NoError(t, err)

	imports := make([]string, 0, len(file.Imports))
	for _, imported := range file.Imports {
		imports = append(imports, strings.Trim(imported.Path.Value, `"`))
	}
	sort.Strings(imports)
	return imports
}

func goSourceFiles(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		require.NoError(t, err)
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)

	sort.Strings(files)
	return files
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "could not find repository root from working directory")
		dir = parent
	}
}

func relativePath(t *testing.T, root, path string) string {
	t.Helper()

	rel, err := filepath.Rel(root, path)
	require.NoError(t, err)
	return filepath.ToSlash(rel)
}
