package config

import (
	"errors"
	"fmt"
	"strings"
)

type AuthMode string

func (m AuthMode) IsValid() bool { return m == ModeOAuth2 || m == ModeCookie || m == ModeNone }

const (
	ModeNone   AuthMode = "none"
	ModeOAuth2 AuthMode = "oauth2"
	ModeCookie AuthMode = "cookie"
)

type Auth struct {
	Mode   AuthMode                    `mapstructure:"mode" json:"mode" yaml:"mode"`
	OAuth2 AuthOAuth2ClientCredentials `mapstructure:"oauth2" json:"oauth2" yaml:"oauth2"`
	Cookie AuthCookieSession           `mapstructure:"cookie" json:"cookie" yaml:"cookie"`
}

func (c *Auth) Normalize() error {
	if strings.TrimSpace(string(c.Mode)) == "" {
		c.Mode = ModeNone
	}
	return nil
}

func (c *Auth) Validate() error {
	var errs []error
	if !c.Mode.IsValid() {
		errs = append(errs, fmt.Errorf("mode: invalid value %q (allowed values: %q, %q)", c.Mode, ModeOAuth2, ModeCookie))
	} else {
		switch c.Mode {
		case ModeOAuth2:
			if err := c.OAuth2.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("oauth2: %w", err))
			}
		case ModeCookie:
			if err := c.Cookie.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("cookie: %w", err))
			}
		}
	}
	return errors.Join(errs...)
}

type Scopes map[string]string

type AuthOAuth2ClientCredentials struct {
	TokenURL     string `mapstructure:"token_url" json:"token_url" yaml:"token_url"`
	ClientID     string `mapstructure:"client_id" json:"client_id" yaml:"client_id"`
	ClientSecret string `mapstructure:"client_secret" json:"client_secret" yaml:"client_secret"`
	Scopes       Scopes `mapstructure:"scopes" json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

func (a *AuthOAuth2ClientCredentials) Validate() error {
	var errs []error

	if strings.TrimSpace(a.TokenURL) == "" {
		errs = append(errs, ErrNoTokenURL)
	}
	if strings.TrimSpace(a.ClientID) == "" {
		errs = append(errs, ErrNoClientID)
	}
	if strings.TrimSpace(a.ClientSecret) == "" {
		errs = append(errs, ErrNoClientSecret)
	}
	return errors.Join(errs...)
}

func (a *AuthOAuth2ClientCredentials) Scope(key string) string {
	if a.Scopes == nil {
		return ""
	}
	return a.Scopes[key]
}

type AuthCookieSession struct {
	BaseURL  string `mapstructure:"base_url" json:"base_url" yaml:"base_url"`
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
}

func (c *AuthCookieSession) Validate() error {
	var errs []error
	if strings.TrimSpace(c.BaseURL) == "" {
		errs = append(errs, fmt.Errorf("auth.cookie.base_url: %w", ErrNoBaseURL))
	}
	return errors.Join(errs...)
}
