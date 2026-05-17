// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"
)

func opsWorkflowElapsedSuffix(duration string) string {
	if duration == "" {
		return ""
	}
	parsed, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Sprintf("; elapsed: %s", duration)
	}
	return fmt.Sprintf("; elapsed: %s", opsWorkflowApproxDuration(parsed))
}

func opsWorkflowApproxDuration(duration time.Duration) string {
	if duration < 0 {
		duration = -duration
	}
	if duration == 0 {
		return "<1s"
	}
	if duration < time.Second {
		return "<1s"
	}
	return duration.Round(time.Second).String()
}

func opsWorkflowDeletionSummary(status string, count int, singular string, plural string, noWait bool) string {
	items := opsWorkflowCountLabel(count, singular, plural)
	switch status {
	case "confirmed":
		return "removed " + items
	case "submitted":
		if noWait {
			return "submitted " + items + " (--no-wait)"
		}
		return "submitted " + items
	default:
		return fmt.Sprintf("%s (%s)", status, items)
	}
}

func opsWorkflowCountLabel(count int, singular string, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
