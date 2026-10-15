// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"fmt"
	"strings"
)

// CanonicalMinor accepts only the operator-facing supported minor spellings.
func CanonicalMinor(value string) (string, error) {
	switch value {
	case "8.8", "8.9", "8.10":
		return value, nil
	default:
		return "", fmt.Errorf("unsupported non-canonical Camunda minor %q", value)
	}
}

// ObservedMinor extracts a supported major/minor from a full observed server
// version without treating arbitrary substring matches as version proof.
func ObservedMinor(value string) (string, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	parts := strings.Split(value, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return "", fmt.Errorf("invalid observed Camunda version %q", value)
	}
	minor := parts[0] + "." + parts[1]
	if _, err := CanonicalMinor(minor); err != nil {
		return "", fmt.Errorf("invalid observed Camunda version %q: %w", value, err)
	}
	for _, part := range parts[2:] {
		if part == "" || strings.IndexFunc(part, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			return "", fmt.Errorf("invalid observed Camunda version %q", value)
		}
	}
	return minor, nil
}
