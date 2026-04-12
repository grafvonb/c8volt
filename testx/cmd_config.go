package testx

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func WriteTestConfig(t *testing.T, baseURL string) string {
	t.Helper()
	return WriteTestConfigForVersion(t, baseURL, "8.8")
}

func WriteTestConfigForVersion(t *testing.T, baseURL string, camundaVersion string) string {
	t.Helper()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	content := fmt.Sprintf(`app:
  camunda_version: %q
auth:
  mode: none
apis:
  camunda_api:
    base_url: %q
`, camundaVersion, baseURL)
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	return cfgPath
}
