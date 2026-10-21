// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"bytes"
	"encoding/json"
)

// CopyJSONMap snapshots a JSON payload, including nested maps/slices,
// without routing integer keys or numbers through floating-point conversion.
// Numbers are normalized to json.Number; invalid JSON values return an error.
func CopyJSONMap(variables map[string]any) (map[string]any, error) {
	if variables == nil {
		return nil, nil
	}
	data, err := json.Marshal(variables)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var result map[string]any
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
