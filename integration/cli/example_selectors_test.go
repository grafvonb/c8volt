// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// Inspect original example text so fixture-substituted keys remain executable.
// Literal documentation keys cannot be assumed to exist in the selected cluster.
func hasHardCodedDemoSelector(command string) bool {
	if strings.Contains(command, "OrderProcess") {
		return true
	}
	args, err := shellFields(command)
	if err != nil {
		return false // The example parser reports malformed commands separately.
	}
	for i, arg := range args {
		flag, value, inline := strings.Cut(arg, "=")
		if flag != "-k" && flag != "--key" && flag != "--id" && !(strings.HasPrefix(flag, "--") && strings.HasSuffix(flag, "-key")) {
			continue
		}
		if !inline && i+1 < len(args) {
			value = args[i+1]
		}
		for _, key := range strings.Split(value, ",") {
			if _, err := strconv.ParseUint(strings.TrimSpace(key), 10, 64); err == nil {
				return true
			}
		}
	}
	return false
}

func shellFields(command string) ([]string, error) {
	var fields []string
	var b strings.Builder
	var quote rune
	escaped := false
	tokenStarted := false
	for _, r := range command {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\':
			tokenStarted = true
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
		case r == '\'' || r == '"':
			tokenStarted = true
			quote = r
		case r == ' ' || r == '\t':
			if tokenStarted {
				fields = append(fields, b.String())
				b.Reset()
				tokenStarted = false
			}
		default:
			tokenStarted = true
			b.WriteRune(r)
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in example: %s", command)
	}
	if tokenStarted {
		fields = append(fields, b.String())
	}
	return fields, nil
}

// These checks use only example text; no CLI subprocess or backend is started.
func TestExampleLiteralResourceKeyGuard(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command string
		blocked bool
	}{
		{"reported bulk example", "./c8volt update job --key 2251799870796844,2251799870796848 --key 2251799870796849 --complete --dry-run", true},
		{"older sample", "./c8volt get pi --key 2251799813685250", true},
		{"different key range", "./c8volt get pi --key 4503599627370497", true},
		{"inline quoted collection", `./c8volt update job --key="123,456" --complete --dry-run`, true},
		{"short flag", "./c8volt get pi -k 123", true},
		{"related instance key", "./c8volt get job --pi-key=123", true},
		{"resource id", "./c8volt get resource --id 123", true},
		{"later literal key", "./c8volt update job --key <job-key> --key 456 --complete --dry-run", true},
		{"owned placeholder", "./c8volt get pi --key <process-instance-key>", false},
		{"job placeholders", "./c8volt update job --key <job-key>,<job-key> --complete --dry-run", false},
		{"unrelated numbers", `./c8volt run pi --count 3 --vars '{"reference":"2251799813685250"}'`, false},
		{"sample process", "./c8volt get pd --bpmn-process-id OrderProcess", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasHardCodedDemoSelector(tc.command); got != tc.blocked {
				t.Fatalf("hasHardCodedDemoSelector(%q) = %v, want %v", tc.command, got, tc.blocked)
			}
		})
	}
}
