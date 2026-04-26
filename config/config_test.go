// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

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

func TestResolveEffectiveConfig_ActiveProfileFlagOverridesEnvAndBaseConfig(t *testing.T) {
	t.Setenv("C8VOLT_ACTIVE_PROFILE", "dev")

	v := viper.New()
	v.SetEnvPrefix("c8volt")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
active_profile: base
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  dev:
    app:
      tenant: profile-dev
    apis:
      camunda_api:
        base_url: http://dev.example.test
  prod:
    app:
      tenant: profile-prod
    apis:
      camunda_api:
        base_url: http://prod.example.test
`))
	require.NoError(t, err)

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("profile", "", "")
	require.NoError(t, flags.Parse([]string{"--profile", "prod"}))
	require.NoError(t, v.BindPFlag("active_profile", flags.Lookup("profile")))

	cfg, err := ResolveEffectiveConfig(
		v,
		func(key string) bool {
			return key == "active_profile" && flags.Lookup("profile").Changed || HasEnvConfigByKey(key)
		},
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.NoError(t, err)

	require.Equal(t, "prod", cfg.ActiveProfile)
	require.Equal(t, "profile-prod", cfg.App.Tenant)
	require.Equal(t, "http://prod.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestResolveEffectiveConfig_EnvOverridesProfileForAPIAuthCredentialsAndScopes(t *testing.T) {
	t.Setenv("C8VOLT_AUTH_MODE", "oauth2")
	t.Setenv("C8VOLT_AUTH_OAUTH2_CLIENT_ID", "env-client")
	t.Setenv("C8VOLT_AUTH_OAUTH2_CLIENT_SECRET", "env-secret")
	t.Setenv("C8VOLT_AUTH_OAUTH2_SCOPES_CAMUNDA_API", "env-scope")
	t.Setenv("C8VOLT_APIS_CAMUNDA_API_BASE_URL", "http://env.example.test")

	v := viper.New()
	v.SetEnvPrefix("c8volt")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
active_profile: dev
app:
  tenant: base-tenant
auth:
  mode: cookie
  oauth2:
    token_url: http://base-token.example.test
    client_id: base-client
    client_secret: base-secret
    scopes:
      camunda_api: base-scope
apis:
  camunda_api:
    base_url: http://base.example.test
    require_scope: true
profiles:
  dev:
    auth:
      mode: oauth2
      oauth2:
        token_url: http://profile-token.example.test
        client_id: profile-client
        client_secret: profile-secret
        scopes:
          camunda_api: profile-scope
    apis:
      camunda_api:
        base_url: http://profile.example.test
`))
	require.NoError(t, err)

	cfg, err := ResolveEffectiveConfig(
		v,
		HasEnvConfigByKey,
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.NoError(t, err)

	require.Equal(t, ModeOAuth2, cfg.Auth.Mode)
	require.Equal(t, "http://profile-token.example.test", cfg.Auth.OAuth2.TokenURL)
	require.Equal(t, "env-client", cfg.Auth.OAuth2.ClientID)
	require.Equal(t, "env-secret", cfg.Auth.OAuth2.ClientSecret)
	require.Equal(t, "env-scope", cfg.Auth.OAuth2.Scope(CamundaApiKeyConst))
	require.Equal(t, "http://env.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestResolveEffectiveConfig_EnvTenantOverridesProfileAndBaseConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_TENANT", "env-tenant")

	v := viper.New()
	v.SetEnvPrefix("c8volt")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
active_profile: dev
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  dev:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`))
	require.NoError(t, err)

	cfg, err := ResolveEffectiveConfig(
		v,
		HasEnvConfigByKey,
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.NoError(t, err)

	require.Equal(t, "env-tenant", cfg.App.Tenant)
	require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestResolveEffectiveConfig_ProfileTenantOverridesBaseConfigWhenNoHigherPrecedenceSource(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
active_profile: dev
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  dev:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`))
	require.NoError(t, err)

	cfg, err := ResolveEffectiveConfig(
		v,
		func(string) bool { return false },
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.NoError(t, err)

	require.Equal(t, "profile-tenant", cfg.App.Tenant)
	require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestResolveEffectiveConfig_CriticalBaselineSettingsShareOneContract(t *testing.T) {
	t.Setenv("C8VOLT_ACTIVE_PROFILE", "prod")
	t.Setenv("C8VOLT_AUTH_MODE", "oauth2")
	t.Setenv("C8VOLT_AUTH_OAUTH2_CLIENT_ID", "env-client")
	t.Setenv("C8VOLT_AUTH_OAUTH2_CLIENT_SECRET", "env-secret")
	t.Setenv("C8VOLT_AUTH_OAUTH2_SCOPES_CAMUNDA_API", "env-scope")

	v := viper.New()
	v.SetEnvPrefix("c8volt")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
active_profile: base
app:
  tenant: base-tenant
auth:
  mode: cookie
  oauth2:
    token_url: http://base-token.example.test
    client_id: base-client
    client_secret: base-secret
    scopes:
      camunda_api: base-scope
apis:
  camunda_api:
    base_url: http://base.example.test
    require_scope: true
profiles:
  prod:
    app:
      tenant: profile-tenant
    auth:
      oauth2:
        token_url: http://profile-token.example.test
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

	require.Equal(t, "prod", cfg.ActiveProfile)
	require.Equal(t, "flag-tenant", cfg.App.Tenant)
	require.Equal(t, ModeOAuth2, cfg.Auth.Mode)
	require.Equal(t, "http://profile-token.example.test", cfg.Auth.OAuth2.TokenURL)
	require.Equal(t, "env-client", cfg.Auth.OAuth2.ClientID)
	require.Equal(t, "env-secret", cfg.Auth.OAuth2.ClientSecret)
	require.Equal(t, "env-scope", cfg.Auth.OAuth2.Scope(CamundaApiKeyConst))
	require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestResolveEffectiveConfig_PreservesExplicitEmptyAuthModeForValidation(t *testing.T) {
	t.Setenv("C8VOLT_AUTH_MODE", "")

	v := viper.New()
	v.SetEnvPrefix("c8volt")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
auth:
  mode: oauth2
  oauth2:
    token_url: http://token.example.test
    client_id: client
    client_secret: secret
apis:
  camunda_api:
    base_url: http://base.example.test
`))
	require.NoError(t, err)

	cfg, err := ResolveEffectiveConfig(
		v,
		HasEnvConfigByKey,
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.NoError(t, err)

	require.Empty(t, cfg.Auth.Mode)
	require.ErrorContains(t, cfg.Validate(), `mode: invalid value ""`)
}

func TestResolveEffectiveConfig_PreservesExplicitZeroAndNegativeAppValuesForValidation(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
app:
  process_instance_page_size: 0
  backoff:
    timeout: 0s
    max_retries: -1
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
`))
	require.NoError(t, err)

	cfg, err := ResolveEffectiveConfig(
		v,
		nil,
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.NoError(t, err)

	require.Equal(t, int32(0), cfg.App.ProcessInstancePageSize)
	require.Zero(t, cfg.App.Backoff.Timeout)
	require.Equal(t, -1, cfg.App.Backoff.MaxRetries)
	require.ErrorContains(t, cfg.Validate(), "process_instance_page_size must be greater than 0")
	require.ErrorContains(t, cfg.Validate(), "max_retries must be non-negative")
	require.ErrorContains(t, cfg.Validate(), "timeout must be a positive duration")
}

func TestHTTPValidate_RejectsInvalidAndNonPositiveTimeouts(t *testing.T) {
	require.ErrorContains(t, (&HTTP{Timeout: "eventually"}).Validate(), "invalid duration")
	require.ErrorContains(t, (&HTTP{Timeout: "0s"}).Validate(), "timeout must be a positive duration")
	require.NoError(t, (&HTTP{Timeout: "30s"}).Validate())
}

func TestResolveEffectiveConfig_UnknownProfileReturnsSentinel(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
active_profile: missing
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  dev:
    app:
      tenant: tenant-dev
`))
	require.NoError(t, err)

	_, err = ResolveEffectiveConfig(
		v,
		nil,
		func(activeProfile, key string) bool {
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	require.ErrorIs(t, err, ErrProfileNotFound)
}
