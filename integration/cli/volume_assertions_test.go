// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var humanLeakPattern = regexp.MustCompile(`(?i)(continue\?|spinner|progress|warning:|fetched [0-9]+ .*continue)`)

func requireVolumeCommandSuccess(result commandResult, label string) error {
	if result.Err != nil {
		return fmt.Errorf("%s failed: %v; stderr: %s", label, result.Err, strings.TrimSpace(result.Stderr))
	}
	return nil
}

func requireVolumeJSON(output string) error {
	if err := validateJSONString(output); err != nil {
		return err
	}
	return nil
}

func requireVolumeKeysOnly(output string) error {
	return validateKeysOnlyString(output)
}

func requireMachineStdoutClean(output string) error {
	if !json.Valid([]byte(strings.TrimSpace(output))) && validateKeysOnlyString(output) != nil {
		return nil
	}
	if humanLeakPattern.MatchString(output) {
		return fmt.Errorf("machine stdout contains human/progress text: %q", compactLogSnippet(output, 300))
	}
	return nil
}

func requireFinalOutcomeText(output string) error {
	normalized := strings.ToLower(output)
	for _, token := range []string{"succeeded", "completed", "created", "updated", "cancelled", "deleted", "resolved", "repaired", "purged", "exported", "found"} {
		if strings.Contains(normalized, token) {
			return nil
		}
	}
	return fmt.Errorf("output does not contain explicit final outcome wording: %q", compactLogSnippet(output, 300))
}

func requireNoWaitOrSubmittedText(output string) error {
	normalized := strings.ToLower(output)
	for _, token := range []string{"no-wait", "submitted", "accepted", "verification skipped"} {
		if strings.Contains(normalized, token) {
			return nil
		}
	}
	return fmt.Errorf("output does not contain no-wait/submitted wording: %q", compactLogSnippet(output, 300))
}

func requireResultContainsOneOf(t *testing.T, result commandResult, values ...string) {
	t.Helper()
	combined := result.Stdout + "\n" + result.Stderr
	for _, value := range values {
		if strings.Contains(combined, value) {
			return
		}
	}
	t.Fatalf("result did not contain any of %q\nstdout:\n%s\nstderr:\n%s", values, result.Stdout, result.Stderr)
}

func observedProcessInstanceKeys(items []seededProcessInstance) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		if item.Key != "" {
			keys = append(keys, item.Key)
		}
	}
	return keys
}
