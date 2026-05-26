// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestResetCommandTreeFlagsPreservesEmptyStringSliceDefault(t *testing.T) {
	var values []string
	root := &cobra.Command{Use: "root"}
	root.Flags().StringSliceVar(&values, "key", nil, "keys")

	require.NoError(t, root.Flags().Set("key", "123"))
	ResetCommandTreeFlags(root)

	require.Nil(t, values)
	require.NoError(t, root.Flags().Set("key", "456"))
	require.Equal(t, []string{"456"}, values)
}
