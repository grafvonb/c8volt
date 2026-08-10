// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

const (
	opsSlowProcessAnalysisTimeExpectedFormat = "RFC3339 timestamp, c8volt timestamp YYYY-MM-DDTHH:MM:SS[.fraction], or YYYY-MM-DD"
	opsSlowProcessAnalysisTimeLayout         = "2006-01-02T15:04:05"
	opsSlowProcessAnalysisTimeFractionLayout = "2006-01-02T15:04:05.999999999"
)

// validateOpsSlowProcessAnalysisCommandArgs rejects invalid selector combinations before client setup.
func validateOpsSlowProcessAnalysisCommandArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 1 && args[0] != "-" {
		return invalidFlagValuef("unexpected positional argument %q; use '-' to read process-instance keys from stdin", args[0])
	}
	if len(args) > 1 {
		return invalidFlagValuef("unexpected positional arguments %v; use one '-' to read process-instance keys from stdin", args)
	}
	if flagOpsAnalyseSlowProcessInstanceBatchSize <= 0 || flagOpsAnalyseSlowProcessInstanceBatchSize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagOpsAnalyseSlowProcessInstanceBatchSize, consts.MaxPISearchSize)
	}
	if flagOpsAnalyseSlowProcessInstanceLimit < 0 || (flagOpsAnalyseSlowProcessInstanceLimit == 0 && cmd != nil && cmd.Flags().Changed("limit")) {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if _, err := parseOpsSlowProcessAnalysisRootDurationLonger(); err != nil {
		return err
	}
	if _, err := parseOpsSlowProcessAnalysisDetailDurationLonger(); err != nil {
		return err
	}
	if _, err := parseOpsSlowProcessAnalysisElementType(); err != nil {
		return err
	}
	if _, err := parseOpsSlowProcessAnalysisElementState(); err != nil {
		return err
	}
	if flagOpsAnalyseSlowProcessInstanceWithListeners && pickMode() == RenderModeKeysOnly {
		return mutuallyExclusiveFlagsf("--with-listeners cannot be combined with --keys-only")
	}
	stdinRequested := len(args) == 1 && args[0] == "-"
	keyedMode := len(flagOpsAnalyseSlowProcessInstanceKeys) > 0 || stdinRequested
	processDefinitionSelectorMode := flagOpsAnalyseSlowProcessInstanceBpmnProcessID != "" || flagOpsAnalyseSlowProcessInstancePDKey != ""
	if flagOpsAnalyseSlowProcessInstanceBpmnProcessID != "" && flagOpsAnalyseSlowProcessInstancePDKey != "" {
		return mutuallyExclusiveFlagsf("--bpmn-process-id cannot be combined with --pd-key")
	}
	if keyedMode && processDefinitionSelectorMode {
		return mutuallyExclusiveFlagsf("--key/stdin '-' cannot be combined with process-definition selectors")
	}
	if keyedMode && hasOpsSlowProcessAnalysisSearchFilterFlags(cmd) {
		return mutuallyExclusiveFlagsf("explicit process-instance keys cannot be combined with process-instance search filters")
	}
	if !keyedMode && !processDefinitionSelectorMode {
		return localPreconditionError(fmt.Errorf("select process instances with --key, stdin '-', --bpmn-process-id, or --pd-key"))
	}
	if _, err := parseOpsSlowProcessAnalysisState(); err != nil {
		return err
	}
	if _, _, err := parseOpsSlowProcessAnalysisDateRange("--start-date-after", flagOpsAnalyseSlowProcessInstanceStartDateAfter, "--start-date-before", flagOpsAnalyseSlowProcessInstanceStartDateBefore); err != nil {
		return err
	}
	if _, _, err := parseOpsSlowProcessAnalysisDateRange("--end-date-after", flagOpsAnalyseSlowProcessInstanceEndDateAfter, "--end-date-before", flagOpsAnalyseSlowProcessInstanceEndDateBefore); err != nil {
		return err
	}
	if ok, firstBadKey, _ := validateKeys(flagOpsAnalyseSlowProcessInstanceKeys); len(flagOpsAnalyseSlowProcessInstanceKeys) > 0 && !ok {
		return invalidFlagValuef("process-instance key %q is not a valid key", firstBadKey)
	}
	return nil
}

// parseOpsSlowProcessAnalysisElementType keeps detail type filters aligned with runtime element search values.
func parseOpsSlowProcessAnalysisElementType() (string, error) {
	if strings.TrimSpace(flagOpsAnalyseSlowProcessInstanceType) == "" {
		return "", nil
	}
	if !validElementType(flagOpsAnalyseSlowProcessInstanceType) {
		return "", invalidFlagValuef("invalid value for --type: %q, valid values are: %s", flagOpsAnalyseSlowProcessInstanceType, strings.Join(validElementTypes, ", "))
	}
	return normalizedElementType(flagOpsAnalyseSlowProcessInstanceType), nil
}

// parseOpsSlowProcessAnalysisElementState keeps detail state filters separate from the process-instance --state flag.
func parseOpsSlowProcessAnalysisElementState() (string, error) {
	if strings.TrimSpace(flagOpsAnalyseSlowProcessInstanceElementState) == "" {
		return "", nil
	}
	if !validElementState(flagOpsAnalyseSlowProcessInstanceElementState) {
		return "", invalidFlagValuef("invalid value for --element-state: %q, valid values are: %s", flagOpsAnalyseSlowProcessInstanceElementState, strings.Join(validElementStates, ", "))
	}
	return normalizedElementState(flagOpsAnalyseSlowProcessInstanceElementState), nil
}

// hasOpsSlowProcessAnalysisSearchFilterFlags identifies flags valid only for process-definition discovery.
func hasOpsSlowProcessAnalysisSearchFilterFlags(cmd *cobra.Command) bool {
	return flagOpsAnalyseSlowProcessInstanceStartDateAfter != "" ||
		flagOpsAnalyseSlowProcessInstanceStartDateBefore != "" ||
		flagOpsAnalyseSlowProcessInstanceEndDateAfter != "" ||
		flagOpsAnalyseSlowProcessInstanceEndDateBefore != "" ||
		flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly ||
		(cmd != nil && cmd.Flags().Changed("state")) ||
		(cmd != nil && cmd.Flags().Changed("batch-size")) ||
		(cmd != nil && cmd.Flags().Changed("limit"))
}

// parseOpsSlowProcessAnalysisState keeps accepted state tokens aligned with process-instance search.
func parseOpsSlowProcessAnalysisState() (process.State, error) {
	state, ok := process.ParseState(flagOpsAnalyseSlowProcessInstanceState)
	if !ok || state == process.StateAbsent {
		return "", invalidFlagValuef("invalid value for --state: %q, expected active, completed, canceled, terminated, or all", flagOpsAnalyseSlowProcessInstanceState)
	}
	if state == process.StateAll {
		return "", nil
	}
	return state, nil
}

// parseOpsSlowProcessAnalysisRootDurationLonger validates root process-instance duration filtering.
func parseOpsSlowProcessAnalysisRootDurationLonger() (time.Duration, error) {
	return parseOpsSlowProcessAnalysisDurationFlag("--dur-longer", flagOpsAnalyseSlowProcessInstanceDurationLonger)
}

// parseOpsSlowProcessAnalysisDetailDurationLonger validates detail duration filtering.
func parseOpsSlowProcessAnalysisDetailDurationLonger() (time.Duration, error) {
	return parseOpsSlowProcessAnalysisDurationFlag("--dur-element-longer", flagOpsAnalyseSlowProcessInstanceElementDurationLonger)
}

// parseOpsSlowProcessAnalysisDurationFlag validates Go duration syntax for ops thresholds.
func parseOpsSlowProcessAnalysisDurationFlag(flagName string, raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, invalidFlagValuef("invalid value for %s: %q, expected duration such as 500ms, 1s, 2m, or 1h", flagName, raw)
	}
	if value < 0 {
		return 0, invalidFlagValuef("%s must not be negative", flagName)
	}
	return value, nil
}

// parseOpsSlowProcessAnalysisDateRange validates and normalizes inclusive search bounds for process-instance discovery.
func parseOpsSlowProcessAnalysisDateRange(afterFlag string, afterValue string, beforeFlag string, beforeValue string) (string, string, error) {
	after, err := normalizeOpsSlowProcessAnalysisLowerBound(afterValue)
	if err != nil {
		return "", "", invalidFlagValuef("invalid value for %s: %q, expected %s", afterFlag, afterValue, opsSlowProcessAnalysisTimeExpectedFormat)
	}
	before, err := normalizeOpsSlowProcessAnalysisUpperBound(beforeValue)
	if err != nil {
		return "", "", invalidFlagValuef("invalid value for %s: %q, expected %s", beforeFlag, beforeValue, opsSlowProcessAnalysisTimeExpectedFormat)
	}
	if after != "" && before != "" {
		afterTime, err := time.Parse(time.RFC3339Nano, after)
		if err != nil {
			return "", "", err
		}
		beforeTime, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			return "", "", err
		}
		if afterTime.After(beforeTime) {
			return "", "", invalidFlagValuef("invalid range for %s and %s: %q is later than %q", afterFlag, beforeFlag, afterValue, beforeValue)
		}
	}
	return after, before, nil
}

// normalizeOpsSlowProcessAnalysisLowerBound preserves precise timestamps and expands date-only lower bounds to day start.
func normalizeOpsSlowProcessAnalysisLowerBound(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if t, ok := parseOpsSlowProcessAnalysisTimestamp(raw); ok {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	return "", fmt.Errorf("parse %q as process-instance date bound", raw)
}

// normalizeOpsSlowProcessAnalysisUpperBound preserves precise timestamps and expands date-only upper bounds to day end.
func normalizeOpsSlowProcessAnalysisUpperBound(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if t, ok := parseOpsSlowProcessAnalysisTimestamp(raw); ok {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	return "", fmt.Errorf("parse %q as process-instance date bound", raw)
}

// parseOpsSlowProcessAnalysisTimestamp accepts RFC3339 and c8volt's compact UTC timestamp forms.
func parseOpsSlowProcessAnalysisTimestamp(raw string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation(opsSlowProcessAnalysisTimeFractionLayout, raw, time.UTC); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation(opsSlowProcessAnalysisTimeLayout, raw, time.UTC); err == nil {
		return t, true
	}
	return time.Time{}, false
}
