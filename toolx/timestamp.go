// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import "time"

const (
	TimestampLayout            = "2006-01-02T15:04:05.000"
	NumericZoneTimestampLayout = "2006-01-02T15:04:05.000-07:00"
)

// FormatTime renders a time with fixed millisecond precision.
func FormatTime(value time.Time, showTimezoneOffset bool) string {
	if showTimezoneOffset {
		return value.Format(NumericZoneTimestampLayout)
	}
	return value.Format(TimestampLayout)
}

// FormatTimestamp renders RFC3339 timestamps with fixed millisecond precision.
func FormatTimestamp(value string, showTimezoneOffset bool) string {
	if value == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return FormatTime(t, showTimezoneOffset)
}

// FormatNumericZoneTime renders a time with a numeric zone offset.
func FormatNumericZoneTime(value time.Time) string {
	return FormatTime(value, true)
}

// FormatNumericZoneTimestamp renders RFC3339 timestamps with numeric zone offsets.
func FormatNumericZoneTimestamp(value string) string {
	return FormatTimestamp(value, true)
}
