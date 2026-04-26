// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"errors"
	"fmt"
	"strings"
)

func FormatValidationError(summary string, err error) error {
	if err == nil {
		return nil
	}

	lines := flattenValidationErrors(err)
	if len(lines) == 0 {
		return fmt.Errorf("%s: %w", summary, err)
	}

	var b strings.Builder
	b.WriteString(summary)
	b.WriteString(":\n")
	for _, line := range lines {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return errors.New(strings.TrimRight(b.String(), "\n"))
}

func flattenValidationErrors(err error) []string {
	if err == nil {
		return nil
	}

	type unwrapMany interface {
		Unwrap() []error
	}
	type unwrapOne interface {
		Unwrap() error
	}

	if multi, ok := err.(unwrapMany); ok {
		var out []string
		for _, child := range multi.Unwrap() {
			out = append(out, flattenValidationErrors(child)...)
		}
		return out
	}

	if single, ok := err.(unwrapOne); ok {
		child := single.Unwrap()
		if child != nil {
			var out []string
			for _, line := range flattenValidationErrors(child) {
				if prefix, ok := strings.CutSuffix(err.Error(), ": "+child.Error()); ok {
					out = append(out, prefix+": "+line)
					continue
				}
				out = append(out, err.Error())
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	return []string{err.Error()}
}
