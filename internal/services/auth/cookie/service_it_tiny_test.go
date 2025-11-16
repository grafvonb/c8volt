package cookie_test

import (
	"context"
	"log/slog"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"

	config2 "github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth/cookie"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestCookie_Login_OK(t *testing.T) {
	srv := testx.StartAuthServerCookie(t, testx.CookieAuthOpts{
		SetCookie: true,
		ExpectUser: struct {
			Name     string
			Password string
		}{Name: "demo", Password: "demo"},
	})
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	httpClient := srv.TS.Client()
	httpClient.Jar = jar
	httpClient.Timeout = 5 * time.Second

	cfg := &config2.Config{
		Auth: config2.Auth{
			Mode: config2.ModeCookie,
			Cookie: config2.AuthCookieSession{
				BaseURL:  srv.BaseURL,
				Username: "demo",
				Password: "demo",
			},
		},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc, err := cookie.New(cfg, httpClient, log)
	require.NoError(t, err)
	require.NotNil(t, svc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Logf("trying to authenticate aginst %s with user %q", cfg.Auth.Cookie.BaseURL, cfg.Auth.Cookie.Username)
	err = svc.Init(ctx)
	require.NoError(t, err)
	require.True(t, svc.IsAuthenticated())
	t.Log("success: got authenticated")
}

func TestCookie_Login_OK_MissingCookie(t *testing.T) {
	srv := testx.StartAuthServerCookie(t, testx.CookieAuthOpts{SetCookie: false})
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	hc := srv.TS.Client()
	hc.Jar = jar

	cfg := &config2.Config{}
	cfg.Auth.Cookie.BaseURL = srv.BaseURL

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc, _ := cookie.New(cfg, hc, log)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := svc.Init(ctx)
	require.Error(t, err)
	require.False(t, svc.IsAuthenticated())
	t.Logf("success: got expected error: %v", err)
}
