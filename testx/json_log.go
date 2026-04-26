// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"encoding/json"
	"testing"
)

func LogJson(t *testing.T, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Errorf("marshal: %v", err)
	}
	t.Logf("\n%s", b)
}
