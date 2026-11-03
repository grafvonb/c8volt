// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

// flagMigration describes rejected syntax, never an accepted flag alias.
// Valueless restricts an existing action token without changing its Boolean metadata.
type flagMigration struct {
	Replacement string `json:"replacement"`
	TakesValue  bool   `json:"takes_value"`
	Valueless   bool   `json:"valueless,omitempty"`
}

const flagMigrationsAnnotation = "c8volt.flag-migrations"

// setFlagMigrations attaches diagnostic metadata without registering old flags.
func setFlagMigrations(cmd *cobra.Command, migrations map[string]flagMigration) {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	encoded, _ := json.Marshal(migrations)
	cmd.Annotations[flagMigrationsAnnotation] = string(encoded)
	// Cobra otherwise emits usage before ExecuteC returns an Args failure.
	cmd.SilenceUsage = true
}

// flagMigrationsForCommand keeps migration rules local to the selected leaf.
func flagMigrationsForCommand(cmd *cobra.Command) map[string]flagMigration {
	var migrations map[string]flagMigration
	if cmd != nil {
		_ = json.Unmarshal([]byte(cmd.Annotations[flagMigrationsAnnotation]), &migrations)
	}
	return migrations
}

// migrationInvocationKey carries a private copy of argv across Cobra's early exits.
type migrationInvocationKey struct{}

// migrationPreRunKey distinguishes early parse/Args failures from bootstrap/runtime failures.
type migrationPreRunKey struct{}

// migrationReportingError carries diagnostic-only mode/exit state to the single
// outer renderer. No recovered flag can enter a command runner.
type migrationReportingError struct {
	cause      error
	original   error
	mode       RenderMode
	noErrCodes bool
}

// Error exposes only the classified, bounded diagnostic prepared for rendering.
func (e *migrationReportingError) Error() string { return e.cause.Error() }

// Unwrap preserves the parser cause for inspection without rendering its values.
func (e *migrationReportingError) Unwrap() []error { return []error{e.cause, e.original} }

// executeCommandWithMigrationContext preserves argv and Cobra's selected leaf;
// ordinary execution still uses Cobra once, without a recovery replay.
func executeCommandWithMigrationContext(root *cobra.Command, args []string) (*cobra.Command, error) {
	ctx := root.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	root.SetContext(context.WithValue(ctx, migrationInvocationKey{}, append([]string(nil), args...)))
	root.SetArgs(args)
	leaf, err := root.ExecuteC()
	if err == nil || leaf == nil || len(flagMigrationsForCommand(leaf)) == 0 {
		return leaf, err
	}
	// Pre-run marks even configuration failures: they are not parse/Args errors.
	if started, _ := leaf.Context().Value(migrationPreRunKey{}).(bool); started {
		return leaf, err
	}
	return leaf, prepareMigrationReporting(leaf, args, err)
}

// migrationFlagError substitutes static replacement guidance only for a real
// removed token, not a flag-looking opaque value or a positional after --.
func migrationFlagError(cmd *cobra.Command, fallback error) error {
	if cmd.Context() == nil || len(flagMigrationsForCommand(cmd)) == 0 {
		return invalidInputError(fallback)
	}
	args, _ := cmd.Context().Value(migrationInvocationKey{}).([]string)
	_, migrationErr := migrationReportingInput(cmd, args)
	if migrationErr != nil {
		return migrationErr
	}
	return invalidInputError(fallback)
}

// migrationSemanticError marks static, command-owned migration diagnostics. Never
// wrap parser errors or messages containing supplied values with this marker.
type migrationSemanticError struct{ error }

// prepareMigrationReporting loads local reporting configuration only. Invalid
// local configuration uses the existing bootstrap fallback, with bounded text
// rather than exposing config contents, payloads, or the original argv.
func prepareMigrationReporting(cmd *cobra.Command, args []string, original error) error {
	reporting, migrationErr := migrationReportingInput(cmd, args)
	var semantic *migrationSemanticError
	if migrationErr == nil && errors.As(original, &semantic) {
		migrationErr = semantic.error
	}
	if migrationErr == nil {
		// Cobra parse/Args errors can include arbitrary supplied values. Keep the
		// original cause privately, with a static invalid-input diagnostic.
		migrationErr = invalidInputError(errors.New("invalid command arguments; see command help"))
	}
	mode := RenderModeOneLine
	if boolFlag(reporting, "json") {
		mode = RenderModeJSON
	} else if boolFlag(reporting, "keys-only") {
		mode = RenderModeKeysOnly
	}
	noErrCodes := boolFlag(reporting, "no-err-codes")
	v, _, cfg, err := resolveLocalCommandConfig(reporting, boolFlag(reporting, "quiet"), boolFlag(reporting, "debug"))
	if err == nil {
		err = cfg.Log.Validate()
	}
	if err == nil {
		ctx := localCommandLogContext(cmd, cfg, v, false)
		cmd.SetContext(ctx)
		noErrCodes = cfg.App.NoErrCodes
	} else {
		migrationErr = invalidInputError(errors.New(strings.TrimPrefix(migrationErr.Error(), "invalid input: ") + "; local diagnostic configuration could not be loaded"))
	}
	return &migrationReportingError{cause: migrationErr, original: original, mode: mode, noErrCodes: noErrCodes}
}

// boolFlag reads isolated reporting state without using command-global storage.
func boolFlag(cmd *cobra.Command, name string) bool {
	value, _ := cmd.Flags().GetBool(name)
	return value
}
