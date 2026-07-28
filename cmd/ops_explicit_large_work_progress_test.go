// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestExplicitLargeWorkProgressEventRespectsOutputModes verifies explicit-work counters stay off machine-mode stderr/stdout paths.
func TestExplicitLargeWorkProgressEventRespectsOutputModes(t *testing.T) {
	prevVerbose, prevJSON, prevKeysOnly, prevQuiet, prevDebug := flagVerbose, flagViewAsJson, flagViewKeysOnly, flagQuiet, flagDebug
	t.Cleanup(func() {
		flagVerbose = prevVerbose
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
		flagQuiet = prevQuiet
		flagDebug = prevDebug
	})

	event := ops.ProgressEvent{
		Kind: ops.ProgressEventKindFrozenScope,
		FrozenScope: &ops.FrozenScopeProgress{
			Phase:        "starting process instances",
			CoreResource: "process instance(s)",
			Done:         2,
			Total:        3,
		},
	}

	t.Run("verbose", func(t *testing.T) {
		flagVerbose, flagViewAsJson, flagViewKeysOnly, flagQuiet, flagDebug = true, false, false, false, false
		var stderr bytes.Buffer
		cmd := newExplicitLargeWorkProgressTestCommand(&stderr)
		cmd.SetContext(context.Background())

		printExplicitLargeWorkProgressEvent(cmd, event)

		require.Contains(t, stderr.String(), "starting process instances 2/3 process instance(s)")
	})

	t.Run("json", func(t *testing.T) {
		flagVerbose, flagViewAsJson, flagViewKeysOnly, flagQuiet, flagDebug = false, true, false, false, false
		var stderr bytes.Buffer
		cmd := newExplicitLargeWorkProgressTestCommand(&stderr)
		cmd.SetContext(context.Background())

		printExplicitLargeWorkProgressEvent(cmd, event)

		require.Empty(t, stderr.String())
	})

	t.Run("keys-only", func(t *testing.T) {
		flagVerbose, flagViewAsJson, flagViewKeysOnly, flagQuiet, flagDebug = false, false, true, false, false
		var stderr bytes.Buffer
		cmd := newExplicitLargeWorkProgressTestCommand(&stderr)
		cmd.SetContext(context.Background())

		printExplicitLargeWorkProgressEvent(cmd, event)

		require.Empty(t, stderr.String())
	})
}

// newExplicitLargeWorkProgressTestCommand pins automation false so package-global flag state cannot affect progress gating.
func newExplicitLargeWorkProgressTestCommand(stderr *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("automation", false, "")
	cmd.SetErr(stderr)
	return cmd
}
