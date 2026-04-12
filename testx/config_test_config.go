package testx

import (
	"testing"

	"github.com/grafvonb/c8volt/config"
)

func TestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		App: config.App{
			Tenant: "tenant",
		},
		Auth: config.Auth{
			OAuth2: config.AuthOAuth2ClientCredentials{
				TokenURL:     "http://localhost/token",
				ClientID:     "test",
				ClientSecret: "test",
			},
			Cookie: config.AuthCookieSession{
				BaseURL:  "http://localhost/cookie",
				Username: "test",
				Password: "test",
			},
		},
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "http://localhost/camunda/v2",
			},
			Operate: config.API{
				BaseURL: "http://localhost/operate",
			},
			Tasklist: config.API{
				BaseURL: "http://localhost/tasklist",
			},
		},
	}
}
