// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

func TestResolveEffectiveConfig_LoadsShowTimezoneOffsetFromConfig(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(`
app:
  show_timezone_offset: true
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
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

	require.True(t, cfg.App.ShowTimezoneOffset)
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

// TestResolveEffectiveConfig_EmptyTenantSemantics documents the unfiltered
// discovery semantics shared by supported runtimes.
func TestResolveEffectiveConfig_VersionSpecificEmptyTenantSemantics(t *testing.T) {
	tests := []struct {
		name        string
		appYAML     string
		wantTenant  string
		wantVersion toolx.CamundaVersion
	}{
		{
			name: "v88 omitted tenant remains empty",
			appYAML: `
  camunda_version: "8.8"`,
			wantVersion: toolx.V88,
		},
		{
			name: "v89 omitted tenant remains empty",
			appYAML: `
  camunda_version: "8.9"`,
			wantVersion: toolx.V89,
		},
		{
			name: "v810 alias omitted tenant remains empty",
			appYAML: `
  camunda_version: "810"`,
			wantVersion: toolx.V810,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			err := v.ReadConfig(strings.NewReader(`
app:` + tt.appYAML + `
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
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
			require.Equal(t, tt.wantVersion, cfg.App.CamundaVersion)
			require.Equal(t, tt.wantTenant, cfg.App.Tenant)
		})
	}
}

func TestConfig_ToSanitizedYAMLWithTenantContextIncludesNamedContext(t *testing.T) {
	cfg := New()
	cfg.App.Tenant = "tenant-a"
	cfg.Auth.Mode = ModeOAuth2
	cfg.Auth.OAuth2.ClientID = "client-id"
	cfg.Auth.OAuth2.ClientSecret = "super-secret"
	cfg.APIs.Camunda.BaseURL = "https://camunda.example.test/v2"

	out, err := cfg.ToSanitizedYAMLWithTenantContext(map[string]any{
		"mode":               "configuration",
		"filter":             "named",
		"configuredTenantId": "tenant-a",
		"resolvedTenantIds":  []string{},
		"unknownTargetCount": 0,
		"crossTenant":        false,
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(out), &got))
	require.Equal(t, "*****", got["auth"].(map[string]any)["oauth2"].(map[string]any)["client_secret"])
	tenantContext := got["tenantContext"].(map[string]any)
	require.Equal(t, "configuration", tenantContext["mode"])
	require.Equal(t, "named", tenantContext["filter"])
	require.Equal(t, "tenant-a", tenantContext["configuredTenantId"])
	require.Equal(t, []any{}, tenantContext["resolvedTenantIds"])
	require.Equal(t, 0, tenantContext["unknownTargetCount"])
	require.Equal(t, false, tenantContext["crossTenant"])
}

func TestConfig_ToSanitizedYAMLWithTenantContextKeepsEmptyTenantUnfiltered(t *testing.T) {
	cfg := New()
	cfg.Auth.Mode = ModeNone
	cfg.APIs.Camunda.BaseURL = "https://camunda.example.test/v2"

	out, err := cfg.ToSanitizedYAMLWithTenantContext(map[string]any{
		"mode":               "configuration",
		"filter":             "none",
		"resolvedTenantIds":  []string{},
		"unknownTargetCount": 0,
		"crossTenant":        false,
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(out), &got))
	require.Equal(t, "", got["app"].(map[string]any)["tenant"])
	tenantContext := got["tenantContext"].(map[string]any)
	require.Equal(t, "configuration", tenantContext["mode"])
	require.Equal(t, "none", tenantContext["filter"])
	require.NotContains(t, tenantContext, "configuredTenantId")
	require.NotContains(t, tenantContext, "targetTenantId")
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
	require.ErrorContains(t, (&HTTP{RequestRateReportInterval: "5s", Timeout: "eventually"}).Validate(), "invalid duration")
	require.ErrorContains(t, (&HTTP{RequestRateReportInterval: "5s", Timeout: "0s"}).Validate(), "timeout must be a positive duration")
	require.NoError(t, (&HTTP{RequestRateReportInterval: "5s", Timeout: "30s"}).Validate())
}

func TestLogNormalize_DefaultsToPlainTimeFormat(t *testing.T) {
	log := Log{}

	log.Normalize()

	require.Equal(t, "plain-time", log.Format)
	require.Equal(t, "info", log.Level)
	require.NoError(t, log.Validate())
}

func TestLogValidate_AcceptsPlainTimeFormat(t *testing.T) {
	require.NoError(t, (&Log{Level: "info", Format: "plain-time"}).Validate())
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

// TestResolveRequestRatePrecedence preserves explicit zero across flags, environment and profiles.
func TestResolveRequestRatePrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, base, profile, env, flag string
		want                           float64
	}{
		{name: "default"},
		{name: "base fraction", base: "0.5", want: 0.5},
		{name: "profile zero", base: "8", profile: "0"},
		{name: "profile fraction", base: "8", profile: "0.5", want: 0.5},
		{name: "environment zero", base: "8", profile: "4", env: "0"},
		{name: "environment fraction", base: "8", profile: "4", env: "0.5", want: 0.5},
		{name: "flag zero", base: "8", profile: "4", env: "2", flag: "0"},
		{name: "flag fraction", base: "8", profile: "4", env: "2", flag: "0.5", want: 0.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			input := "auth:\n  mode: none\n"
			if tc.base != "" {
				input += "http:\n  request_rate: " + tc.base + "\n"
			}
			if tc.profile != "" {
				input += "active_profile: dev\nprofiles:\n  dev:\n    http:\n      request_rate: " + tc.profile + "\n"
			}
			require.NoError(t, v.ReadConfig(strings.NewReader(input)))
			if tc.env != "" {
				t.Setenv("C8VOLT_HTTP_REQUEST_RATE", tc.env)
			}
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flags.Float64("request-rate", 0, "")
			if tc.flag != "" {
				require.NoError(t, flags.Parse([]string{"--request-rate", tc.flag}))
			}
			require.NoError(t, v.BindPFlag("http.request_rate", flags.Lookup("request-rate")))
			cfg, err := ResolveEffectiveConfig(v, func(key string) bool {
				return key == "http.request_rate" && flags.Lookup("request-rate").Changed || HasEnvConfigByKey(key)
			}, func(profile, key string) bool { return v.InConfig("profiles." + profile + "." + key) })
			require.NoError(t, err)
			require.NoError(t, cfg.HTTP.Validate())
			require.Equal(t, tc.want, cfg.HTTP.RequestRate)
		})
	}
}

// TestResolveRequestRateInvalidSources prevents weak numeric decoding from turning malformed input into zero.
func TestResolveRequestRateInvalidSources(t *testing.T) {
	for _, value := range []string{"-1", ".nan", ".inf", "-.inf", "wrong", "''", "true", "[]"} {
		for _, source := range []string{"file", "profile", "env"} {
			t.Run(source+"/"+value, func(t *testing.T) {
				v := viper.New()
				v.SetConfigType("yaml")
				input := "http:\n  request_rate: " + value + "\n"
				if source == "profile" {
					input = "active_profile: dev\nprofiles:\n  dev:\n    http:\n      request_rate: " + value + "\n"
				}
				if source == "env" {
					input = "{}"
					t.Setenv("C8VOLT_HTTP_REQUEST_RATE", strings.Trim(value, "'"))
				}
				require.NoError(t, v.ReadConfig(strings.NewReader(input)))
				cfg, err := ResolveEffectiveConfig(v, HasEnvConfigByKey, func(profile, key string) bool { return v.InConfig("profiles." + profile + "." + key) })
				if err == nil {
					err = cfg.HTTP.Validate()
				}
				require.Error(t, err)
				require.Contains(t, err.Error(), "request_rate")
			})
		}
	}
}

// TestResolveRequestRateReportInterval preserves omitted, empty and zero across source overlays.
func TestResolveRequestRateReportInterval(t *testing.T) {
	for _, tc := range []struct {
		name, base, profile, env, want string
		setEnv, invalid                bool
	}{
		{name: "default", want: "5s"},
		{name: "base", base: "10s", want: "10s"},
		{name: "profile zero", base: "10s", profile: "0s", want: "0s"},
		{name: "profile custom", base: "0s", profile: "250ms", want: "250ms"},
		{name: "env zero", base: "10s", profile: "1s", env: "0s", setEnv: true, want: "0s"},
		{name: "env custom", base: "10s", profile: "1s", env: "2s", setEnv: true, want: "2s"},
		{name: "empty file", base: "''", invalid: true},
		{name: "empty profile", base: "5s", profile: "''", invalid: true},
		{name: "empty env", setEnv: true, invalid: true},
		{name: "negative", base: "-1s", invalid: true},
		{name: "malformed", profile: "nope", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			input := "auth:\n  mode: none\n"
			if tc.base != "" {
				input += "http:\n  request_rate_report_interval: " + tc.base + "\n"
			}
			if tc.profile != "" {
				input += "active_profile: dev\nprofiles:\n  dev:\n    http:\n      request_rate_report_interval: " + tc.profile + "\n"
			}
			require.NoError(t, v.ReadConfig(strings.NewReader(input)))
			if tc.setEnv {
				t.Setenv("C8VOLT_HTTP_REQUEST_RATE_REPORT_INTERVAL", tc.env)
			}
			cfg, err := ResolveEffectiveConfig(v, HasEnvConfigByKey, func(profile, key string) bool { return v.InConfig("profiles." + profile + "." + key) })
			require.NoError(t, err)
			if tc.invalid {
				require.ErrorContains(t, cfg.HTTP.Validate(), "http.request_rate_report_interval")
				return
			}
			require.NoError(t, cfg.HTTP.Validate())
			require.Equal(t, tc.want, cfg.HTTP.RequestRateReportInterval)
			template, err := cfg.ToTemplateYAML()
			require.NoError(t, err)
			require.Contains(t, template, "request_rate_report_interval:")
		})
	}
}
