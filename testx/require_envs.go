//go:build integration

package testx

import (
	"os"
	"testing"
)

const Envc8voltTestPrefix = "C8VOLT_TEST_"

func RequireEnv(t testing.TB, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("missing %s; skipping integration test", key)
	}
	return v
}

func RequireEnvs(t testing.TB, keys ...string) map[string]string {
	t.Helper()
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = RequireEnv(t, k)
	}
	return out
}

func RequireEnvWithPrefix(t testing.TB, key string) string {
	t.Helper()
	return RequireEnv(t, Envc8voltTestPrefix+key)
}
func RequireEnvsWithPrefix(t testing.TB, keys ...string) map[string]string {
	t.Helper()
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = RequireEnvWithPrefix(t, k)
	}
	return out
}
func GetEnvWithPrefix(key string) string { return os.Getenv(Envc8voltTestPrefix + key) }

func RequireEnvRaw(t testing.TB, key string) string { return RequireEnv(t, key) }
func GetEnvRaw(key string) string                   { return os.Getenv(key) }
