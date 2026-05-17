// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var ErrCmdAborted = errors.New("aborted by user")

func NewCli(cmd *cobra.Command) (c8volt.API, *slog.Logger, *config.Config, error) {
	log, _ := logging.FromContext(cmd.Context())
	svcs, err := NewFromContext(cmd.Context())
	if err != nil {
		return nil, nil, nil, bootstrapLocalPrecondition(fmt.Errorf("error getting services from context: %w", err))
	}
	cli, err := c8volt.New(
		c8volt.WithConfig(svcs.Config),
		c8volt.WithHTTPClient(svcs.HTTP.Client()),
		c8volt.WithLogger(log),
	)
	if err != nil {
		return nil, nil, nil, normalizeBootstrapError(fmt.Errorf("error creating c8volt client: %w", err))
	}
	return cli, log, svcs.Config, nil
}

func handleNewCliError(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, err error) {
	if err == nil {
		return
	}

	// This helper owns runtime context only. Shared classification and final
	// rendering stay inside ferrors so command bootstrapping does not grow a
	// parallel error-composition path.
	noErrCodes := false
	if cfg != nil {
		noErrCodes = cfg.App.NoErrCodes
	} else {
		fallbackLog, fallbackNoErrCodes := bootstrapFailureContext(cmd)
		if log == nil {
			log = fallbackLog
		}
		noErrCodes = fallbackNoErrCodes
	}

	handleCommandError(cmd, log, noErrCodes, err)
}

func confirmCmdOrAbort(autoConfirm bool, prompt string) error {
	if autoConfirm || !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}
	fmt.Print(formatConfirmationPrompt(prompt, "[y/N]"))
	in := bufio.NewScanner(os.Stdin)
	if !in.Scan() {
		return localPreconditionError(ErrCmdAborted)
	}
	switch strings.ToLower(strings.TrimSpace(in.Text())) {
	case "y", "yes":
		return nil
	default:
		return localPreconditionError(ErrCmdAborted)
	}
}

var confirmCmdOrAbortFn = confirmCmdOrAbort

func formatConfirmationPrompt(prompt string, choiceLabel string) string {
	prompt = strings.TrimSpace(prompt)
	if !strings.Contains(prompt, "\n") {
		prompt = splitFinalConfirmationQuestion(prompt)
	}
	return fmt.Sprintf("%s %s: ", prompt, choiceLabel)
}

func splitFinalConfirmationQuestion(prompt string) string {
	if !strings.HasSuffix(prompt, "?") {
		return prompt
	}
	splitAt := -1
	for _, marker := range []string{". ", "; ", "! "} {
		if idx := strings.LastIndex(prompt, marker); idx > splitAt {
			splitAt = idx
		}
	}
	if splitAt < 0 {
		return prompt
	}
	return strings.TrimSpace(prompt[:splitAt+1]) + "\n" + strings.TrimSpace(prompt[splitAt+2:])
}

func shouldImplicitlyConfirm(cmd *cobra.Command) bool {
	return flagCmdAutoConfirm || automationModeEnabled(cmd)
}

func requireAutomationSupport(cmd *cobra.Command) error {
	if !automationModeEnabled(cmd) {
		return nil
	}
	if automationSupportForCommand(cmd) == AutomationSupportFull {
		return nil
	}

	message := fmt.Sprintf("%s does not support --automation", commandPath(cmd))
	if notes := automationNotesForCommand(cmd); notes != "" {
		message = fmt.Sprintf("%s: %s", message, notes)
	}

	return ferrors.WrapClass(
		ferrors.ErrUnsupported,
		fmt.Errorf("%s; remove --automation or inspect `c8volt capabilities --json` for supported commands", message),
	)
}

func mergeAndValidateKeys(baseKeys []string, stdinKeys []string, log *slog.Logger, cfg *config.Config) typex.Keys {
	keys := append([]string{}, baseKeys...)

	if len(stdinKeys) > 0 {
		if ok, firstBadKey, firstBadIndex := validateKeys(stdinKeys); !ok {
			if strings.HasPrefix(firstBadKey, "filter: ") {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes,
					invalidFlagValuef("validating keys from stdin failed: use --keys-only flag to get only keys as input"))
			}
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes,
				invalidFlagValuef("validating keys from stdin failed: line %q at index %d is not a valid key; have you forgotten to use --keys-only flag in case of c8volt commands?",
					firstBadKey, firstBadIndex))
		}
		keys = append(keys, stdinKeys...)
	}
	return keys
}
