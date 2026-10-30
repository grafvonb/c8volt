// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalidCamundaRelease never includes untrusted observed version text.
var ErrInvalidCamundaRelease = errors.New("invalid Camunda release")

// CamundaRelease retains a numeric compatibility line and a safe observed display.
type CamundaRelease struct {
	Major   uint64
	Minor   uint64
	Display string
}

// camundaReleasePattern accepts release lines with optional patch and conventional suffixes.
var camundaReleasePattern = regexp.MustCompile(`^[vV]?([0-9]+)\.([0-9]+)(?:\.([0-9]+))?(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

// ParseCamundaRelease rejects unsafe/ambiguous values before preserving display text.
// Spaces may surround the release; controls and non-ASCII bytes are never accepted.
func ParseCamundaRelease(raw string) (CamundaRelease, error) {
	for i := range len(raw) {
		if raw[i] < ' ' || raw[i] > '~' {
			return CamundaRelease{}, ErrInvalidCamundaRelease
		}
	}
	display := strings.Trim(raw, " ")
	if len(display) > 128 {
		return CamundaRelease{}, ErrInvalidCamundaRelease
	}
	parts := camundaReleasePattern.FindStringSubmatch(display)
	if parts == nil {
		return CamundaRelease{}, ErrInvalidCamundaRelease
	}
	var numbers [3]uint64
	for i, part := range parts[1:] {
		if part == "" {
			continue
		}
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return CamundaRelease{}, ErrInvalidCamundaRelease
		}
		numbers[i] = n
	}
	return CamundaRelease{Major: numbers[0], Minor: numbers[1], Display: display}, nil
}
