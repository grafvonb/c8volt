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
