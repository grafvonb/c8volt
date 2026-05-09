// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var relativeDayNow = func() time.Time {
	return time.Now().UTC()
}

// validatePIKeyedModeDateFilters keeps date selectors scoped to search mode,
// where they can be represented in Operate filters. Direct keyed lookup must
// remain an exact-key operation with no hidden post-filtering.
func validatePIKeyedModeDateFilters(keyCount int) error {
	if keyCount > 0 && (hasPIDateFilterFlags() || hasPIRelativeDayFilterFlags()) {
		return mutuallyExclusiveFlagsf("date filters are only supported for list/search usage and cannot be combined with --key")
	}
	return nil
}

// validatePISearchFlags performs command-level validation that depends on the
// combined flag state. It catches invalid values, unsupported flag combinations,
// date-range mistakes, and selector dependencies before the command creates
// remote requests.
func validatePISearchFlags(cmds ...*cobra.Command) error {
	var cmd *cobra.Command
	if len(cmds) > 0 {
		cmd = cmds[0]
	}
	if flagGetPISize <= 0 || flagGetPISize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagGetPISize, consts.MaxPISearchSize)
	}
	if flagGetPILimit < 0 || (flagGetPILimit == 0 && isPILimitFlagChanged(cmd)) {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if flagGetPIState != "" && flagGetPIState != "all" {
		if _, ok := process.ParseState(flagGetPIState); !ok {
			return invalidFlagValuef("invalid value for --state: %q, valid values are: %s", flagGetPIState, process.ValidStateStrings())
		}
	}
	if flagGetPITotal {
		switch {
		case flagGetPILimit > 0:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --limit")
		case flagViewAsJson:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --json")
		case flagViewKeysOnly:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --keys-only")
		case flagGetPIWithIncidents:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --with-incidents")
		case flagGetPIWithVars:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --with-vars")
		}
	}
	if flagGetPIIncidentMessageLimit < 0 {
		return invalidFlagValuef("invalid value for --incident-message-limit: %d, expected non-negative integer", flagGetPIIncidentMessageLimit)
	}
	if err := validatePIIncidentStateFlag(flagGetPIIncidentState); err != nil {
		return err
	}
	if err := validatePIIncidentErrorTypeFlag(flagGetPIIncidentErrorType); err != nil {
		return err
	}
	if isPIIncidentMessageLimitFlagChanged(cmd) && !flagGetPIWithIncidents {
		return missingDependentFlagsf("--incident-message-limit requires --with-incidents")
	}
	if isPIIncidentStateFlagChanged(cmd) && !flagGetPIWithIncidents {
		return missingDependentFlagsf("--incident-state requires --with-incidents")
	}
	if isPIIncidentErrorTypeFlagChanged(cmd) && !flagGetPIWithIncidents {
		return missingDependentFlagsf("--incident-error-type requires --with-incidents")
	}
	if isPIIncidentErrorMessageFlagChanged(cmd) && !flagGetPIWithIncidents {
		return missingDependentFlagsf("--incident-error-message requires --with-incidents")
	}
	if flagGetPIVarValueLimit < 0 {
		return invalidFlagValuef("invalid value for --var-value-limit: %d, expected non-negative integer", flagGetPIVarValueLimit)
	}
	if isPIVarValueLimitFlagChanged(cmd) && !flagGetPIWithVars {
		return missingDependentFlagsf("--var-value-limit requires --with-vars")
	}
	if flagGetPIProcessDefinitionKey != "" &&
		(flagGetPIBpmnProcessID != "" ||
			flagGetPIProcessVersion != 0 ||
			flagGetPIProcessVersionTag != "") {
		return mutuallyExclusiveFlagsf("--pd-key is mutually exclusive with --bpmn-process-id, --pd-version, and --pd-version-tag")
	}
	if err := validatePIDateFlag("--start-date-after", flagGetPIStartDateAfter); err != nil {
		return err
	}
	if err := validatePIDateFlag("--start-date-before", flagGetPIStartDateBefore); err != nil {
		return err
	}
	if err := validatePIDateFlag("--end-date-after", flagGetPIEndDateAfter); err != nil {
		return err
	}
	if err := validatePIDateFlag("--end-date-before", flagGetPIEndDateBefore); err != nil {
		return err
	}
	if err := validatePIRelativeDayFlag("--start-date-older-days", flagGetPIStartAfterDays); err != nil {
		return err
	}
	if err := validatePIRelativeDayFlag("--start-date-newer-days", flagGetPIStartBeforeDays); err != nil {
		return err
	}
	if err := validatePIRelativeDayFlag("--end-date-older-days", flagGetPIEndAfterDays); err != nil {
		return err
	}
	if err := validatePIRelativeDayFlag("--end-date-newer-days", flagGetPIEndBeforeDays); err != nil {
		return err
	}
	if err := validatePIMixedDateFilterInputs(); err != nil {
		return err
	}
	if err := validatePIDateRange("--start-date-after", flagGetPIStartDateAfter, "--start-date-before", flagGetPIStartDateBefore); err != nil {
		return err
	}
	if err := validatePIDateRange("--end-date-after", flagGetPIEndDateAfter, "--end-date-before", flagGetPIEndDateBefore); err != nil {
		return err
	}
	if err := validatePIDateRange("--start-date-newer-days", pickPIDateBound("", flagGetPIStartBeforeDays), "--start-date-older-days", pickPIDateUpperBound("", flagGetPIStartAfterDays)); err != nil {
		return err
	}
	if err := validatePIDateRange("--end-date-newer-days", pickPIDateBound("", flagGetPIEndBeforeDays), "--end-date-older-days", pickPIDateUpperBound("", flagGetPIEndAfterDays)); err != nil {
		return err
	}
	if flagGetPIBpmnProcessID == "" &&
		(flagGetPIProcessVersion != 0 || flagGetPIProcessVersionTag != "") {
		return missingDependentFlagsf("--pd-version and --pd-version-tag require --bpmn-process-id to be set")
	}
	if flagGetPIChildrenOnly && flagGetPIRootsOnly {
		return forbiddenFlagCombinationf("using both --children-only and --roots-only filters returns does not make sense")
	}
	if activeIncidentFilterCount() > 1 {
		return forbiddenFlagCombinationf("using --incidents-only, --direct-incidents-only, and --no-incidents-only together does not make sense")
	}
	return nil
}

// isPILimitFlagChanged distinguishes an omitted --limit from an explicit zero,
// which should be rejected because the flag is defined as a positive boundary.
func isPILimitFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("limit")
}

// isPIIncidentMessageLimitFlagChanged detects whether the incident truncation
// flag was supplied so validation can require --with-incidents only when the
// operator actually requested that dependent behavior.
func isPIIncidentMessageLimitFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("incident-message-limit")
}

func isPIIncidentStateFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("incident-state")
}

func isPIIncidentErrorTypeFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("incident-error-type")
}

func isPIIncidentErrorMessageFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("incident-error-message")
}

func validatePIIncidentStateFlag(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active", "pending", "resolved", "migrated", "unknown", "all":
		return nil
	default:
		return invalidFlagValuef("invalid value for --incident-state: %q, valid values are: active, pending, resolved, migrated, unknown, all", value)
	}
}

func validatePIIncidentErrorTypeFlag(value string) error {
	if _, ok := incidentfilter.NormalizeErrorType(value); ok {
		return nil
	}
	return invalidFlagValuef("invalid value for --incident-error-type: %q, valid values are: %s", value, incidentfilter.ValidErrorTypesString())
}

// isPIVarValueLimitFlagChanged detects whether variable truncation was supplied
// so validation can reject a dangling --var-value-limit without --with-vars.
func isPIVarValueLimitFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("var-value-limit")
}

// validatePIKeyedModeLimit prevents --limit from implying that direct keyed
// lookup is a search. Multiple --key values already define the exact requested
// set, so a second local cap would be ambiguous.
func validatePIKeyedModeLimit(keyCount int) error {
	if keyCount > 0 && flagGetPILimit > 0 {
		return mutuallyExclusiveFlagsf("--limit cannot be combined with --key")
	}
	return nil
}

// validatePIWithIncidentsUsage keeps incident enrichment out of modes that cannot attach details unambiguously.
func validatePIWithIncidentsUsage(cmd *cobra.Command, keyCount int, filterFlagsSet bool) error {
	if isPIIncidentStateFlagChanged(cmd) && keyCount == 0 {
		return missingDependentFlagsf("--incident-state is only supported with --key for process-instance incident lookup; list/search process-instance results use the active hasIncident marker")
	}
	if !flagGetPIWithIncidents {
		return nil
	}
	if keyCount > 0 && (filterFlagsSet || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPIDirectIncidentsOnly || flagGetPINoIncidentsOnly || flagGetPITotal) {
		return mutuallyExclusiveFlagsf("--with-incidents cannot be combined with search-mode filters")
	}
	return nil
}

// validatePIWithVarsUsage keeps variable enrichment out of keyed/filter combinations that cannot attach details unambiguously.
func validatePIWithVarsUsage(keyCount int, filterFlagsSet bool) error {
	if !flagGetPIWithVars {
		return nil
	}
	if keyCount > 0 && (filterFlagsSet || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPIDirectIncidentsOnly || flagGetPINoIncidentsOnly || flagGetPITotal) {
		return mutuallyExclusiveFlagsf("--with-vars cannot be combined with search-mode filters")
	}
	return nil
}

func activeIncidentFilterCount() int {
	count := 0
	if flagGetPIIncidentsOnly {
		count++
	}
	if flagGetPIDirectIncidentsOnly {
		count++
	}
	if flagGetPINoIncidentsOnly {
		count++
	}
	return count
}

// useInvalidInputFlagErrors maps Cobra flag parsing failures into the command's
// invalid-input error class so bad flag syntax follows the same exit behavior as
// semantic validation failures.
func useInvalidInputFlagErrors(cmd *cobra.Command) {
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return silenceUsageForError(cmd, invalidInputError(err))
	})
}

// validatePISearchVersionSupport rejects search filters whose behavior is not
// available or tenant-safe on the configured Camunda version.
func validatePISearchVersionSupport(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	if (hasPIDateFilterFlags() || hasPIRelativeDayFilterFlags()) && cfg.App.CamundaVersion == toolx.V87 {
		return ferrors.WrapClass(ferrors.ErrUnsupported,
			fmt.Errorf("process-instance date filters require Camunda 8.8"))
	}
	if flagGetPIOrphanChildrenOnly && cfg.App.CamundaVersion == toolx.V87 {
		return ferrors.WrapClass(ferrors.ErrUnsupported,
			fmt.Errorf("--orphan-children-only is not supported in Camunda 8.7 because orphan-parent follow-up lookup is not tenant-safe"))
	}
	return nil
}

// validatePIMixedDateFilterInputs rejects using absolute dates and relative-day
// selectors for the same bound family. Keeping one style per start/end range
// avoids silently choosing one operator intent over another.
func validatePIMixedDateFilterInputs() error {
	if hasStartPIAbsoluteDateFlags() && hasStartPIRelativeDayFlags() {
		return mutuallyExclusiveFlagsf("start-date absolute and relative day filters cannot be combined")
	}
	if hasEndPIAbsoluteDateFlags() && hasEndPIRelativeDayFlags() {
		return mutuallyExclusiveFlagsf("end-date absolute and relative day filters cannot be combined")
	}
	return nil
}

// hasStartPIAbsoluteDateFlags reports whether either absolute start-date bound
// participates in validation.
func hasStartPIAbsoluteDateFlags() bool {
	return flagGetPIStartDateAfter != "" || flagGetPIStartDateBefore != ""
}

// hasEndPIAbsoluteDateFlags reports whether either absolute end-date bound
// participates in validation.
func hasEndPIAbsoluteDateFlags() bool {
	return flagGetPIEndDateAfter != "" || flagGetPIEndDateBefore != ""
}

// hasStartPIRelativeDayFlags reports whether either relative start-date bound
// was set by the operator.
func hasStartPIRelativeDayFlags() bool {
	return flagGetPIStartAfterDays >= 0 || flagGetPIStartBeforeDays >= 0
}

// hasEndPIRelativeDayFlags reports whether either relative end-date bound was
// set by the operator.
func hasEndPIRelativeDayFlags() bool {
	return flagGetPIEndAfterDays >= 0 || flagGetPIEndBeforeDays >= 0
}

// validatePIDateFlag enforces the YYYY-MM-DD contract accepted by Operate date
// filters before the value is embedded in a search request.
func validatePIDateFlag(flagName, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(time.DateOnly, value); err != nil {
		return invalidFlagValuef("invalid value for %s: %q, expected YYYY-MM-DD", flagName, value)
	}
	return nil
}

// validatePIRelativeDayFlag accepts the internal unset sentinel while rejecting
// negative operator values. Zero is valid and means the current UTC day.
func validatePIRelativeDayFlag(flagName string, value int) error {
	if value < 0 {
		if value == -1 {
			return nil
		}
		return invalidFlagValuef("invalid value for %s: %d, expected non-negative integer", flagName, value)
	}
	return nil
}

// validatePIDateRange protects lower/upper date pairs from being reversed after
// absolute values or resolved relative-day values have been normalized.
func validatePIDateRange(afterFlag, afterValue, beforeFlag, beforeValue string) error {
	if afterValue == "" || beforeValue == "" {
		return nil
	}

	after, err := time.Parse(time.DateOnly, afterValue)
	if err != nil {
		return err
	}
	before, err := time.Parse(time.DateOnly, beforeValue)
	if err != nil {
		return err
	}
	if after.After(before) {
		return invalidFlagValuef("invalid range for %s and %s: %q is later than %q", afterFlag, beforeFlag, afterValue, beforeValue)
	}
	return nil
}

// pickPIDateBound chooses an explicit absolute lower bound when provided, or
// derives one from a relative-day selector. An unset relative selector leaves the
// bound empty so the service omits it.
func pickPIDateBound(absolute string, relativeDays int) string {
	if absolute != "" {
		return absolute
	}
	if relativeDays < 0 {
		return ""
	}
	return derivePIDateBound(relativeDays)
}

// pickPIDateUpperBound mirrors pickPIDateBound for upper bounds. It exists as a
// separate function so future end-of-day behavior can change without touching
// lower-bound resolution.
func pickPIDateUpperBound(absolute string, relativeDays int) string {
	if absolute != "" {
		return absolute
	}
	if relativeDays < 0 {
		return ""
	}
	return derivePIUpperDateBound(relativeDays)
}

// derivePIDateBound converts a relative age in days to the concrete date string
// passed to Operate for lower-bound comparisons.
func derivePIDateBound(relativeDays int) string {
	return relativeDayNow().AddDate(0, 0, -relativeDays).Format(time.DateOnly)
}

// derivePIUpperDateBound converts a relative age in days to the concrete date
// string passed to Operate for upper-bound comparisons.
func derivePIUpperDateBound(relativeDays int) string {
	return relativeDayNow().AddDate(0, 0, -relativeDays).Format(time.DateOnly)
}
