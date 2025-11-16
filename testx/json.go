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
