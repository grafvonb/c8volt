package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	ErrNoBaseURL       = errors.New("no base_url provided in api configuration")
	ErrNoTokenURL      = errors.New("token_url is required")
	ErrNoClientID      = errors.New("client_id is required")
	ErrNoClientSecret  = errors.New("client_secret is required")
	ErrProfileNotFound = errors.New("profile not found")

	ErrNoConfigInContext       = errors.New("no config in context")
	ErrInvalidServiceInContext = errors.New("invalid config in context")

	ErrInvalidLogLevel  = errors.New("invalid log.level")
	ErrInvalidLogFormat = errors.New("invalid log.format")
)

func New() *Config {
	return &Config{
		App: App{
			Backoff: BackoffConfig{},
		},
		Auth: Auth{
			OAuth2: AuthOAuth2ClientCredentials{
				Scopes: Scopes{},
			},
			Cookie: AuthCookieSession{},
		},
		APIs: APIs{
			Camunda:  API{},
			Operate:  API{},
			Tasklist: API{},
		},
		HTTP: HTTP{},
		Log:  Log{},
	}
}

type Config struct {
	Config string `mapstructure:"config" json:"-" yaml:"-"`

	App  App  `mapstructure:"app" json:"app" yaml:"app"`
	Auth Auth `mapstructure:"auth" json:"auth" yaml:"auth"`
	APIs APIs `mapstructure:"apis" json:"apis" yaml:"apis"`
	HTTP HTTP `mapstructure:"http" json:"http" yaml:"http"`
	Log  Log  `mapstructure:"log" json:"log" yaml:"log"`

	ActiveProfile string             `mapstructure:"active_profile" json:"active_profile,omitempty" yaml:"active_profile,omitempty"`
	Profiles      map[string]Profile `mapstructure:"profiles" json:"-" yaml:"-"`
}

type Profile struct {
	App  App  `mapstructure:"app" json:"app" yaml:"app"`
	Auth Auth `mapstructure:"auth" json:"auth" yaml:"auth"`
	APIs APIs `mapstructure:"apis" json:"apis" yaml:"apis"`
	HTTP HTTP `mapstructure:"http" json:"http" yaml:"http"`
}

// BindConfigEnvVars binds all config fields to their environment variables using mapstructure tags
func BindConfigEnvVars(v *viper.Viper) {
	BindAllEnvVars(v, "C8VOLT_", reflect.TypeOf(Config{}), nil)
}

func BindConfigEnvVarsForProfile(v *viper.Viper, cfg *Config) {
	BindAllEnvVars(v, "C8VOLT_", reflect.TypeOf(*cfg), nil)
}

// BindAllEnvVars recursively binds all config fields to their environment variables using mapstructure tags
func BindAllEnvVars(v *viper.Viper, prefix string, t reflect.Type, path []string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}
		keyPath := append(path, tag)
		ft := field.Type
		// Handle pointers to structs
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			BindAllEnvVars(v, prefix, ft, keyPath)
		} else {
			envKey := prefix + strings.ToUpper(strings.Join(keyPath, "_"))
			_ = v.BindEnv(strings.Join(keyPath, "."), envKey)
		}
	}
}

func ResolveEffectiveConfig(
	v *viper.Viper,
	hasHigherPrecedenceSource func(string) bool,
	isProfileKeyConfigured func(activeProfile string, key string) bool,
) (*Config, error) {
	v.AllowEmptyEnv(true)
	BindConfigEnvVars(v)

	var base Config
	if err := v.Unmarshal(&base); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	applyEnvScopeOverrides(&base)

	cfg, err := base.withProfileOverlay(hasHigherPrecedenceSource, isProfileKeyConfigured)
	if err != nil {
		return nil, fmt.Errorf("apply profile: %w", err)
	}
	if err := cfg.normalizeWithConfiguredKeys(newConfigKeyRegistry(v, cfg.ActiveProfile, hasHigherPrecedenceSource, isProfileKeyConfigured)); err != nil {
		return nil, fmt.Errorf("normalize config: %w", err)
	}
	return cfg, nil
}

// WithProfile returns an effective config for the selected profile.
func (c *Config) WithProfile() (*Config, error) {
	return c.withProfileOverlay(nil, nil)
}

func (c *Config) withProfileOverlay(
	hasHigherPrecedenceSource func(string) bool,
	isProfileKeyConfigured func(activeProfile string, key string) bool,
) (*Config, error) {
	if c.ActiveProfile == "" {
		return c, nil
	}
	p, ok := c.Profiles[c.ActiveProfile]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrProfileNotFound, c.ActiveProfile)
	}

	eff := *c
	overlayProfileValue(reflect.ValueOf(&eff.App).Elem(), reflect.ValueOf(p.App), []string{"app"}, c.ActiveProfile, hasHigherPrecedenceSource, isProfileKeyConfigured)
	overlayProfileValue(reflect.ValueOf(&eff.Auth).Elem(), reflect.ValueOf(p.Auth), []string{"auth"}, c.ActiveProfile, hasHigherPrecedenceSource, isProfileKeyConfigured)
	overlayProfileValue(reflect.ValueOf(&eff.APIs).Elem(), reflect.ValueOf(p.APIs), []string{"apis"}, c.ActiveProfile, hasHigherPrecedenceSource, isProfileKeyConfigured)
	overlayProfileValue(reflect.ValueOf(&eff.HTTP).Elem(), reflect.ValueOf(p.HTTP), []string{"http"}, c.ActiveProfile, hasHigherPrecedenceSource, isProfileKeyConfigured)

	return &eff, nil
}

func overlayProfileValue(
	dst reflect.Value,
	src reflect.Value,
	path []string,
	activeProfile string,
	hasHigherPrecedenceSource func(string) bool,
	isProfileKeyConfigured func(activeProfile string, key string) bool,
) {
	if !dst.CanSet() || !src.IsValid() {
		return
	}

	if src.Kind() == reflect.Pointer {
		if src.IsNil() {
			return
		}
		src = src.Elem()
	}
	if dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}

	switch src.Kind() {
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			field := src.Type().Field(i)
			tag := field.Tag.Get("mapstructure")
			if tag == "" || tag == "-" {
				continue
			}
			overlayProfileValue(dst.Field(i), src.Field(i), append(path, tag), activeProfile, hasHigherPrecedenceSource, isProfileKeyConfigured)
		}
	case reflect.Map:
		if src.IsNil() {
			key := strings.Join(path, ".")
			if !profileKeyConfigured(activeProfile, key, src, isProfileKeyConfigured) {
				return
			}
			if hasHigherPrecedenceSource != nil && hasHigherPrecedenceSource(key) {
				return
			}
			dst.Set(reflect.Zero(dst.Type()))
			return
		}
		if dst.IsNil() {
			dst.Set(reflect.MakeMapWithSize(dst.Type(), src.Len()))
		}
		for _, mapKey := range src.MapKeys() {
			entryPath := append(path, fmt.Sprint(mapKey.Interface()))
			entryValue := src.MapIndex(mapKey)
			key := strings.Join(entryPath, ".")
			if !profileKeyConfigured(activeProfile, key, entryValue, isProfileKeyConfigured) {
				continue
			}
			if hasHigherPrecedenceSource != nil && hasHigherPrecedenceSource(key) {
				continue
			}
			dst.SetMapIndex(mapKey, entryValue)
		}
	default:
		key := strings.Join(path, ".")
		if !profileKeyConfigured(activeProfile, key, src, isProfileKeyConfigured) {
			return
		}
		if hasHigherPrecedenceSource != nil && hasHigherPrecedenceSource(key) {
			return
		}
		dst.Set(src)
	}
}

func profileKeyConfigured(
	activeProfile string,
	key string,
	src reflect.Value,
	isProfileKeyConfigured func(activeProfile string, key string) bool,
) bool {
	if isProfileKeyConfigured != nil {
		return isProfileKeyConfigured(activeProfile, key)
	}
	return !src.IsZero()
}

func applyEnvScopeOverrides(cfg *Config) {
	keys := knownOAuth2ScopeKeys(cfg)
	if len(keys) == 0 {
		return
	}
	if cfg.Auth.OAuth2.Scopes == nil {
		cfg.Auth.OAuth2.Scopes = Scopes{}
	}
	for _, key := range keys {
		if val, ok := os.LookupEnv(envName("auth.oauth2.scopes." + key)); ok {
			cfg.Auth.OAuth2.Scopes[key] = val
		}
	}
}

func knownOAuth2ScopeKeys(cfg *Config) []string {
	seen := make(map[string]struct{})
	var keys []string
	add := func(key string) {
		key = strings.TrimSpace(key)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	add(CamundaApiKeyConst)
	add(OperateApiKeyConst)
	add(TasklistApiKeyConst)
	add(cfg.APIs.Camunda.Key)
	add(cfg.APIs.Operate.Key)
	add(cfg.APIs.Tasklist.Key)
	for key := range cfg.Auth.OAuth2.Scopes {
		add(key)
	}
	for _, profile := range cfg.Profiles {
		add(profile.APIs.Camunda.Key)
		add(profile.APIs.Operate.Key)
		add(profile.APIs.Tasklist.Key)
		for key := range profile.Auth.OAuth2.Scopes {
			add(key)
		}
	}
	return keys
}

func HasEnvConfigByKey(key string) bool {
	_, ok := os.LookupEnv(envName(key))
	return ok
}

func envName(key string) string {
	key = strings.ReplaceAll(key, ".", "_")
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ToUpper(key)
	return "C8VOLT_" + key
}

func (c *Config) Normalize() error {
	return c.normalizeWithConfiguredKeys(func(string) bool { return false })
}

func (c *Config) normalizeWithConfiguredKeys(isConfigured func(string) bool) error {
	var errs []error
	if err := c.App.normalizeWithConfiguredKeys(isConfigured); err != nil {
		errs = append(errs, fmt.Errorf("app: %w", err))
	}
	if err := c.Auth.normalizeWithConfiguredKeys(isConfigured); err != nil {
		errs = append(errs, fmt.Errorf("auth: %w", err))
	}
	if err := c.APIs.normalizeWithConfiguredKeys(isConfigured); err != nil {
		errs = append(errs, fmt.Errorf("apis: %w", err))
	}
	if err := c.HTTP.Normalize(); err != nil {
		errs = append(errs, fmt.Errorf("http: %w", err))
	}
	c.Log.Normalize()
	return errors.Join(errs...)
}

func (c *Config) Validate() error {
	var errs []error
	if err := c.App.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("app: %w", err))
	}
	if err := c.Auth.Validate(); err != nil {
		errs = append(errs, err)
	}
	if err := c.APIs.Validate(c.Auth.OAuth2.Scopes); err != nil {
		errs = append(errs, err)
	}
	if err := c.HTTP.Validate(); err != nil {
		errs = append(errs, err)
	}
	if err := c.Log.Validate(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func newConfigKeyRegistry(
	v *viper.Viper,
	activeProfile string,
	hasHigherPrecedenceSource func(string) bool,
	isProfileKeyConfigured func(activeProfile string, key string) bool,
) func(string) bool {
	return func(key string) bool {
		if hasHigherPrecedenceSource != nil && hasHigherPrecedenceSource(key) {
			return true
		}
		if v != nil && v.InConfig(key) {
			return true
		}
		if activeProfile != "" && isProfileKeyConfigured != nil && isProfileKeyConfigured(activeProfile, key) {
			return true
		}
		return false
	}
}

type ctxConfigKey struct{}

func (c *Config) ToContext(ctx context.Context) context.Context {
	return c.ToContextWithLogWriter(ctx, nil)
}

func (c *Config) ToContextWithLogWriter(ctx context.Context, w io.Writer) context.Context {
	ctx = context.WithValue(ctx, ctxConfigKey{}, c)
	log := c.Log.NewLoggerWithWriter(w)
	return logging.ToContext(ctx, log)
}

func FromContext(ctx context.Context) (*Config, error) {
	v := ctx.Value(ctxConfigKey{})
	if v == nil {
		return nil, ErrNoConfigInContext
	}
	c, ok := v.(*Config)
	if !ok || c == nil {
		return nil, ErrInvalidServiceInContext
	}
	return c, nil
}

func (c *Config) ToSanitizedYAML() (string, error) {
	return c.toYaml(yamlExportOptions{
		template: false,
		sanitizeKeys: []string{
			"client_secret",
			"password",
			"token",
		},
	})
}

func (c *Config) ToTemplateYAML() (string, error) {
	return c.toYaml(yamlExportOptions{
		template:     true,
		sanitizeKeys: nil,
	})
}

type yamlExportOptions struct {
	template     bool
	sanitizeKeys []string
}

func (c *Config) toYaml(opts yamlExportOptions) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return "", err
	}

	if len(opts.sanitizeKeys) > 0 {
		sanitize(m, opts.sanitizeKeys)
	}
	if opts.template {
		applyValueHints(m)
	}
	humanizeDurations(m)

	out, err := yaml.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

var templateHints = map[string]string{
	"mode":     "oauth2|cookie|none",
	"format":   "text|json|plain",
	"level":    "debug|info|warn|error",
	"strategy": "exponential|fixed",
}

var durationKeys = map[string]struct{}{
	"initial_delay": {},
	"max_delay":     {},
	"timeout":       {},
}

func applyValueHints(m map[string]any) {
	for k, v := range m {
		switch x := v.(type) {
		case map[string]any:
			applyValueHints(x)
		case []any:
			m[k] = []any{}
		default:
			if hint, ok := templateHints[k]; ok {
				m[k] = hint
			}
		}
	}
}

func humanizeDurations(m map[string]any) {
	for k, v := range m {
		switch x := v.(type) {
		case map[string]any:
			humanizeDurations(x)
		case []any:
			for i, elem := range x {
				if mm, ok := elem.(map[string]any); ok {
					humanizeDurations(mm)
					x[i] = mm
				}
			}
		case float64:
			if _, ok := durationKeys[k]; ok {
				dur := time.Duration(x)
				m[k] = dur.String()
			}
		}
	}
}

func sanitize(m map[string]any, sensitive []string) {
	for k, v := range m {
		if isSensitive(k, sensitive) {
			m[k] = "*****"
			continue
		}
		switch x := v.(type) {
		case map[string]any:
			sanitize(x, sensitive)
		case []any:
			for _, e := range x {
				if sub, ok := e.(map[string]any); ok {
					sanitize(sub, sensitive)
				}
			}
		}
	}
}

func isSensitive(k string, sensitive []string) bool {
	for _, s := range sensitive {
		if k == s {
			return true
		}
	}
	return false
}
