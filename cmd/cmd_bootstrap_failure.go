// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

// failBeforeCli reports command validation failures before the CLI client is created.
func failBeforeCli(cmd *cobra.Command, err error) {
	log, noErrCodes := bootstrapFailureContext(cmd)
	handleCommandError(cmd, log, noErrCodes, err)
}
