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
