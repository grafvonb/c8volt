// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package camunda_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCamundaClientInventory prevents retired versions and unused component
// clients from returning, while requiring every client used by current adapters.
func TestCamundaClientInventory(t *testing.T) {
	t.Parallel()

	want := map[string][]string{
		"v88":  {"camunda", "operate", "tasklist"},
		"v89":  {"camunda", "tasklist"},
		"v810": {"camunda"},
	}
	versions, err := os.ReadDir(".")
	require.NoError(t, err)
	var gotVersions []string
	for _, version := range versions {
		if !version.IsDir() {
			continue
		}
		gotVersions = append(gotVersions, version.Name())
		t.Run(version.Name(), func(t *testing.T) {
			expected, supported := want[version.Name()]
			require.True(t, supported, "unsupported Camunda client directory: %s", version.Name())
			components, err := os.ReadDir(version.Name())
			require.NoError(t, err)
			var gotComponents []string
			for _, component := range components {
				if component.IsDir() {
					gotComponents = append(gotComponents, component.Name())
				}
			}
			require.ElementsMatch(t, expected, gotComponents, "only runtime-used clients belong in this version")
			for _, component := range expected {
				require.FileExists(t, filepath.Join(version.Name(), component, "client.gen.go"))
			}
		})
	}
	require.ElementsMatch(t, []string{"v88", "v89", "v810"}, gotVersions)
}

// TestCamundaClientGenerationTargets guards both generation calls and the
// --commit manifest without fetching specs or running code generation. V810
// remains owned by its separate pinned generator and provenance tests.
func TestCamundaClientGenerationTargets(t *testing.T) {
	t.Parallel()

	script, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "3-generate-clients-from-fetched-specs.sh"))
	require.NoError(t, err)
	want := []string{
		"internal/clients/auth/oauth2/client.gen.go",
		"internal/clients/camunda/v88/camunda/client.gen.go",
		"internal/clients/camunda/v88/operate/client.gen.go",
		"internal/clients/camunda/v88/tasklist/client.gen.go",
		"internal/clients/camunda/v89/camunda/client.gen.go",
		"internal/clients/camunda/v89/tasklist/client.gen.go",
	}
	manifest := regexp.MustCompile(`(?s)GENERATED_FILES=\((.*?)\n\)`).FindSubmatch(script)
	require.Len(t, manifest, 2, "generation must declare its commit manifest")
	paths := regexp.MustCompile(`internal/clients/[a-zA-Z0-9_/.-]+/client\.gen\.go`)
	t.Run("commit manifest", func(t *testing.T) {
		require.ElementsMatch(t, want, paths.FindAllString(string(manifest[1]), -1))
	})
	t.Run("generation calls", func(t *testing.T) {
		calls := regexp.MustCompile(`(?m)^\./generate-go-client\.sh .*$`).FindAllString(string(script), -1)
		var outputs []string
		for _, call := range calls {
			output := paths.FindAllString(call, -1)
			require.Len(t, output, 1, "generation call must name one checked-in output: %s", call)
			outputs = append(outputs, output[0])
		}
		require.ElementsMatch(t, want, outputs)
	})
	// A retired docs source must not remain as a mutation-only prerequisite.
	retiredSources := regexp.MustCompile(`version-8\.[0-7]([^0-9]|$)`).FindAllString(string(script), -1)
	require.Empty(t, retiredSources, "retired docs sources must not remain as generation prerequisites")
}
