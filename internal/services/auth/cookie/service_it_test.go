//go:build integration

package cookie_test

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth/cookie"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestCookie_Login_OK_IT(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	cfg := &config.Config{
		Auth: config.Auth{
			Mode: config.ModeCookie,
			Cookie: config.AuthCookieSession{
				BaseURL:  testx.RequireEnvWithPrefix(t, "COOKIE_BASE_URL"),
				Username: testx.RequireEnvWithPrefix(t, "COOKIE_USERNAME"),
				Password: testx.RequireEnvWithPrefix(t, "COOKIE_PASSWORD"),
			},
		},
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc, err := cookie.New(cfg, httpClient, log)
	require.NoError(t, err)
	require.NotNil(t, svc)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	t.Logf("trying to authenticate aginst %s with user %q", cfg.Auth.Cookie.BaseURL, cfg.Auth.Cookie.Username)
	err = svc.Init(ctx)
	require.NoError(t, err)
	require.True(t, svc.IsAuthenticated())
	t.Log("success: got authenticated")
}
