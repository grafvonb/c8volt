package ferrors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/exitcode"
)

var (
	ErrBadRequest   = errors.New("bad request, check params and payload provided")
	ErrInvalidState = errors.New("invalid process instance state")
	ErrNotFound     = errors.New("process instance not found")
	ErrTimeout      = errors.New("operation timed out")
	ErrConflict     = errors.New("conflict")
	ErrUnavailable  = errors.New("service unavailable")
	ErrInternal     = errors.New("internal error")
)

func FromDomain(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrBadRequest):
		return fmt.Errorf("%w: %s", ErrBadRequest, err)
	case errors.Is(err, domain.ErrNotFound):
		return fmt.Errorf("%w: %s", ErrNotFound, err)
	case errors.Is(err, domain.ErrConflict):
		return fmt.Errorf("%w: %s", ErrConflict, err)
	case errors.Is(err, domain.ErrGatewayTimeout) || errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %s", ErrTimeout, err)
	case errors.Is(err, domain.ErrUnavailable) || errors.Is(err, context.Canceled):
		return fmt.Errorf("%w: %s", ErrUnavailable, err)
	default:
		return fmt.Errorf("%w: %s", ErrInternal, err)
	}
}

func HandleAndExitOK(log *slog.Logger, message string) {
	log.Info(message)
	os.Exit(exitcode.OK)
}

func HandleAndExit(log *slog.Logger, err error) {
	if err == nil {
		os.Exit(exitcode.OK)
	}
	log.Error(err.Error())
	switch {
	case errors.Is(err, ErrBadRequest):
		os.Exit(exitcode.InvalidArgs)
	case errors.Is(err, ErrNotFound):
		os.Exit(exitcode.NotFound)
	case errors.Is(err, ErrTimeout):
		os.Exit(exitcode.Timeout)
	case errors.Is(err, ErrUnavailable):
		os.Exit(exitcode.Unavailable)
	case errors.Is(err, ErrConflict):
		os.Exit(exitcode.Conflict)
	default:
		os.Exit(exitcode.Error)
	}
}
