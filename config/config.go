package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx/logging"
	"gopkg.in/yaml.v3"
)

var (
	ErrNoBaseURL      = errors.New("no base_url provided in api configuration")
	ErrNoTokenURL     = errors.New("no token_url provided in auth configuration")
	ErrNoClientID     = errors.New("no client_id provided in auth configuration")
	ErrNoClientSecret = errors.New("no client_secret provided in auth configuration")

	ErrNoConfigInContext       = errors.New("no config in context")
	ErrInvalidServiceInContext = errors.New("invalid config in context")

	ErrInvalidLogLevel  = errors.New("invalid log.level")
	ErrInvalidLogFormat = errors.New("invalid log.format")
)

func New() *Config {
	return &Config{
		App: App{
			Backoff: common.BackoffConfig{},
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

// WithProfile returns an effective config for the selected profile.
func (c *Config) WithProfile() (*Config, error) {
	if c.ActiveProfile == "" {
		return c, nil
	}
	p, ok := c.Profiles[c.ActiveProfile]
	if !ok {
		return nil, fmt.Errorf("profile %q not found", c.ActiveProfile)
	}

	eff := *c
	eff.App = p.App
	eff.Auth = p.Auth
	eff.APIs = p.APIs
	eff.HTTP = p.HTTP

	return &eff, nil
}

func (c *Config) Normalize() error {
	var errs []error
	if err := c.App.Normalize(); err != nil {
		errs = append(errs, fmt.Errorf("app: %w", err))
	}
	if err := c.Auth.Normalize(); err != nil {
		errs = append(errs, fmt.Errorf("auth: %w", err))
	}
	if err := c.APIs.Normalize(); err != nil {
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
	if err := c.Auth.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("auth: %w", err))
	}
	if err := c.APIs.Validate(c.Auth.OAuth2.Scopes); err != nil {
		errs = append(errs, fmt.Errorf("apis: %w", err))
	}
	if err := c.HTTP.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("http: %w", err))
	}
	if err := c.Log.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("log: %w", err))
	}
	return errors.Join(errs...)
}

type ctxConfigKey struct{}

func (c *Config) ToContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, ctxConfigKey{}, c)
	log := c.Log.NewLogger()
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
