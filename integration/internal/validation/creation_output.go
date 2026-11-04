// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// HumanCreationKeys validates logged creation results without confusing them
// with selected-format stdout, supporting the configured plain/JSON logger.
func HumanCreationKeys(stdout, stderr string, wantCount int) ([]string, error) {
	if stdout != "" {
		return nil, fmt.Errorf("human creation results must use the configured logger, got stdout %q", stdout)
	}
	var keys []string
	seen := map[string]bool{}
	summaries := 0
	for _, line := range strings.Split(stderr, "\n") {
		var record struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
		}
		if json.Unmarshal([]byte(line), &record) != nil || record.Level == "" {
			fields := strings.Fields(line)
			if len(fields) < 3 || fields[1] != "INFO" {
				continue
			}
			record.Level, record.Msg = fields[1], strings.Join(fields[2:], " ")
		}
		if record.Level != "INFO" {
			continue
		}
		if record.Msg == fmt.Sprintf("found: %d", wantCount) {
			summaries++
		}
		fields := strings.Fields(record.Msg)
		if len(fields) < 2 {
			continue
		}
		if key, err := strconv.ParseUint(fields[0], 10, 64); err == nil && key > 0 {
			if seen[fields[0]] {
				return nil, fmt.Errorf("duplicate logged creation key %s", fields[0])
			}
			seen[fields[0]] = true
			keys = append(keys, fields[0])
		}
	}
	if len(keys) != wantCount || summaries != 1 {
		return nil, fmt.Errorf("logged creation output has %d keys and %d summaries, want %d keys and one summary", len(keys), summaries, wantCount)
	}
	return keys, nil
}
