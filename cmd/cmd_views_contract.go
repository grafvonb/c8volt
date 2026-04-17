package cmd

import (
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
