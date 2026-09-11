// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetElementCommand_DeclinedPagingUsesCommandStderr verifies the caller passes its stderr writer and stops before another element page.
func TestGetElementCommand_DeclinedPagingUsesCommandStderr(t *testing.T) {
	var bodies []map[string]any
	srv := newElementSearchServerResponses(t, &bodies,
		`{"items":[{"elementInstanceKey":"2251799813689002","state":"ACTIVE"}],"page":{"totalItems":2,"hasMoreTotalItems":true}}`,
		`{"items":[{"elementInstanceKey":"2251799813689003","state":"ACTIVE"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(writer io.Writer, autoConfirm bool, prompt string) error {
		_, err := writer.Write([]byte("element paging prompt writer\n"))
		require.NoError(t, err)
		require.False(t, autoConfirm)
		require.Contains(t, prompt, "More matching elements remain")
		return localPreconditionError(ErrCmdAborted)
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	stdout, stderr := executeRootForElementTestWithSeparateOutputs(t,
		"--config", cfgPath,
		"--keys-only",
		"get", "element",
		"--batch-size", "1",
	)

	require.Len(t, bodies, 1)
	require.Equal(t, "2251799813689002\n", stdout)
	require.Equal(t, "element paging prompt writer\n", stderr)
}
