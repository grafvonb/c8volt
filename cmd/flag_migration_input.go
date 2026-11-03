// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// migrationReportingFlags bounds recovery to local configuration and output.
// Resource inputs (including variable files) are inspected for arity only.
var migrationReportingFlags = map[string]bool{
	"config": true, "profile": true, "log-level": true, "log-format": true,
	"log-with-source": true, "quiet": true, "debug": true, "json": true,
	"keys-only": true, "no-err-codes": true,
	"timeout": true, "request-rate": true, "tenant": true, "camunda-version": true,
	"no-version-check": true, "automation": true, "auto-confirm": true,
	"backoff-timeout": true, "backoff-max-retries": true,
}

// migrationReportingInput walks tokens using registered arity, collecting only
// reporting controls into fresh values. Removed flags supply diagnostic arity;
// they are never added to either the real or temporary command's flag set.
func migrationReportingInput(cmd *cobra.Command, args []string) (*cobra.Command, error) {
	reporting := &cobra.Command{Use: cmd.Use}
	known := pflag.NewFlagSet("migration-arity", pflag.ContinueOnError)
	known.AddFlagSet(cmd.Flags())
	known.AddFlagSet(cmd.InheritedFlags())
	known.AddFlagSet(cmd.PersistentFlags())
	known.VisitAll(func(flag *pflag.Flag) {
		if !migrationReportingFlags[flag.Name] {
			return
		}
		if flag.Value.Type() == "bool" {
			value, _ := strconv.ParseBool(flag.DefValue)
			reporting.Flags().Bool(flag.Name, value, "")
		} else {
			reporting.Flags().String(flag.Name, flag.DefValue, "")
		}
	})
	migrations := flagMigrationsForCommand(cmd)
	var rejected error
	save := func(flag *pflag.Flag, value string) {
		if migrationReportingFlags[flag.Name] {
			_ = reporting.Flags().Set(flag.Name, value)
		}
	}
	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "--" {
			break
		}
		if strings.HasPrefix(token, "--") {
			name, value, attached := strings.Cut(token[2:], "=")
			flag := known.Lookup(name)
			if flag == nil {
				if migration, ok := migrations[name]; ok {
					if rejected == nil {
						rejected = invalidInputError(&removedFlagError{name: name, replacement: migration.Replacement})
					}
					if migration.TakesValue && !attached && i+1 < len(args) {
						i++
					}
				} else {
					// Unknown arity cannot safely distinguish a value from a reporting flag.
					break
				}
				continue
			}
			if !attached {
				if flag.NoOptDefVal != "" {
					value = flag.NoOptDefVal
				} else if i+1 < len(args) {
					i++
					value = args[i]
				} else {
					break
				}
			}
			save(flag, value)
			continue
		}
		if len(token) < 2 || token[0] != '-' {
			continue
		}
		// pflag clusters booleans; a value-taking shorthand consumes the remainder
		// (with optional '=') or the following token even when it looks like a flag.
		for short := token[1:]; len(short) > 0; {
			flag := known.ShorthandLookup(short[:1])
			short = short[1:]
			if flag == nil {
				return reporting, rejected
			}
			value := flag.NoOptDefVal
			if strings.HasPrefix(short, "=") {
				value = short[1:]
				short = ""
			} else if value == "" {
				if short != "" {
					value = short
					short = ""
				} else if i+1 < len(args) {
					i++
					value = args[i]
				}
			}
			save(flag, value)
		}
	}
	return reporting, rejected
}

// removedFlagError contains static metadata only, never a supplied value.
type removedFlagError struct{ name, replacement string }

// Error directs the caller to canonical syntax without reproducing payloads.
func (e *removedFlagError) Error() string {
	return "--" + e.name + " was removed; use " + e.replacement
}
