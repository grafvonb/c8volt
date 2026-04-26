// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var bootstrapErrorClassRules = append([]errorClassRule{
	{match: config.ErrNoConfigInContext, class: ferrors.ErrLocalPrecondition},
	{match: config.ErrInvalidServiceInContext, class: ferrors.ErrLocalPrecondition},
	{match: config.ErrNoBaseURL, class: ferrors.ErrLocalPrecondition},
	{match: config.ErrNoTokenURL, class: ferrors.ErrLocalPrecondition},
	{match: config.ErrNoClientID, class: ferrors.ErrLocalPrecondition},
	{match: config.ErrNoClientSecret, class: ferrors.ErrLocalPrecondition},
	{match: config.ErrInvalidLogLevel, class: ferrors.ErrLocalPrecondition},
	{match: config.ErrInvalidLogFormat, class: ferrors.ErrLocalPrecondition},
	{match: config.ErrProfileNotFound, class: ferrors.ErrInvalidInput},
	{match: httpc.ErrNoHttpServiceInContext, class: ferrors.ErrLocalPrecondition},
	{match: httpc.ErrInvalidServiceInContext, class: ferrors.ErrLocalPrecondition},
}, commandErrorClassRules...)

func normalizeBootstrapError(err error) error {
	if err == nil {
		return nil
	}

	for _, rule := range bootstrapErrorClassRules {
		if errors.Is(err, rule.match) {
			return ferrors.WrapClass(rule.class, err)
		}
	}

	return ferrors.Normalize(err)
}

func bootstrapLocalPrecondition(err error) error {
	return ferrors.WrapClass(ferrors.ErrLocalPrecondition, err)
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
