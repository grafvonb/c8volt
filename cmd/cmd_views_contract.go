// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"log/slog"
	"os"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

func renderResultEnvelope[T any](cmd *cobra.Command, envelope ResultEnvelope[T]) error {
	cmd.Print(toolx.ToJSONString(envelope))
	return nil
}

func renderSucceededResult[T any](cmd *cobra.Command, payload T) error {
	return renderResultEnvelope(cmd, ResultEnvelope[T]{
		Outcome: OutcomeSucceeded,
		Command: commandPath(cmd),
		Payload: payload,
	})
}

func renderAcceptedResult[T any](cmd *cobra.Command, payload T) error {
	return renderResultEnvelope(cmd, ResultEnvelope[T]{
		Outcome: OutcomeAccepted,
		Command: commandPath(cmd),
		Payload: payload,
	})
}

func commandUsesSharedEnvelope(cmd *cobra.Command, mode RenderMode) bool {
	return cmd != nil && machineReadableModeEnabled(mode) && contractSupportForCommand(cmd) == ContractSupportFull
}

func renderJSONPayload[T any](cmd *cobra.Command, mode RenderMode, payload T) error {
	if commandUsesSharedEnvelope(cmd, mode) {
		return renderSucceededResult(cmd, payload)
	}
	cmd.Print(toolx.ToJSONString(payload))
	return nil
}

func renderCommandResult[T any](cmd *cobra.Command, payload T) error {
	if !commandUsesSharedEnvelope(cmd, pickMode()) {
		return nil
	}
	if commandMutationForCommand(cmd) == CommandMutationStateChanging && flagNoWait {
		return renderAcceptedResult(cmd, payload)
	}
	return renderSucceededResult(cmd, payload)
}

func handleCommandError(cmd *cobra.Command, log *slog.Logger, noErrCodes bool, err error) {
	if err == nil {
		return
	}
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		_ = renderResultEnvelope(cmd, resultEnvelopeForError(cmd, err))
		os.Exit(ferrors.ResolveExitCode(noErrCodes, err))
	}
	ferrors.HandleAndExit(log, noErrCodes, err)
}

func resultEnvelopeForError(cmd *cobra.Command, err error) ResultEnvelope[any] {
	normalized := ferrors.Normalize(err)
	class := string(ferrors.Classify(normalized))
	detail := &ResultDetail{
		Message: strings.TrimSpace(normalized.Error()),
		Class:   class,
	}

	return ResultEnvelope[any]{
		Outcome: outcomeForError(normalized),
		Class:   class,
		Command: commandPath(cmd),
		Detail:  detail,
	}
}

func outcomeForError(err error) Outcome {
	switch ferrors.Outcome(err) {
	case "invalid":
		return OutcomeInvalid
	case "failed":
		return OutcomeFailed
	default:
		return OutcomeFailed
	}
}
