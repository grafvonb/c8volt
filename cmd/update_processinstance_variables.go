// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/grafvonb/c8volt/c8volt/process"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// parseUpdateProcessInstanceVariablesFromFlags selects exactly one variable payload source and decodes it.
func parseUpdateProcessInstanceVariablesFromFlags(cmd *cobra.Command, raw string, filePath string) (map[string]any, error) {
	varsChanged := cmd.Flags().Changed("vars")
	varsFileChanged := cmd.Flags().Changed("vars-file")
	if varsChanged && varsFileChanged {
		return nil, mutuallyExclusiveFlagsf("--vars cannot be combined with --vars-file")
	}
	if varsFileChanged {
		if filePath == "" {
			return nil, invalidFlagValuef("--vars-file requires a file path")
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, invalidFlagValuef("--vars-file could not be read: %v", err)
		}
		return parseUpdateProcessInstanceVariables(string(data), "--vars-file")
	}
	return parseUpdateProcessInstanceVariables(raw, "--vars")
}

// parseUpdateProcessInstanceVariables decodes the --vars JSON object used for process-instance updates.
func parseUpdateProcessInstanceVariables(raw string, source string) (map[string]any, error) {
	if raw == "" {
		return nil, invalidFlagValuef("--vars or --vars-file is required and must be a JSON object")
	}
	var variables map[string]any
	if err := json.Unmarshal([]byte(raw), &variables); err != nil {
		return nil, invalidFlagValuef("%s must be a valid JSON object: %v", source, err)
	}
	if variables == nil {
		return nil, invalidFlagValuef("%s must be a JSON object", source)
	}
	return variables, nil
}

// validateUpdateProcessInstanceJSONConfirmation keeps machine-readable mutation output free of prompts and human plans.
func validateUpdateProcessInstanceJSONConfirmation(cmd *cobra.Command) error {
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for update pi")
	}
	if flagDryRun || pickMode() != RenderModeJSON || shouldImplicitlyConfirm(cmd) {
		return nil
	}
	return missingDependentFlagsf("--json update pi requires --dry-run, --auto-confirm, or --automation")
}

// planUpdateProcessInstanceVariables loads current process-scope variables and computes the requested update preview.
func planUpdateProcessInstanceVariables(ctx context.Context, cmd *cobra.Command, cli process.API, keys types.Keys, requested map[string]any) (processInstanceVariableUpdatePreview, error) {
	uniqueKeys := keys.Unique()
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("preparing update variable plan for %d process instance(s)", len(uniqueKeys)))
	defer stopActivity()

	plans := make([]processInstanceVariableUpdatePlan, 0, len(uniqueKeys))
	for _, key := range uniqueKeys {
		existing, err := cli.SearchProcessInstanceVariables(ctx, key, collectOptions()...)
		if err != nil {
			return processInstanceVariableUpdatePreview{}, fmt.Errorf("load variables for process-instance %s: %w", key, err)
		}
		plans = append(plans, newProcessInstanceVariableUpdatePlan(key, existing, requested))
	}
	return newProcessInstanceVariableUpdatePreview(uniqueKeys, plans), nil
}

// newProcessInstanceVariableUpdatePlan compares existing process-scope variables with the requested payload.
func newProcessInstanceVariableUpdatePlan(key string, existing []process.ProcessInstanceVariable, requested map[string]any) processInstanceVariableUpdatePlan {
	currentByName := processScopeVariableValuesByName(key, existing)
	requestedNames := sortedMapKeys(requested)

	plan := processInstanceVariableUpdatePlan{ProcessInstanceKey: key}
	for _, name := range requestedNames {
		after := requested[name]
		before, ok := currentByName[name]
		switch {
		case !ok:
			plan.Additions = append(plan.Additions, processInstanceVariablePlannedValue{Name: name, Value: after})
		case reflect.DeepEqual(before.Value, after):
			plan.UnchangedRequested = append(plan.UnchangedRequested, processInstanceVariablePlannedValue{Name: name, Value: after, APITruncated: before.APITruncated})
		default:
			plan.Changes = append(plan.Changes, processInstanceVariablePlannedChange{Name: name, Before: before.Value, After: after, APITruncated: before.APITruncated})
		}
		delete(currentByName, name)
	}

	untouchedNames := sortedMapKeys(currentByName)
	for _, name := range untouchedNames {
		current := currentByName[name]
		plan.Untouched = append(plan.Untouched, processInstanceVariablePlannedValue{Name: name, Value: current.Value, APITruncated: current.APITruncated})
	}
	return plan
}

type processInstanceVariableCurrentValue struct {
	Value        any
	APITruncated bool
}

func processScopeVariableValuesByName(key string, variables []process.ProcessInstanceVariable) map[string]processInstanceVariableCurrentValue {
	out := make(map[string]processInstanceVariableCurrentValue, len(variables))
	for _, variable := range variables {
		if variable.ProcessInstanceKey != "" && variable.ProcessInstanceKey != key {
			continue
		}
		if variable.ScopeKey != "" && variable.ScopeKey != key {
			continue
		}
		out[variable.Name] = processInstanceVariableCurrentValue{
			Value:        decodeProcessInstanceVariableValue(variable.Value),
			APITruncated: variable.APITruncated,
		}
	}
	return out
}

func decodeProcessInstanceVariableValue(raw string) any {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	return value
}

func sortedMapKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
