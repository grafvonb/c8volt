// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
	"unicode/utf8"
)

// searchBusinessIDFlag preserves scalar bytes and distinguishes omission from explicit empty input.
func searchBusinessIDFlag(cmds ...*cobra.Command) *string {
	if len(cmds) == 0 || cmds[0] == nil || !cmds[0].Flags().Changed("bid") {
		return nil
	}
	value, _ := cmds[0].Flags().GetString("bid")
	return &value
}

// validateSearchBusinessIDFlag gives local input feedback; adapters remain authoritative for library callers.
func validateSearchBusinessIDFlag(cmd *cobra.Command) error {
	value := searchBusinessIDFlag(cmd)
	if value != nil && (*value == "" || !utf8.ValidString(*value)) {
		return invalidFlagValuef("--bid requires a nonempty valid UTF-8 value")
	}
	return nil
}
