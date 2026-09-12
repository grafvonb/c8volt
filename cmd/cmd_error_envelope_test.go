// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// commandErrorEnvelopeExpectation keeps regression expectations independent
// from the production result-envelope structs and renderer.
type commandErrorEnvelopeExpectation struct {
	Outcome string
	Class   string
	Command string
	Message string
}

// decodedCommandErrorEnvelope is the minimal independent wire shape needed to
// verify command execution errors without reusing production construction.
type decodedCommandErrorEnvelope struct {
	Outcome string `json:"outcome"`
	Class   string `json:"class"`
	Command string `json:"command"`
	Detail  *struct {
		Message    string `json:"message"`
		Class      string `json:"class"`
		Suggestion string `json:"suggestion"`
	} `json:"detail"`
}

// runCommandErrorEnvelopeSubprocess captures command results and diagnostics
// separately while asserting the exact process status selected by error policy.
func runCommandErrorEnvelopeSubprocess(t *testing.T, scope string, dir string, env map[string]string, stdin string, wantExitCode int) (string, string) {
	t.Helper()

	stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, scope, dir, env, stdin)
	if wantExitCode == 0 {
		require.NoError(t, err)
		return stdout, stderr
	}
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok, "expected subprocess exit error, got %T", err)
	require.Equal(t, wantExitCode, exitErr.ExitCode())
	return stdout, stderr
}

// assertCommandErrorEnvelope requires exactly one JSON value followed by EOF
// and verifies the established error-only schema and normalized detail.
func assertCommandErrorEnvelope(t *testing.T, stdout string, stderr string, want commandErrorEnvelopeExpectation) {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewBufferString(stdout))
	var got decodedCommandErrorEnvelope
	require.NoError(t, decoder.Decode(&got))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	require.Equal(t, want.Outcome, got.Outcome, "invalid input uses outcome invalid; all other existing failures use failed")
	require.Equal(t, want.Class, got.Class, "the envelope retains the existing normalized classification")
	require.Equal(t, want.Command, got.Command, "the actual Cobra command supplies the canonical command path")
	require.NotNil(t, got.Detail, "error envelopes retain normalized detail")
	require.Equal(t, want.Message, got.Detail.Message, "detail text stays trimmed, contextual, and actionable")
	require.Equal(t, want.Class, got.Detail.Class, "detail class matches the envelope class")
	require.Empty(t, got.Detail.Suggestion, "existing guidance remains in the detail message rather than a new suggestion")
	require.Empty(t, stderr, "JSON errors do not repeat the human diagnostic")

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(stdout), &fields))
	require.ElementsMatch(t, []string{"outcome", "class", "command", "detail"}, mapKeys(fields),
		"payload and unavailable tenant context remain omitted")
	var detailFields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(fields["detail"], &detailFields))
	require.ElementsMatch(t, []string{"message", "class"}, mapKeys(detailFields),
		"suggestion remains absent on this path")
}

// assertHumanCommandError verifies ordinary failures reserve stdout for command
// results and emit the normalized diagnostic exactly once on stderr.
func assertHumanCommandError(t *testing.T, stdout string, stderr string, message string) {
	t.Helper()
	require.Empty(t, stdout)
	require.Contains(t, stderr, message)
	require.Equal(t, 1, strings.Count(stderr, message), "human diagnostic must be emitted once")
}

// mapKeys returns JSON object keys for omission assertions without relying on
// production envelope types or field tags.
func mapKeys(fields map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	return keys
}
