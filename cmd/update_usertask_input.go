// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"strings"

	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// userTaskKeyValue remembers empty flag occurrences that pflag's
// StringSlice discards. It delegates all existing flag behavior, so variable mode
// remains unchanged. Install it when registering keys before parsing.
type userTaskKeyValue struct {
	pflag.Value
	pflag.SliceValue
	emptyOccurrence bool
}

// Set records an explicit empty occurrence without altering legacy key values.
func (v *userTaskKeyValue) Set(raw string) error {
	if err := v.Value.Set(raw); err != nil {
		return err
	}
	v.emptyOccurrence = v.emptyOccurrence || raw == ""
	return nil
}

// Replace supports command flag reset helpers without retaining old evidence.
func (v *userTaskKeyValue) Replace(values []string) error {
	if err := v.SliceValue.Replace(values); err != nil {
		return err
	}
	v.emptyOccurrence = false
	return nil
}

// trackUserTaskKeyInput wraps an existing string-slice key flag before
// parsing. Only strict mutation input validation consults the extra occurrence evidence.
func trackUserTaskKeyInput(flag *pflag.Flag) {
	flag.Value = &userTaskKeyValue{Value: flag.Value, SliceValue: flag.Value.(pflag.SliceValue)}
}

// collectUserTaskMutationKeys freezes strict flag-first/stdin-next selection.
// Callers validate positional arguments first; variable mode keeps its legacy parser.
func collectUserTaskMutationKeys(cmd *cobra.Command, args []string, purpose string) (typex.Keys, error) {
	flagKeys, err := cmd.Flags().GetStringSlice("key")
	if err != nil {
		return nil, invalidFlagValuef("could not read user task key flags")
	}
	if flag := cmd.Flags().Lookup("key"); flag != nil {
		if value, ok := flag.Value.(*userTaskKeyValue); ok && value.emptyOccurrence {
			return nil, invalidFlagValuef("--key requires a user task key for every occurrence")
		}
	}
	inputs := make([]string, 0, len(flagKeys))
	for _, raw := range flagKeys {
		inputs = append(inputs, strings.Split(raw, ",")...)
	}
	// pflag represents --key= as an empty slice, which must not disappear into
	// stdin selection. CSV empty components remain visible in nonempty slices.
	if cmd.Flags().Changed("key") && len(inputs) == 0 {
		return nil, invalidFlagValuef("--key requires a user task key")
	}
	if len(args) == 1 {
		scanner := bufio.NewScanner(cmd.InOrStdin())
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			if key := strings.TrimSpace(scanner.Text()); key != "" {
				inputs = append(inputs, key)
			}
		}
		if scanner.Err() != nil {
			return nil, invalidFlagValuef("could not read user task keys from stdin")
		}
	}
	for i, raw := range inputs {
		inputs[i] = strings.TrimSpace(raw)
	}
	if ok, _, index := validateKeys(inputs); !ok {
		return nil, invalidFlagValuef("invalid user task key at position %d: expected exactly 16 decimal digits", index+1)
	}
	if len(inputs) == 0 {
		return nil, invalidFlagValuef("at least one user task key is required for %s", purpose)
	}

	return typex.Keys(inputs).Unique(), nil
}
