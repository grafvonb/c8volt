// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import "fmt"

func processInstanceBulkActivity(verb string, rootCount int, affectedCount int) string {
	if affectedCount > rootCount {
		return fmt.Sprintf("%s %d pi via %d root(s)", verb, affectedCount, rootCount)
	}
	return fmt.Sprintf("%s %d pi", verb, rootCount)
}
