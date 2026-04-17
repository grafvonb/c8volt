package ferrors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
)

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

func FromDomain(err error) error {
	return NormalizeDomain(err)
}

func Classify(err error) Class {
	switch normalized := Normalize(err); {
	case normalized == nil:
		return ""
	case errors.Is(normalized, ErrInvalidInput):
		return ClassInvalidInput
	case errors.Is(normalized, ErrLocalPrecondition):
		return ClassLocalPrecondition
	case errors.Is(normalized, ErrUnsupported):
		return ClassUnsupported
	case errors.Is(normalized, ErrNotFound):
		return ClassNotFound
	case errors.Is(normalized, ErrConflict):
		return ClassConflict
	case errors.Is(normalized, ErrTimeout):
		return ClassTimeout
	case errors.Is(normalized, ErrUnavailable):
		return ClassUnavailable
	case errors.Is(normalized, ErrMalformedResponse):
		return ClassMalformedResponse
	default:
		return ClassInternal
	}
}

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

func ResolveExitCode(noErrCodes bool, err error) int {
	code := ExitCode(err)
	if err != nil && noErrCodes {
		return exitcode.OK
	}
	return code
}

func HandleAndExitOK(log *slog.Logger, message string) {
	log.Info(message)
	os.Exit(exitcode.OK)
}

func HandleAndExit(log *slog.Logger, noErrCodes bool, err error) {
	if err == nil {
		os.Exit(exitcode.OK)
	}

	err = Normalize(err)
	log.Error(err.Error())
	os.Exit(ResolveExitCode(noErrCodes, err))
}

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

func wrap(classErr error, err error) error {
	if err == nil || errors.Is(err, classErr) {
		return err
	}
	return fmt.Errorf("%w: %v", classErr, err)
}
