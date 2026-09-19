// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"time"

	"github.com/grafvonb/c8volt/toolx"
)

// listenerTimestampColumns keeps optional listener times in stable s:, e:, d: positions for aligned rows.
func listenerTimestampColumns(creationTime *time.Time, endTime *time.Time, deadline *time.Time, state string, showTimezoneOffset bool) flatRow {
	columns := flatRow{"", "", ""}
	if creationTime != nil {
		columns[0] = "s:" + toolx.FormatTime(*creationTime, showTimezoneOffset)
	}
	if endTime != nil {
		columns[1] = "e:" + toolx.FormatTime(*endTime, showTimezoneOffset)
	}
	if state == "ACTIVATED" && deadline != nil {
		columns[2] = "d:" + toolx.FormatTime(*deadline, showTimezoneOffset)
	}
	return columns
}

// runtimeListenerDuration measures job lifetime, including time waiting for a worker.
func runtimeListenerDuration(creationTime, endTime *time.Time, state string, capturedNow time.Time) string {
	if creationTime == nil || creationTime.IsZero() {
		return ""
	}
	end := capturedNow
	if endTime != nil {
		end = *endTime
	} else {
		switch state {
		case "CREATED", "ACTIVATED", "FAILED", "TIMED_OUT", "RETRIES_UPDATED":
		default:
			return ""
		}
	}
	if end.Before(*creationTime) {
		return ""
	}
	return end.Sub(*creationTime).String()
}
