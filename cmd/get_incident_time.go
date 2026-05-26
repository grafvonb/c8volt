// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"
)

const (
	incidentCreationTimeExpectedFormat = "RFC3339 timestamp, c8volt timestamp YYYY-MM-DDTHH:MM:SS[.fraction], or YYYY-MM-DD"
	incidentCreationTimeLayout         = "2006-01-02T15:04:05"
	incidentCreationTimeFractionLayout = "2006-01-02T15:04:05.999999999"
)

func validateIncidentCreationTimeFilters(afterFlag, afterValue, beforeFlag, beforeValue, newerFlag string, newerDays int, olderFlag string, olderDays int) error {
	if err := validateIncidentCreationTimeFlag(afterFlag, afterValue); err != nil {
		return err
	}
	if err := validateIncidentCreationTimeFlag(beforeFlag, beforeValue); err != nil {
		return err
	}
	if err := validateIncidentRelativeDayFlag(newerFlag, newerDays); err != nil {
		return err
	}
	if err := validateIncidentRelativeDayFlag(olderFlag, olderDays); err != nil {
		return err
	}
	if (afterValue != "" || beforeValue != "") && (newerDays >= 0 || olderDays >= 0) {
		return mutuallyExclusiveFlagsf("creation-time absolute and relative day filters cannot be combined")
	}
	after, err := pickIncidentCreationTimeLowerBound(afterValue, newerDays)
	if err != nil {
		return err
	}
	before, err := pickIncidentCreationTimeUpperBound(beforeValue, olderDays)
	if err != nil {
		return err
	}
	return validateIncidentCreationTimeRange(afterFlagForRange(afterFlag, newerFlag, newerDays), after, beforeFlagForRange(beforeFlag, olderFlag, olderDays), before)
}

func validateIncidentCreationTimeFlag(name string, value string) error {
	if value == "" {
		return nil
	}
	if _, err := normalizeIncidentCreationTimeLowerBound(value); err == nil {
		return nil
	}
	return invalidFlagValuef("invalid value for %s: %q, expected %s", name, value, incidentCreationTimeExpectedFormat)
}

func validateIncidentRelativeDayFlag(flagName string, value int) error {
	if value < 0 {
		if value == -1 {
			return nil
		}
		return invalidFlagValuef("invalid value for %s: %d, expected non-negative integer", flagName, value)
	}
	return nil
}

func validateIncidentCreationTimeRange(afterFlag, afterValue, beforeFlag, beforeValue string) error {
	if afterValue == "" || beforeValue == "" {
		return nil
	}
	after, err := time.Parse(time.RFC3339Nano, afterValue)
	if err != nil {
		return err
	}
	before, err := time.Parse(time.RFC3339Nano, beforeValue)
	if err != nil {
		return err
	}
	if after.After(before) {
		return invalidFlagValuef("invalid range for %s and %s: %q is later than %q", afterFlag, beforeFlag, afterValue, beforeValue)
	}
	return nil
}

func pickIncidentCreationTimeLowerBound(absolute string, relativeDays int) (string, error) {
	if absolute != "" {
		return normalizeIncidentCreationTimeLowerBound(absolute)
	}
	if relativeDays < 0 {
		return "", nil
	}
	return deriveIncidentCreationTimeLowerBound(relativeDays), nil
}

func pickIncidentCreationTimeUpperBound(absolute string, relativeDays int) (string, error) {
	if absolute != "" {
		return normalizeIncidentCreationTimeUpperBound(absolute)
	}
	if relativeDays < 0 {
		return "", nil
	}
	return deriveIncidentCreationTimeUpperBound(relativeDays), nil
}

func normalizeIncidentCreationTimeLowerBound(raw string) (string, error) {
	if t, ok := parseIncidentCreationTimestamp(raw); ok {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	return "", fmt.Errorf("parse %q as incident creation time", raw)
}

func normalizeIncidentCreationTimeUpperBound(raw string) (string, error) {
	if t, ok := parseIncidentCreationTimestamp(raw); ok {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	return "", fmt.Errorf("parse %q as incident creation time", raw)
}

func parseIncidentCreationTimestamp(raw string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation(incidentCreationTimeFractionLayout, raw, time.UTC); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation(incidentCreationTimeLayout, raw, time.UTC); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func deriveIncidentCreationTimeLowerBound(relativeDays int) string {
	day := relativeDayNow().AddDate(0, 0, -relativeDays).Format(time.DateOnly)
	t, _ := time.Parse(time.DateOnly, day)
	return t.UTC().Format(time.RFC3339Nano)
}

func deriveIncidentCreationTimeUpperBound(relativeDays int) string {
	day := relativeDayNow().AddDate(0, 0, -relativeDays).Format(time.DateOnly)
	t, _ := time.Parse(time.DateOnly, day)
	t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
	return t.UTC().Format(time.RFC3339Nano)
}

func afterFlagForRange(absoluteFlag, relativeFlag string, relativeDays int) string {
	if relativeDays >= 0 {
		return relativeFlag
	}
	return absoluteFlag
}

func beforeFlagForRange(absoluteFlag, relativeFlag string, relativeDays int) string {
	if relativeDays >= 0 {
		return relativeFlag
	}
	return absoluteFlag
}
