package testx

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth"
	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// Reuse one authenticated client per package run.
// Safe for parallel tests if underlying client is safe.
var (
	once         sync.Once
	sharedClient *http.Client
	sharedErr    error
)

func ITHttpClient(t *testing.T, ctx context.Context, cfg *config.Config, log *slog.Logger) *http.Client {
	t.Helper()
	once.Do(func() {
		var httpSvc *httpc.Service
		httpSvc, sharedErr = httpc.New(cfg, log, httpc.WithCookieJar())
		if sharedErr != nil {
			return
		}
		var ator authenticator.Authenticator
		ator, sharedErr = auth.BuildAuthenticator(cfg, httpSvc.Client(), log)
		if sharedErr != nil {
			return
		}
		sharedErr = ator.Init(ctx)
		if sharedErr != nil {
			return
		}
		httpSvc.InstallAuthEditor(ator.Editor())
		sharedClient = httpSvc.Client()
	})
	require.NoError(t, sharedErr)
	require.NotNil(t, sharedClient)
	return sharedClient
}

func ITCtx(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

func ITConfigFromEnv(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		App: config.App{
			CamundaVersion: toolx.CamundaVersion(RequireEnvWithPrefix(t, "API_VERSION")),
		},
		Auth: config.Auth{
			Mode: "cookie",
			Cookie: config.AuthCookieSession{
				BaseURL:  RequireEnvWithPrefix(t, "COOKIE_BASE_URL"),
				Username: RequireEnvWithPrefix(t, "COOKIE_USER"),
				Password: RequireEnvWithPrefix(t, "COOKIE_PASSWORD"),
			},
		},
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: RequireEnvWithPrefix(t, "CAMUNDA_API_BASE_URL"),
			},
			Operate: config.API{
				BaseURL: RequireEnvWithPrefix(t, "OPERATE_API_BASE_URL"),
			},
			Tasklist: config.API{
				BaseURL: RequireEnvWithPrefix(t, "TASKLIST_API_BASE_URL"),
			},
		},
		HTTP: config.HTTP{Timeout: "30s"},
	}
}

func Logger(t *testing.T) *slog.Logger {
	t.Helper()
	return logging.New(logging.LoggerConfig{
		Level:      "debug",
		Format:     "plain",
		WithSource: false,
	})
}
