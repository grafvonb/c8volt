// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatActiveFields renders collected filter fields for debug logging.
func FormatActiveFields(parts []string) string {
	if len(parts) == 0 {
		return "none"
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// AppendQuotedField appends a non-empty string field using Go-style quoting.
func AppendQuotedField(parts []string, name, value string) []string {
	if value == "" {
		return parts
	}
	return append(parts, fmt.Sprintf("%s=%s", name, strconv.Quote(value)))
}

// AppendInt32Field appends a non-zero int32 field.
func AppendInt32Field(parts []string, name string, value int32) []string {
	if value == 0 {
		return parts
	}
	return append(parts, fmt.Sprintf("%s=%d", name, value))
}

// AppendBoolPtrField appends an explicitly set optional boolean field.
func AppendBoolPtrField(parts []string, name string, value *bool) []string {
	if value == nil {
		return parts
	}
	return append(parts, fmt.Sprintf("%s=%t", name, *value))
}

// AppendTrueBoolField appends a boolean field only when it is true.
func AppendTrueBoolField(parts []string, name string, value bool) []string {
	if !value {
		return parts
	}
	return append(parts, fmt.Sprintf("%s=true", name))
}

// AppendRawField appends a non-empty field without quoting its value.
func AppendRawField(parts []string, name, value string) []string {
	if value == "" {
		return parts
	}
	return append(parts, fmt.Sprintf("%s=%s", name, value))
}
