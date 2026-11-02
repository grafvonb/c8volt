// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

// handleBusinessIDSearchError keeps native query/enrichment transport chains out of normal diagnostics while preserving machine evidence and exit classes.
func handleBusinessIDSearchError(cmd *cobra.Command, log *slog.Logger, noErrCodes bool, err error, selected ...int) {
	if searchBusinessIDFlag(cmd) == nil {
		handleCommandError(cmd, log, noErrCodes, err)
		return
	}
	message := "business-ID search failed: " + boundedCommandFailureReason(err)
	if len(selected) > 0 && selected[0] > 0 {
		message = fmt.Sprintf("business-ID search failed: %d selected; %s", selected[0], boundedCommandFailureReason(err))
	}
	if errors.Is(err, ferrors.ErrUnsupported) {
		message += "; business-ID search requires Camunda 8.10"
	}
	handleCommandError(cmd, log, noErrCodes, err, message)
}
