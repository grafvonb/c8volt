// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatConfirmationPromptSplitsFinalQuestion(t *testing.T) {
	got := formatConfirmationPrompt(
		"All process-definitions purge matched 6 candidate process definition(s); delete planning will affect 170 process instance(s) across 6 unique process definition(s). Do you want to proceed?",
		"[y/N]",
	)

	require.Equal(t, ""+
		"All process-definitions purge matched 6 candidate process definition(s); delete planning will affect 170 process instance(s) across 6 unique process definition(s).\n"+
		"Do you want to proceed? [y/N]: ", got)
}

func TestFormatConfirmationPromptKeepsSingleQuestionOnOneLine(t *testing.T) {
	got := formatConfirmationPrompt("Proceed with this deletion?", "[y/N]")

	require.Equal(t, "Proceed with this deletion? [y/N]: ", got)
}

func TestFormatConfirmationPromptSplitsContinuationQuestion(t *testing.T) {
	got := formatConfirmationPrompt(
		"Fetched 2 process instance(s) on this page (2/3+ loaded). More matching process instances remain. Continue?",
		"[y/N]",
	)

	require.Equal(t, ""+
		"Fetched 2 process instance(s) on this page (2/3+ loaded). More matching process instances remain.\n"+
		"Continue? [y/N]: ", got)
}

func TestFormatConfirmationPromptPreservesExistingMultilinePrompt(t *testing.T) {
	got := formatConfirmationPrompt("Summary line\nDo you want to proceed?", "[Y/n]")

	require.Equal(t, "Summary line\nDo you want to proceed? [Y/n]: ", got)
}
