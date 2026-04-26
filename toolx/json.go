// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// JSON writes v as pretty-printed JSON to io.Writer
func JSON(w io.Writer, v any) error {
	return newJSONEncoder(w).Encode(v)
}

// ToJSONString returns v as a pretty-printed JSON string.
func ToJSONString(v any) string {
	var buf bytes.Buffer
	if err := newJSONEncoder(&buf).Encode(v); err != nil {
		return fmt.Sprintf("error encoding JSON: %v", err)
	}
	return buf.String()
}

// newJSONEncoder returns a JSON encoder configured with pretty printing and HTML escaping disabled.
func newJSONEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc
}
