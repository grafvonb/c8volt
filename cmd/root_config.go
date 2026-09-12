// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type configSourceDescription struct {
	loadedPath string
}

type configSourceContextKey struct{}

type resolverBindings struct {
	flags map[string]*pflag.Flag
}

// newResolverBindings creates the flag registry used during config resolution.
func newResolverBindings() *resolverBindings {
	return &resolverBindings{
		flags: make(map[string]*pflag.Flag),
	}
}

func (r *resolverBindings) bindPFlag(v *viper.Viper, key string, flag *pflag.Flag) {
	if flag == nil {
		return
	}
	_ = v.BindPFlag(key, flag)
	r.flags[key] = flag
}

func (r *resolverBindings) hasHigherPrecedenceSource(key string) bool {
	if flag, ok := r.flags[key]; ok && flag != nil && flag.Changed {
		return true
	}
	return hasEnvConfigByKeys([]string{key})
}

// changedFlagValue returns the raw persistent flag value only when the user
// explicitly supplied that flag.
func (r *resolverBindings) changedFlagValue(key string) (string, bool) {
	if r == nil {
		return "", false
	}
	flag, ok := r.flags[key]
	if !ok || flag == nil || !flag.Changed {
		return "", false
	}
	return flag.Value.String(), true
}

func initViper(v *viper.Viper, cmd *cobra.Command) (*resolverBindings, error) {
	fs := cmd.Flags()
	bindings := newResolverBindings()

	bindings.bindPFlag(v, "config", fs.Lookup("config"))
	bindings.bindPFlag(v, "active_profile", fs.Lookup("profile"))
	bindings.bindPFlag(v, "http.timeout", fs.Lookup("timeout"))

	bindings.bindPFlag(v, "log.level", fs.Lookup("log-level"))
	bindings.bindPFlag(v, "log.format", fs.Lookup("log-format"))
	bindings.bindPFlag(v, "log.with_source", fs.Lookup("log-with-source"))

	bindings.bindPFlag(v, "app.tenant", fs.Lookup("tenant"))
	bindings.bindPFlag(v, "app.camunda_version", fs.Lookup("camunda-version"))
	bindings.bindPFlag(v, "app.automation", fs.Lookup("automation"))
	bindings.bindPFlag(v, "app.no_err_codes", fs.Lookup("no-err-codes"))
	bindings.bindPFlag(v, "app.auto-confirm", fs.Lookup("auto-confirm"))
	bindCommandLocalConfigFlags(v, bindings, fs)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "plain-time")
	v.SetDefault("log.with_source", false)
	v.SetDefault("log.with_request_body", false)
	v.SetDefault("http.timeout", "30s")
	v.SetDefault("app.process_instance_page_size", consts.MaxPISearchSize)
	v.SetDefault("app.show_timezone_offset", false)

	v.SetEnvPrefix("c8volt")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	if p := v.GetString("config"); p != "" {
		v.SetConfigFile(p)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$XDG_CONFIG_HOME/c8volt")
		v.AddConfigPath("$HOME/.config/c8volt")
		v.AddConfigPath("$HOME/.c8volt")
		v.AddConfigPath("/etc/c8volt")
	}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok || v.GetString("config") != "" {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}
	return bindings, nil
}

func bindCommandLocalConfigFlags(v *viper.Viper, bindings *resolverBindings, fs *pflag.FlagSet) {
	bindings.bindPFlag(v, "app.backoff.timeout", fs.Lookup("backoff-timeout"))
	bindings.bindPFlag(v, "app.backoff.max_retries", fs.Lookup("backoff-max-retries"))

	if fs.Lookup("backoff-timeout") == nil {
		return
	}
	v.SetDefault("app.backoff.timeout", defaultBackoffTimeout)
	v.SetDefault("app.backoff.max_retries", defaultBackoffMaxRetries)
	v.SetDefault("app.backoff.strategy", defaultBackoffStrategy)
	v.SetDefault("app.backoff.initial_delay", defaultBackoffInitialDelay)
	v.SetDefault("app.backoff.max_delay", defaultBackoffMaxDelay)
	v.SetDefault("app.backoff.multiplier", defaultBackoffMultiplier)
}

func retrieveAndNormalizeConfig(v *viper.Viper, bindings *resolverBindings) (*config.Config, error) {
	cfg, err := config.ResolveEffectiveConfig(
		v,
		bindings.hasHigherPrecedenceSource,
		func(activeProfile, key string) bool {
			if activeProfile == "" {
				return false
			}
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// applyAllTenantsOverride clears the post-normalization tenant only when the
// command-line all-tenants flag actively broadens a named configured filter.
func applyAllTenantsOverride(cfg *config.Config) tenantOverrideProvenance {
	if cfg == nil || !flagAllTenants || cfg.App.Tenant == "" {
		return tenantOverrideProvenance{}
	}
	configuredTenantID := cfg.App.Tenant
	cfg.App.Tenant = ""
	return tenantOverrideProvenance{
		ConfiguredTenantID: configuredTenantID,
		AllTenants:         true,
	}
}

// tenantOverrideProvenanceFromConfig captures command-line tenant provenance
// after config normalization and applies any active all-tenants override.
func tenantOverrideProvenanceFromConfig(v *viper.Viper, bindings *resolverBindings, cfg *config.Config) tenantOverrideProvenance {
	if flagAllTenants {
		return applyAllTenantsOverride(cfg)
	}
	explicitTenantID, explicit := bindings.changedFlagValue("app.tenant")
	if !explicit {
		return tenantOverrideProvenance{}
	}
	return tenantOverrideProvenance{
		ConfiguredTenantID: tenantBeforeExplicitFlag(v, cfg),
		ExplicitTenantID:   explicitTenantID,
		Explicit:           true,
	}
}

// tenantBeforeExplicitFlag approximates the tenant that configuration would
// have produced before the command-line tenant flag overrode it.
func tenantBeforeExplicitFlag(v *viper.Viper, cfg *config.Config) string {
	if value, ok := os.LookupEnv(envNameForKey("app.tenant")); ok {
		return value
	}
	activeProfile := ""
	if v != nil {
		activeProfile = v.GetString("active_profile")
	}
	if value, ok := tenantFromConfigFile(v, activeProfile); ok {
		return value
	}
	if cfg != nil && cfg.App.CamundaVersion == toolx.V87 {
		return config.DefaultTenant
	}
	return ""
}

// tenantFromConfigFile reads only the tenant field from the loaded config file,
// preserving explicit empty values and profile overlay precedence.
func tenantFromConfigFile(v *viper.Viper, activeProfile string) (string, bool) {
	if v == nil || v.ConfigFileUsed() == "" {
		return "", false
	}
	data, err := os.ReadFile(v.ConfigFileUsed())
	if err != nil {
		return "", false
	}
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return "", false
	}
	tenantPath := []string{"app", "tenant"}
	if activeProfile != "" {
		if value, ok := stringAtYAMLPath(root, []string{"profiles", activeProfile, "app", "tenant"}); ok {
			return value, true
		}
	}
	return stringAtYAMLPath(root, tenantPath)
}

// stringAtYAMLPath returns a string scalar from nested YAML maps.
func stringAtYAMLPath(root map[string]any, path []string) (string, bool) {
	var current any = root
	for _, part := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		value, ok := m[part]
		if !ok {
			return "", false
		}
		current = value
	}
	switch value := current.(type) {
	case string:
		return value, true
	default:
		return "", false
	}
}

func (s configSourceDescription) InfoMessage() string {
	if s.loadedPath != "" {
		return "config loaded: " + s.loadedPath
	}
	return "no config file loaded, using defaults and environment variables"
}

func (s configSourceDescription) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, configSourceContextKey{}, s)
}

func configSourceDescriptionFromContext(ctx context.Context) configSourceDescription {
	if ctx == nil {
		return configSourceDescription{}
	}
	source, _ := ctx.Value(configSourceContextKey{}).(configSourceDescription)
	return source
}

func missingConfigHint() string {
	const exampleName = "config.example.yaml"
	if wd, err := os.Getwd(); err == nil {
		examplePath := filepath.Join(wd, exampleName)
		if info, statErr := os.Stat(examplePath); statErr == nil && !info.IsDir() {
			return fmt.Sprintf("no configuration found (environment variables, or config file); c8volt cannot run properly without configuration; found %q in the current directory, copy or edit it into a local config.yaml and run 'c8volt --config ./config.yaml config show --validate'", exampleName)
		}
	}
	return "no configuration found (environment variables, or config file); c8volt cannot run properly without configuration; run 'c8volt config show --template' and use the output to create a config.yaml file"
}

func envNameForKey(key string) string {
	key = strings.ReplaceAll(key, ".", "_")
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ToUpper(key)
	return "C8VOLT_" + key
}

func hasEnvConfigByKeys(configKeys []string) bool {
	for _, key := range configKeys {
		envName := envNameForKey(key)
		if _, ok := os.LookupEnv(envName); ok {
			return true
		}
	}
	return false
}
