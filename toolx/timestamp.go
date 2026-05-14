// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import "time"

const NumericZoneTimestampLayout = "2006-01-02T15:04:05.999999999-07:00"

// FormatNumericZoneTime renders a time with a numeric zone offset.
func FormatNumericZoneTime(value time.Time) string {
	return value.Format(NumericZoneTimestampLayout)
}

// FormatNumericZoneTimestamp renders RFC3339 timestamps with numeric zone offsets.
func FormatNumericZoneTimestamp(value string) string {
	if value == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return t.Format(NumericZoneTimestampLayout)
}
