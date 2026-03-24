package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

func normalizeBootstrapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, config.ErrNoConfigInContext),
		errors.Is(err, config.ErrInvalidServiceInContext),
		errors.Is(err, config.ErrNoBaseURL),
		errors.Is(err, config.ErrNoTokenURL),
		errors.Is(err, config.ErrNoClientID),
		errors.Is(err, config.ErrNoClientSecret),
		errors.Is(err, config.ErrInvalidLogLevel),
		errors.Is(err, config.ErrInvalidLogFormat),
		errors.Is(err, httpc.ErrNoHttpServiceInContext),
		errors.Is(err, httpc.ErrInvalidServiceInContext):
		return wrapBootstrapClass(ferrors.ErrLocalPrecondition, err)
	default:
		return ferrors.Normalize(err)
	}
}

func bootstrapLocalPrecondition(err error) error {
	return wrapBootstrapClass(ferrors.ErrLocalPrecondition, err)
}

func handleBootstrapError(cmd *cobra.Command, err error) {
	log, noErrCodes := bootstrapFailureContext(cmd)
	ferrors.HandleAndExit(log, noErrCodes, normalizeBootstrapError(err))
}

func bootstrapFailureContext(cmd *cobra.Command) (*slog.Logger, bool) {
	log := slog.Default()
	noErrCodes := flagNoErrCodes
	if cmd == nil {
		return log, noErrCodes
	}

	if ctxLog, err := logging.FromContext(cmd.Context()); err == nil && ctxLog != nil {
		log = ctxLog
	}
	if cfg, err := config.FromContext(cmd.Context()); err == nil && cfg != nil {
		noErrCodes = cfg.App.NoErrCodes
	}
	return log, noErrCodes
}

func wrapBootstrapClass(classErr error, err error) error {
	if err == nil || errors.Is(err, classErr) {
		return err
	}
	return fmt.Errorf("%w: %v", classErr, err)
}
