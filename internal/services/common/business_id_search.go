// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"fmt"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"unicode/utf8"
)

// ValidateBusinessIDSearch checks presence, adapter support and opaque search input without creation length rules or I/O.
func ValidateBusinessIDSearch(value *string, version toolx.CamundaVersion) error {
	if value == nil {
		return nil
	}
	if version != toolx.V810 {
		return fmt.Errorf("%w: business-ID search requires Camunda 8.10", d.ErrUnsupported)
	}
	if *value == "" || !utf8.ValidString(*value) {
		return fmt.Errorf("%w: business-ID search requires a nonempty valid UTF-8 value", d.ErrValidation)
	}
	return nil
}
