// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ferrors

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
)

// ProcessInstanceMutationFailure exposes operational facts for command-owned
// human formatting while preserving the original service error and machine text.
type ProcessInstanceMutationFailure struct {
	Operation              string
	Phase                  string
	FailureReason          string
	RootKey                string
	Scope                  []string
	Timeout                time.Duration
	LastStates             map[string]string
	DeleteConflictKey      string
	CancellationSubmitted  bool
	ResumedDeletionReached bool
	err                    error
}

func (e *ProcessInstanceMutationFailure) Error() string { return e.err.Error() }
func (e *ProcessInstanceMutationFailure) Unwrap() error { return e.err }

type classifiedError struct {
	class error
	cause error
}

func (e *classifiedError) Error() string        { return e.class.Error() + ": " + e.cause.Error() }
func (e *classifiedError) Unwrap() error        { return e.cause }
func (e *classifiedError) Is(target error) bool { return target == e.class }

// Class is the bounded machine-facing classification for CLI failures.
type Class string

const (
	ClassInvalidInput      Class = "invalid_input"
	ClassLocalPrecondition Class = "local_precondition"
	ClassUnsupported       Class = "unsupported"
	ClassNotFound          Class = "not_found"
	ClassConflict          Class = "conflict"
	ClassTimeout           Class = "timeout"
	ClassUnavailable       Class = "unavailable"
	ClassMalformedResponse Class = "malformed_response"
	ClassInternal          Class = "internal"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrLocalPrecondition = errors.New("local precondition failed")
	ErrUnsupported       = errors.New("unsupported capability")
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("conflict")
	ErrTimeout           = errors.New("operation timed out")
	ErrUnavailable       = errors.New("service unavailable")
	ErrMalformedResponse = errors.New("malformed response")
	ErrInternal          = errors.New("internal error")
	ErrFailedFast        = errors.New("operation failed fast due to context cancellation")

	// Legacy aliases kept for existing call sites and downstream `errors.Is` checks.
	ErrBadRequest   = ErrInvalidInput
	ErrInvalidState = ErrInvalidInput
)

// NormalizeDomain maps service/domain transport and payload errors into the shared CLI model.
func NormalizeDomain(err error) error {
	if err == nil {
		return nil
	}

	err = exposeProcessInstanceMutationFailure(err)
	switch {
	case isNormalized(err):
		return err
	case errors.Is(err, domain.ErrBadRequest),
		errors.Is(err, domain.ErrValidation):
		return wrap(ErrInvalidInput, err)
	case errors.Is(err, domain.ErrUnauthorized),
		errors.Is(err, domain.ErrForbidden),
		errors.Is(err, domain.ErrPrecondition):
		return wrap(ErrLocalPrecondition, err)
	case errors.Is(err, domain.ErrUnsupported):
		return wrap(ErrUnsupported, err)
	case errors.Is(err, domain.ErrNotFound):
		return wrap(ErrNotFound, err)
	case errors.Is(err, domain.ErrConflict):
		return wrap(ErrConflict, err)
	case errors.Is(err, domain.ErrGatewayTimeout),
		errors.Is(err, context.DeadlineExceeded):
		return wrap(ErrTimeout, err)
	case errors.Is(err, domain.ErrRateLimited),
		errors.Is(err, domain.ErrUnavailable):
		return wrap(ErrUnavailable, err)
	case errors.Is(err, domain.ErrMalformedResponse):
		return wrap(ErrMalformedResponse, err)
	case errors.Is(err, domain.ErrUpstream),
		errors.Is(err, domain.ErrInternal):
		return wrap(ErrInternal, err)
	default:
		return err
	}
}

// NormalizeLocal maps local configuration, lifecycle, and unsupported-version errors into the shared CLI model.
func NormalizeLocal(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case isNormalized(err):
		return err
	case errors.Is(err, services.ErrNoConfig),
		errors.Is(err, services.ErrNoHTTPClient),
		errors.Is(err, services.ErrNoLogger):
		return wrap(ErrLocalPrecondition, err)
	case errors.Is(err, services.ErrUnknownAPIVersion):
		return wrap(ErrUnsupported, err)
	case errors.Is(err, toolx.ErrUnknownCamundaVersion):
		return wrap(ErrUnsupported, err)
	case errors.Is(err, context.Canceled),
		errors.Is(err, ErrFailedFast):
		return wrap(ErrLocalPrecondition, err)
	default:
		return err
	}
}

// Normalize is the shared CLI normalization entry point used before rendering or
// exit-code resolution. It classifies failures but intentionally preserves the
// wrapped detail text; upstream wrappers own message deduplication.
func Normalize(err error) error {
	if err == nil {
		return nil
	}

	err = NormalizeLocal(err)
	err = NormalizeDomain(err)

	if isNormalized(err) {
		return err
	}

	return wrap(ErrInternal, err)
}

// WrapClass applies a shared CLI failure class without changing the wrapped
// error text. It must not become a second message-composition layer.
func WrapClass(classErr error, err error) error {
	return wrap(classErr, err)
}

// FromDomain normalizes domain/service errors without applying local CLI error classification.
func FromDomain(err error) error {
	return NormalizeDomain(err)
}

// Classify returns the machine-facing failure class after full normalization.
func Classify(err error) Class {
	normalized := Normalize(err)
	if normalized == nil {
		return ""
	}
	switch {
	case matchesClassification(normalized, ErrInvalidInput):
		return ClassInvalidInput
	case matchesClassification(normalized, ErrLocalPrecondition):
		return ClassLocalPrecondition
	case matchesClassification(normalized, ErrUnsupported):
		return ClassUnsupported
	case matchesClassification(normalized, ErrNotFound):
		return ClassNotFound
	case matchesClassification(normalized, ErrConflict):
		return ClassConflict
	case matchesClassification(normalized, ErrTimeout):
		return ClassTimeout
	case matchesClassification(normalized, ErrUnavailable):
		return ClassUnavailable
	case matchesClassification(normalized, ErrMalformedResponse):
		return ClassMalformedResponse
	default:
		return ClassInternal
	}
}

// matchesClassification preserves the historical priority across joined branches,
// while an explicit class masks its own cause for classification only.
// Unwrapping remains available to errors.Is/As callers.
func matchesClassification(err, target error) bool {
	if err == nil {
		return false
	}
	if classified, ok := err.(*classifiedError); ok {
		return classified.class == target
	}
	if err == target {
		return true
	}
	if matcher, ok := err.(interface{ Is(error) bool }); ok && matcher.Is(target) {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			if matchesClassification(child, target) {
				return true
			}
		}
		return false
	}
	return matchesClassification(errors.Unwrap(err), target)
}

// ExitCode maps a normalized or raw error to the process exit code used by CLI entry points.
func ExitCode(err error) int {
	switch Classify(err) {
	case "":
		return exitcode.OK
	case ClassInvalidInput:
		return exitcode.InvalidArgs
	case ClassNotFound:
		return exitcode.NotFound
	case ClassTimeout:
		return exitcode.Timeout
	case ClassUnavailable:
		return exitcode.Unavailable
	case ClassConflict:
		return exitcode.Conflict
	default:
		return exitcode.Error
	}
}

// ResolveExitCode applies the --no-err-codes behavior after deriving the standard exit code.
// noErrCodes forces success for failures so shell callers can opt out of non-zero exits while still receiving error output.
func ResolveExitCode(noErrCodes bool, err error) int {
	code := ExitCode(err)
	if err != nil && noErrCodes {
		return exitcode.OK
	}
	return code
}

// Outcome returns the compact result label used in machine-readable error envelopes.
func Outcome(err error) string {
	switch Classify(err) {
	case "":
		return ""
	case ClassInvalidInput:
		return "invalid"
	default:
		return "failed"
	}
}

// HandleAndExitOK logs a success message and terminates the process with the OK exit code.
func HandleAndExitOK(log *slog.Logger, message string) {
	log.Info(message)
	os.Exit(exitcode.OK)
}

// HandleAndExit normalizes, logs, and exits for CLI entry points.
// noErrCodes mirrors --no-err-codes and can turn a failure into an OK process status.
func HandleAndExit(log *slog.Logger, noErrCodes bool, err error) {
	if err == nil {
		os.Exit(exitcode.OK)
	}

	err = Normalize(err)
	log.Error(err.Error())
	os.Exit(ResolveExitCode(noErrCodes, err))
}

// isNormalized reports whether err already carries one of the shared facade error sentinels.
func isNormalized(err error) bool {
	return errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, ErrLocalPrecondition) ||
		errors.Is(err, ErrUnsupported) ||
		errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrConflict) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrUnavailable) ||
		errors.Is(err, ErrMalformedResponse) ||
		errors.Is(err, ErrInternal)
}

// wrap attaches a failure-class sentinel while preserving the original error text exactly once.
func wrap(classErr error, err error) error {
	if err == nil || errors.Is(err, classErr) {
		return err
	}
	return &classifiedError{class: classErr, cause: err}
}

func exposeProcessInstanceMutationFailure(err error) error {
	var exposed *ProcessInstanceMutationFailure
	if errors.As(err, &exposed) {
		return err
	}
	var failure *domain.ProcessInstanceMutationFailure
	if !errors.As(err, &failure) {
		return err
	}
	return projectProcessInstanceMutationFailure(failure, err)
}

// ProcessInstanceMutationFailures returns every per-tree mutation failure in a
// joined error while keeping the error tree and its ordering untouched.
func ProcessInstanceMutationFailures(err error) []*ProcessInstanceMutationFailure {
	var domainFailures []*domain.ProcessInstanceMutationFailure
	var exposedFailures []*ProcessInstanceMutationFailure
	seenDomain := make(map[*domain.ProcessInstanceMutationFailure]struct{})
	seenExposed := make(map[*ProcessInstanceMutationFailure]struct{})
	var collect func(error)
	collect = func(candidate error) {
		if candidate == nil {
			return
		}
		switch failure := candidate.(type) {
		case *domain.ProcessInstanceMutationFailure:
			if _, ok := seenDomain[failure]; !ok {
				seenDomain[failure] = struct{}{}
				domainFailures = append(domainFailures, failure)
			}
			// An outer mutation failure enriches an inner failure for the same
			// tree; treating both as roots would double-count submitted work.
			return
		case *ProcessInstanceMutationFailure:
			if _, ok := seenExposed[failure]; !ok {
				seenExposed[failure] = struct{}{}
				exposedFailures = append(exposedFailures, failure)
			}
		}
		if joined, ok := candidate.(interface{ Unwrap() []error }); ok {
			for _, child := range joined.Unwrap() {
				collect(child)
			}
			return
		}
		collect(errors.Unwrap(candidate))
	}
	collect(err)
	if len(domainFailures) == 0 {
		return exposedFailures
	}
	projected := make([]*ProcessInstanceMutationFailure, 0, len(domainFailures))
	for _, failure := range domainFailures {
		projected = append(projected, projectProcessInstanceMutationFailure(failure, failure))
	}
	return projected
}

func projectProcessInstanceMutationFailure(failure *domain.ProcessInstanceMutationFailure, err error) *ProcessInstanceMutationFailure {
	states := make(map[string]string, len(failure.LastStates))
	for key, state := range failure.LastStates {
		states[key] = state.String()
	}
	return &ProcessInstanceMutationFailure{
		Operation:              failure.Operation,
		Phase:                  failure.Phase,
		FailureReason:          failure.FailureReason,
		RootKey:                failure.RootKey,
		Scope:                  slices.Clone(failure.Scope),
		Timeout:                failure.Timeout,
		LastStates:             states,
		DeleteConflictKey:      failure.DeleteConflictKey,
		CancellationSubmitted:  failure.CancellationSubmitted,
		ResumedDeletionReached: failure.ResumedDeletionReached,
		err:                    err,
	}
}
