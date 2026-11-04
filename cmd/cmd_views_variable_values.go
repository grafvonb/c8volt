// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
)

// variableValueHumanLine formats one received variable with an explicit
// presentation limit so unrelated commands do not share flag state.
func variableValueHumanLine(variable process.ProcessInstanceVariable, valueLimit int) string {
	value := compactVariableValue(variable.Value)
	value, cliTruncated := truncateVariableHumanValue(value, valueLimit)
	labels := variableTruncationLabels(variable.APITruncated, cliTruncated)
	if labels != "" {
		return fmt.Sprintf("%s=%s [%s]", variable.Name, value, labels)
	}
	return fmt.Sprintf("%s=%s", variable.Name, value)
}

// compactVariableValue JSON-compacts object and array values while leaving
// scalar, empty, and malformed values unchanged.
func compactVariableValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return value
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(trimmed)); err != nil {
		return value
	}
	return buf.String()
}

// truncateVariableHumanValue applies a Unicode-rune display limit and reports
// whether presentation, independently of the backend, shortened the value.
func truncateVariableHumanValue(value string, limit int) (string, bool) {
	if limit <= 0 {
		return value, false
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value, false
	}
	return string(runes[:limit]) + "...", true
}

// variableTruncationLabels distinguishes backend and presentation shortening.
func variableTruncationLabels(apiTruncated bool, cliTruncated bool) string {
	switch {
	case apiTruncated && cliTruncated:
		return "api-truncated,cli-truncated"
	case apiTruncated:
		return "api-truncated"
	case cliTruncated:
		return "cli-truncated"
	default:
		return ""
	}
}
