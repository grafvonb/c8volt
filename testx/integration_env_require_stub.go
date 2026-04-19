//go:build !integration

package testx

import "testing"

func RequireEnv(t testing.TB, key string) string {
	t.Helper()
	t.Skip("integration-only helper; build with -tags=integration")
	return ""
}

func RequireEnvs(t testing.TB, keys ...string) map[string]string {
	t.Helper()
	t.Skip("integration-only helper; build with -tags=integration")
	return nil
}

func RequireEnvWithPrefix(t testing.TB, key string) string {
	t.Helper()
	t.Skip("integration-only helper; build with -tags=integration")
	return ""
}

func RequireEnvsWithPrefix(t testing.TB, keys ...string) map[string]string {
	t.Helper()
	t.Skip("integration-only helper; build with -tags=integration")
	return nil
}

func GetEnvWithPrefix(key string) string { return "" }

func RequireEnvRaw(t testing.TB, key string) string {
	t.Helper()
	t.Skip("integration-only helper; build with -tags=integration")
	return ""
}

func GetEnvRaw(key string) string { return "" }
