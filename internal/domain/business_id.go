// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"unicode/utf8"
)

// ValidateBusinessID validates creation input using the backend's UTF-16 length
// limit. Observed IDs must not use this validator: empty observations are facts.
func ValidateBusinessID(value *string) error {
	if value == nil {
		return nil
	}
	if *value == "" {
		return fmt.Errorf("%w: business ID must not be empty", ErrValidation)
	}
	if !utf8.ValidString(*value) {
		return fmt.Errorf("%w: business ID must be valid UTF-8", ErrValidation)
	}
	units := 0
	for _, r := range *value {
		units++
		if r > 0xffff {
			units++
		}
		if units > 256 {
			return fmt.Errorf("%w: business ID must not exceed 256 UTF-16 code units", ErrValidation)
		}
	}
	return nil
}
