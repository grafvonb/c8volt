// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// userTaskCompletionInput freezes local selection and payload before preview or
// confirmation. Nil variables means omitted; an empty nonnil map means supplied {}.
type userTaskCompletionInput struct {
	Keys          typex.Keys
	Variables     map[string]any
	WantedWorkers int
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

	keys, err := collectUserTaskMutationKeys(cmd, args, "completion")
	if err != nil {
		return userTaskCompletionInput{}, err
	}

	var variables map[string]any
	if cmd.Flags().Changed("vars") || cmd.Flags().Changed("vars-file") {
		raw, source, err := readUpdateVariablesFromFlags(cmd, flagUpdateUserTaskVars, flagUpdateUserTaskVarsFile)
		if err != nil {
			return userTaskCompletionInput{}, err
		}
		variables, err = parseUserTaskCompletionVariables(raw, source)
		if err != nil {
			return userTaskCompletionInput{}, err
		}
	}
	return userTaskCompletionInput{Keys: keys, Variables: variables, WantedWorkers: flagWorkers}, nil
}

// parseUserTaskCompletionVariables owns the decoded object and preserves JSON numbers
// before any float conversion. Existing variable-update decoding remains unchanged.
func parseUserTaskCompletionVariables(raw, source string) (map[string]any, error) {
	if raw == "" {
		return nil, invalidFlagValuef("--vars or --vars-file is required and must be a JSON object")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var variables map[string]any
	if err := decoder.Decode(&variables); err != nil {
		return nil, invalidFlagValuef("%s must be a valid JSON object: %v", source, err)
	}
	if variables == nil {
		return nil, invalidFlagValuef("%s must be a JSON object", source)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, invalidFlagValuef("%s must contain exactly one JSON object", source)
	}
	return variables, nil
}
