// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"strings"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// userTaskCompletionInput freezes local selection and payload before preview or
// confirmation. Nil variables means omitted; an empty nonnil map means supplied {}.
type userTaskCompletionInput struct {
	Keys          typex.Keys
	Variables     map[string]any
	WantedWorkers int
}

// userTaskCompletionKeyValue remembers empty flag occurrences that pflag's
// StringSlice discards. It delegates all existing flag behavior, so variable mode
// remains unchanged. The completion runner must install it when registering keys.
type userTaskCompletionKeyValue struct {
	pflag.Value
	pflag.SliceValue
	emptyOccurrence bool
}

// Set records an explicit empty occurrence without altering legacy key values.
func (v *userTaskCompletionKeyValue) Set(raw string) error {
	if err := v.Value.Set(raw); err != nil {
		return err
	}
	v.emptyOccurrence = v.emptyOccurrence || raw == ""
	return nil
}

// Replace supports command flag reset helpers without retaining old evidence.
func (v *userTaskCompletionKeyValue) Replace(values []string) error {
	if err := v.SliceValue.Replace(values); err != nil {
		return err
	}
	v.emptyOccurrence = false
	return nil
}

// trackUserTaskCompletionKeyInput wraps an existing string-slice key flag before
// parsing. Only completion validation consults the extra occurrence evidence.
func trackUserTaskCompletionKeyInput(flag *pflag.Flag) {
	flag.Value = &userTaskCompletionKeyValue{Value: flag.Value, SliceValue: flag.Value.(pflag.SliceValue)}
}

// parseUpdateUserTaskCompletionInput validates the whole local request without
// constructing a client, reading backend state or changing variable-mode parsing.
func parseUpdateUserTaskCompletionInput(cmd *cobra.Command, args []string) (userTaskCompletionInput, error) {
	if err := validateOptionalDashArg(args); err != nil {
		return userTaskCompletionInput{}, err
	}
	if cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return userTaskCompletionInput{}, invalidFlagValuef("--workers must be positive integer")
	}
	if flagNoWait {
		return userTaskCompletionInput{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --no-wait")
	}
	if err := validateUpdateUserTaskJSONConfirmation(cmd); err != nil {
		return userTaskCompletionInput{}, err
	}

	if flag := cmd.Flags().Lookup("key"); flag != nil {
		if value, ok := flag.Value.(*userTaskCompletionKeyValue); ok && value.emptyOccurrence {
			return userTaskCompletionInput{}, invalidFlagValuef("--key requires a user task key for every occurrence")
		}
	}
	inputs := make([]string, 0, len(flagUpdateUserTaskKeys))
	for _, raw := range flagUpdateUserTaskKeys {
		inputs = append(inputs, strings.Split(raw, ",")...)
	}
	// pflag represents --key= as an empty slice, which must not disappear into
	// stdin selection. CSV empty components remain visible in nonempty slices.
	if cmd.Flags().Changed("key") && len(inputs) == 0 {
		return userTaskCompletionInput{}, invalidFlagValuef("--key requires a user task key")
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
			return userTaskCompletionInput{}, invalidFlagValuef("could not read user task keys from stdin")
		}
	}
	for i, raw := range inputs {
		inputs[i] = strings.TrimSpace(raw)
	}
	if ok, _, index := validateKeys(inputs); !ok {
		return userTaskCompletionInput{}, invalidFlagValuef("invalid user task key at position %d: expected exactly 16 decimal digits", index+1)
	}
	if len(inputs) == 0 {
		return userTaskCompletionInput{}, invalidFlagValuef("at least one user task key is required for completion")
	}

	var variables map[string]any
	if cmd.Flags().Changed("vars") || cmd.Flags().Changed("vars-file") {
		parsed, err := parseUpdateVariablesFromFlags(cmd, flagUpdateUserTaskVars, flagUpdateUserTaskVarsFile)
		if err != nil {
			return userTaskCompletionInput{}, err
		}
		variables, err = toolx.CopyJSONMap(parsed)
		if err != nil {
			return userTaskCompletionInput{}, invalidFlagValuef("completion variables must be a valid JSON object")
		}
	}
	return userTaskCompletionInput{Keys: typex.Keys(inputs).Unique(), Variables: variables, WantedWorkers: flagWorkers}, nil
}
