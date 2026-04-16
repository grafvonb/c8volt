package config

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestResolveEffectiveConfig_ProfileOverlaysBaseConfigWithoutReplacingExplicitSources(t *testing.T) {
	t.Setenv("C8VOLT_AUTH_MODE", "none")

	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
active_profile: dev
app:
  tenant: base-tenant
auth:
  mode: cookie
apis:
  camunda_api:
    base_url: http://base.example.test
http:
  timeout: 30s
profiles:
  dev:
    app:
      tenant: profile-tenant
    auth:
      mode: oauth2
    apis:
      camunda_api:
        base_url: http://profile.example.test
`))
	require.NoError(t, err)

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("tenant", "", "")
	require.NoError(t, flags.Parse([]string{"--tenant", "flag-tenant"}))
	require.NoError(t, v.BindPFlag("app.tenant", flags.Lookup("tenant")))

	cfg, err := ResolveEffectiveConfig(
		v,
		func(key string) bool {
			return key == "app.tenant" && flags.Lookup("tenant").Changed || HasEnvConfigByKey(key)
		},
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.NoError(t, err)

	require.Equal(t, "flag-tenant", cfg.App.Tenant)
	require.Equal(t, ModeNone, cfg.Auth.Mode)
	require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestConfigWithProfile_PreservesBaseValuesForUnsetProfileFields(t *testing.T) {
	cfg := &Config{
		ActiveProfile: "dev",
		App: App{
			Tenant: "base-tenant",
		},
		Auth: Auth{
			Mode: ModeCookie,
		},
		Profiles: map[string]Profile{
			"dev": {
				Auth: Auth{
					Mode: ModeOAuth2,
				},
			},
		},
	}

	effective, err := cfg.WithProfile()
	require.NoError(t, err)

	require.Equal(t, "base-tenant", effective.App.Tenant)
	require.Equal(t, ModeOAuth2, effective.Auth.Mode)
}

func TestResolveEffectiveConfig_UsesCommandLocalBackoffValuesInSharedResolver(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
active_profile: dev
app:
  backoff:
    timeout: 12s
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
http:
  timeout: 30s
profiles:
  dev:
    app:
      backoff:
        timeout: 9s
`))
	require.NoError(t, err)

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Duration("backoff-timeout", 0, "")
	require.NoError(t, flags.Parse([]string{"--backoff-timeout", "45s"}))
	require.NoError(t, v.BindPFlag("app.backoff.timeout", flags.Lookup("backoff-timeout")))

	cfg, err := ResolveEffectiveConfig(
		v,
		func(key string) bool {
			return key == "app.backoff.timeout" && flags.Lookup("backoff-timeout").Changed
		},
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.NoError(t, err)

	require.Equal(t, 45*time.Second, cfg.App.Backoff.Timeout)
}
