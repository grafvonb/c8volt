// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"time"
)

// runtimeElementDuration returns a display duration only when element timestamps prove it.
func runtimeElementDuration(startDate string, endDate string, state string, capturedNow time.Time) string {
	start, err := time.Parse(time.RFC3339Nano, startDate)
	if err != nil || start.IsZero() {
		return ""
	}
	var end time.Time
	switch {
	case endDate != "":
		end, err = time.Parse(time.RFC3339Nano, endDate)
		if err != nil || end.Before(start) {
			return ""
		}
	case strings.EqualFold(state, "ACTIVE"):
		end = capturedNow
		if end.Before(start) {
			return ""
		}
	default:
		return ""
	}
	return end.Sub(start).String()
}
