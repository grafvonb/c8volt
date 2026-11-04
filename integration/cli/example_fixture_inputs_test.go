// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Supply only the documented sample vars file, after parsing shell arguments.
// Other paths remain unchanged so genuine input errors are not hidden.
func prepareExampleVarsFile(args []string, workDir, marker string) ([]string, []string, error) {
	result := append([]string(nil), args...)
	path := ""
	var substitutions []string
	for i, arg := range args {
		name, value, inline := strings.Cut(arg, "=")
		if name != "--set-vars-file" {
			continue
		}
		if !inline && i+1 < len(args) {
			value = args[i+1]
		}
		if value != "./vars.json" && value != "vars.json" {
			continue
		}
		if path == "" {
			absDir, err := filepath.Abs(workDir)
			if err != nil {
				return nil, nil, err
			}
			file, err := os.CreateTemp(absDir, "example-vars-*.json")
			if err != nil {
				return nil, nil, err
			}
			path = file.Name()
			err = json.NewEncoder(file).Encode(map[string]any{"c8voltITRunId": marker, "exampleScenario": true})
			closeErr := file.Close()
			if err != nil {
				return nil, nil, err
			}
			if closeErr != nil {
				return nil, nil, closeErr
			}
			substitutions = append(substitutions, "--set-vars-file "+value+"="+path)
		}
		if inline {
			result[i] = "--set-vars-file=" + path
		} else {
			result[i+1] = path
		}
	}
	return result, substitutions, nil
}

// unresolvedPlaceholdersAllowed records prerequisites that dedicated family
// slices cover beyond the generic held-task example seed.
func unresolvedPlaceholdersAllowed(placeholders []string) bool {
	// Child selection is exercised by the cascade/lifecycle family fixtures.
	allowed := map[string]struct{}{
		"<job-key>":              {},
		"<child-instance-key>":   {},
		"<incident-key>":         {},
		"<another-incident-key>": {},
		"<resource-id>":          {},
		"<resource-key>":         {},
		"<element-instance-key>": {},
		"<element-id>":           {},
		"<user-task-key>":        {},
		"<tenant-id>":            {},
	}
	for _, placeholder := range placeholders {
		if _, ok := allowed[placeholder]; !ok {
			return false
		}
	}
	return len(placeholders) > 0
}

// Canonical variable-file examples receive isolated generated payload files.
func TestExampleVarsFileFixture(t *testing.T) {
	for _, command := range []string{"update process-instance", "update ut"} {
		for _, inline := range []bool{false, true} {
			t.Run(command+"/inline="+map[bool]string{false: "false", true: "true"}[inline], func(t *testing.T) {
				dir := filepath.Join(t.TempDir(), "evidence with spaces")
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				original := append(strings.Fields(command), "--key", "123", "--dry-run")
				if inline {
					original = append(original, "--set-vars-file=./vars.json")
				} else {
					original = append(original, "--set-vars-file", "./vars.json")
				}
				snapshot := append([]string(nil), original...)
				args, substitutions, err := prepareExampleVarsFile(original, dir, "test-marker")
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(original, snapshot) {
					t.Fatal("input arguments mutated")
				}
				path := strings.TrimPrefix(args[len(args)-1], "--set-vars-file=")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var vars map[string]any
				if err := json.Unmarshal(data, &vars); err != nil {
					t.Fatal(err)
				}
				if vars["c8voltITRunId"] != "test-marker" || vars["exampleScenario"] != true {
					t.Fatalf("wrong vars: %v", vars)
				}
				if filepath.Dir(path) != dir || len(substitutions) != 1 {
					t.Fatalf("missing owned fixture evidence: %v %v", args, substitutions)
				}
			})
		}
	}
}

// Fixture substitution preserves unrelated arguments and explicit file choices.
func TestExampleVarsFilePreservesOtherInputs(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"get", "pd"}, {"--set-vars-file", "custom.json"}, {"--set-vars-file"}} {
		result, substitutions, err := prepareExampleVarsFile(args, dir, "marker")
		if err != nil || !reflect.DeepEqual(result, args) || len(substitutions) != 0 {
			t.Fatalf("unexpected rewrite: %v %v %v", result, substitutions, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unexpected fixture files: %v %v", entries, err)
	}
	if _, _, err := prepareExampleVarsFile([]string{"--set-vars-file", "./vars.json"}, filepath.Join(dir, "missing"), "marker"); err == nil {
		t.Fatal("fixture write failure swallowed")
	}
}

func TestExampleMissingResourceKeyIsSetupGap(t *testing.T) {
	if !unresolvedPlaceholdersAllowed([]string{"<resource-key>"}) || !unresolvedPlaceholdersAllowed([]string{"<resource-id>"}) {
		t.Fatal("missing resource fixture must remain a recorded setup gap")
	}
	if unresolvedPlaceholdersAllowed(nil) || unresolvedPlaceholdersAllowed([]string{"<resource-key>", "<unexpected>"}) {
		t.Fatal("unrelated unresolved input incorrectly allowlisted")
	}
}

// Relative evidence directories remain valid for canonical file inputs.
func TestExampleVarsFileRelativeEvidenceDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("evidence", 0700); err != nil {
		t.Fatal(err)
	}
	args, _, err := prepareExampleVarsFile([]string{"--set-vars-file", "vars.json"}, "evidence", "marker")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(args[1]) {
		t.Fatalf("fixture path depends on subprocess working directory: %q", args[1])
	}
	if _, err := os.ReadFile(args[1]); err != nil {
		t.Fatal(err)
	}
}
