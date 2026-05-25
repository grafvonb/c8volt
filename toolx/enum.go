// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import "strings"

// CanonicalEnumValue returns the canonical value from valid that matches value
// case-insensitively after trimming surrounding whitespace.
func CanonicalEnumValue[T ~string](value string, valid []T) (T, bool) {
	var zero T
	want := strings.TrimSpace(value)
	if want == "" {
		return zero, false
	}
	for _, candidate := range valid {
		if strings.EqualFold(want, string(candidate)) {
			return candidate, true
		}
	}
	return zero, false
}

// ValidEnumValue reports whether value has a canonical match in valid.
func ValidEnumValue[T ~string](value string, valid []T) bool {
	_, ok := CanonicalEnumValue(value, valid)
	return ok
}

// CanonicalEnumString is a string-specialized convenience wrapper.
func CanonicalEnumString(value string, valid []string) (string, bool) {
	return CanonicalEnumValue[string](value, valid)
}

// ValidEnumString is a string-specialized convenience wrapper.
func ValidEnumString(value string, valid []string) bool {
	return ValidEnumValue[string](value, valid)
}
