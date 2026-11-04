// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestAllCommandHelpFocusesOnActions checks every command, including grouping
// and hidden commands, without coupling help to runtime result formatting.
func TestAllCommandHelpFocusesOnActions(t *testing.T) {
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		t.Run(command.CommandPath(), func(t *testing.T) {
			prose := strings.ToLower(command.Short + "\n" + command.Long)
			for _, presentation := range []string{
				"default human", "stdout", "stderr", "milestone", "repaint",
				"shared error envelope", "one key per line", "zero bytes",
				"warning-level", "selection scope:", "creation target:",
				"compact human", "under matching element rows",
			} {
				require.NotContains(t, prose, presentation)
			}
		})
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(Root())
}

// TestProcessInstanceMutationStateHelp keeps suspended selection discoverable
// with its version requirement while preserving the existing selector default.
func TestProcessInstanceMutationStateHelp(t *testing.T) {
	for _, verb := range []string{"cancel", "delete"} {
		t.Run(verb, func(t *testing.T) {
			command, _, err := Root().Find([]string{verb, "process-instance"})
			require.NoError(t, err)
			state := command.Flags().Lookup("state")
			require.NotNil(t, state)
			require.Contains(t, state.Usage, "all, active, suspended (only c8.10+), completed, canceled, terminated")
			require.Equal(t, "all", state.DefValue)
			require.Equal(t, "s", state.Shorthand)
		})
	}
}

// TestVersionCheckHelp keeps connection guidance concise while documenting the advisory.
func TestVersionCheckHelp(t *testing.T) {
	require.Contains(t, configTestConnectionCmd.Long, "warn")
	require.Contains(t, configTestConnectionCmd.Long, "does not fail")
	require.Contains(t, Root().PersistentFlags().Lookup("no-version-check").Usage, "advisory")
}

// TestOperatorHelpContract protects the brief operational guidance that must
// survive edits without requiring edge-case manuals in every command's help.
func TestOperatorHelpContract(t *testing.T) {
	cases := []struct {
		command   *cobra.Command
		fragments []string
	}{
		{Root(), []string{"c8volt v5", "Camunda 8.10", "8.9 and 8.8 remain supported", "--dry-run", "--auto-confirm"}},
		{getProcessInstanceCmd, []string{"filters combine with AND", "--bid", "only c8.10+", "--with-incidents", "--with-elements"}},
		{getProcessDefinitionCmd, []string{"--latest", "--with-stats", "DRAINING", "active and suspended counts are separate", "--watch", "--xml"}},
		{getElementCmd, []string{"--pi-key", "--with-vars", "Parent-scope variables are excluded", "worker waiting time"}},
		{getUserTaskCmd, []string{"--bid", "local variables", "including inherited", "Explicit keys"}},
		{updateJobCmd, []string{"--set-retries", "--set-timeout", "--timeout controls HTTP", "--throw-bpmn-error", "--set-error-code", "frozen before confirmation", "partial failures"}},
		{updateProcessInstanceCmd, []string{"root-scope variables", "--set-vars-file", "--set-bid", "uniqueness disabled", "reassignment conflicts"}},
		{updateElementCmd, []string{"element-local", "parent and sibling scopes", "Unchanged plans need no writes"}},
		{updateUserTaskCmd, []string{"owning scope", "--complete", "Already completed tasks", "--set-assignee", "--auto-confirm alone", "cannot be combined with --no-wait"}},
		{runProcessInstanceCmd, []string{"latest version", "--set-vars", "--set-bid", "does not mean the process has completed"}},
		{cancelProcessInstanceCmd, []string{"affected instance families", "--force", "terminal", "Empty searches"}},
		{deleteProcessInstanceCmd, []string{"children before parents", "Unfinished instances require --force", "Empty searches"}},
		{deleteProcessDefinitionCmd, []string{"Unfinished instances block deletion", "--force", "cannot be", "independently confirmed"}},
		{resolveProcessInstanceCmd, []string{"incident scope before confirmation", "does not", "job retries"}},
		{walkProcessInstanceCmd, []string{"--parent", "--children", "multiple matches are ambiguous"}},
		{suspendProcessInstanceCmd, []string{"only c8.10+", "--tree", "ancestors and siblings are excluded", "does not stop running worker code", "do not roll back"}},
		{resumeProcessInstanceCmd, []string{"only c8.10+", "--tree", "ancestors and siblings are excluded", "buffered work", "do not roll back"}},
		{opsRepairIncidentCmd, []string{"--set-retries 0 skips retry restoration", "--dry-run", "--report-file"}},
		{opsRepairProcessInstanceCmd, []string{"--set-retries 0 skips retry restoration", "--dry-run", "--report-file"}},
	}
	for _, tc := range cases {
		t.Run(tc.command.CommandPath(), func(t *testing.T) {
			prose := strings.Join(strings.Fields(tc.command.Long), " ")
			for _, fragment := range tc.fragments {
				require.Contains(t, prose, fragment)
			}
		})
	}
}

// TestAllCommandHelpStaysCompact catches later feature additions that turn help
// back into a contract manual or crowd out the everyday operator examples.
func TestAllCommandHelpStaysCompact(t *testing.T) {
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		t.Run(command.CommandPath(), func(t *testing.T) {
			require.LessOrEqual(t, len(strings.Fields(command.Long)), 150)
			examples := strings.Split(strings.TrimSpace(command.Example), "\n")
			require.LessOrEqual(t, len(examples), 5)
			for _, uncommon := range []string{"--no-state-check", "--no-worker-limit", "--no-version-check", "--backoff-", "--watch-timeout", "--request-rate", "--var-value-limit", "--log-with-source"} {
				require.NotContains(t, command.Example, uncommon)
			}
			require.NotContains(t, command.Example, "./c8volt")
		})
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(Root())
}

// TestPublishedExamplesUseRegisteredFlags validates help and authored Markdown
// examples locally. It never executes a command or bootstraps a remote client.
func TestPublishedExamplesUseRegisteredFlags(t *testing.T) {
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		checkPublishedExamples(t, command.CommandPath(), command.Example)
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(Root())
	paths := []string{filepath.Join("..", "README.md")}
	for _, pattern := range []string{"../docs/*.md", "../docs/ops/*.md", "../docs/cli/index.md", "../docs/cli/runtime-elements.md"} {
		matches, err := filepath.Glob(pattern)
		require.NoError(t, err)
		paths = append(paths, matches...)
	}
	for _, path := range paths {
		if filepath.Base(path) == "index.md" && filepath.Dir(path) == "../docs" {
			continue
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		checkPublishedExamples(t, path, string(data))
	}
}

// checkPublishedExamples tokenizes the simple shell invocations used in docs,
// resolves their leaf command, and checks flags without parsing global values.
func checkPublishedExamples(t *testing.T, source, content string) {
	t.Helper()
	tokens := regexp.MustCompile(`'[^']*'|"(?:[^"\\]|\\.)*"|\||[^\s|]+`)
	for lineNumber, line := range strings.Split(content, "\n") {
		fields := tokens.FindAllString(strings.TrimSpace(line), -1)
		for start := 0; start < len(fields); {
			end := start
			for end < len(fields) && fields[end] != "|" {
				end++
			}
			if start < end && fields[start] == "c8volt" {
				args := append([]string(nil), fields[start+1:end]...)
				for i := range args {
					args[i] = strings.Trim(args[i], "\"'")
				}
				require.NoError(t, validatePublishedExample(Root(), args), "%s:%d", source, lineNumber+1)
			}
			start = end + 1
		}
	}
}

// validatePublishedExample checks syntax without mutating flag values, calling
// Args hooks, reading stdin, or initializing command services.
func validatePublishedExample(root *cobra.Command, args []string) error {
	command, remaining, err := root.Find(args)
	if err != nil {
		return err
	}
	bpmnAction, errorCode := false, false
	for i := 0; i < len(remaining); i++ {
		arg := remaining[i]
		if arg == "-" || arg == "--help" {
			continue
		}
		if !strings.HasPrefix(arg, "-") || arg == "--" {
			return fmt.Errorf("unexpected positional argument %q for %s", arg, command.CommandPath())
		}
		name, value, attached := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		flag := command.Flags().Lookup(name)
		if flag == nil {
			flag = command.InheritedFlags().Lookup(name)
		}
		if shorthand := strings.TrimPrefix(name, "-"); flag == nil && !strings.HasPrefix(arg, "--") && len(shorthand) == 1 {
			flag = command.Flags().ShorthandLookup(shorthand)
			if flag == nil {
				flag = command.InheritedFlags().ShorthandLookup(shorthand)
			}
		}
		if flag == nil {
			return fmt.Errorf("unknown flag %s for %s", arg, command.CommandPath())
		}
		if flag.Name == "throw-bpmn-error" {
			if attached {
				return fmt.Errorf("BPMN-error action must be bare")
			}
			bpmnAction = true
		}
		if flag.NoOptDefVal == "" && !attached {
			i++
			if i >= len(remaining) {
				return fmt.Errorf("missing value for %s", arg)
			}
			value = remaining[i]
		}
		if flag.Name == "set-error-code" {
			errorCode = strings.TrimSpace(value) != ""
		}
	}
	if bpmnAction != errorCode {
		return fmt.Errorf("BPMN-error action requires --set-error-code and vice versa")
	}
	return nil
}

// TestPublishedExampleValidation detects obsolete actions and malformed examples
// while accepting canonical attached values and opaque flag-looking payloads.
func TestPublishedExampleValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		valid bool
	}{
		{"known", []string{"get", "job", "--key", "<job-key>"}, true},
		{"unknown flag", []string{"get", "job", "--obsolete"}, false},
		{"unknown short flag", []string{"get", "job", "-obsolete"}, false},
		{"missing value", []string{"get", "job", "--key"}, false},
		{"unexpected argument", []string{"get", "job", "extra"}, false},
		{"canonical code", []string{"update", "job", "--throw-bpmn-error", "--set-error-code=DECLINED"}, true},
		{"attached action", []string{"update", "job", "--throw-bpmn-error=DECLINED"}, false},
		{"positional code", []string{"update", "job", "--throw-bpmn-error", "DECLINED"}, false},
		{"missing code", []string{"update", "job", "--throw-bpmn-error"}, false},
		{"code without action", []string{"update", "job", "--set-error-code", "DECLINED"}, false},
		{"opaque value", []string{"update", "job", "--fail", "--set-error-message", "--obsolete"}, true},
		{"inherited short flag", []string{"get", "job", "-j"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePublishedExample(Root(), tc.args)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestOperatorHelpUsesOmissionForUnboundedDefaults protects the distinction
// between omitted limit/worker flags and explicitly supplied zero values.
func TestOperatorHelpUsesOmissionForUnboundedDefaults(t *testing.T) {
	for _, command := range []*cobra.Command{getProcessInstanceCmd, cancelProcessInstanceCmd, deleteProcessInstanceCmd, opsExecuteRetentionPolicyCmd, opsAnalyseSlowProcessInstancesCmd} {
		limit := command.Flags().Lookup("limit")
		require.NotNil(t, limit)
		require.Contains(t, limit.Usage, "omit for all matches")
		require.NotContains(t, limit.Usage, "0 selects")
	}
	for _, command := range []*cobra.Command{getProcessInstanceCmd, cancelProcessInstanceCmd, deleteProcessInstanceCmd, runProcessInstanceCmd, updateUserTaskCmd, updateJobCmd} {
		workers := command.Flags().Lookup("workers")
		require.NotNil(t, workers)
		require.Contains(t, workers.Usage, "omit for an automatic limit")
	}
}
