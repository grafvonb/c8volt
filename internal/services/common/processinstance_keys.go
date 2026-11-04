// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"fmt"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/typex"
)

// NormalizeProcessInstanceKeys validates all keys before work begins,
// then deduplicates exact identities in first-input order without expanding relatives.
func NormalizeProcessInstanceKeys(keys typex.Keys) (typex.Keys, error) {
	result := make(typex.Keys, 0, len(keys))
	seen := make(map[string]bool, len(keys))
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		valid := len(key) == 16
		for _, ch := range key {
			if ch < '0' || ch > '9' {
				valid = false
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("%w: process-instance key must contain exactly 16 digits", d.ErrValidation)
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, key)
		}
	}
	return result, nil
}
