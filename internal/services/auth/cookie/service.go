package cookie

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
	"github.com/grafvonb/c8volt/internal/services/common"
)

var _ authenticator.Authenticator = (*Service)(nil)

type Service struct {
	cfg     *config.Config
	http    *http.Client
	log     *slog.Logger
	baseURL *url.URL

	isAuth bool // set to true when observe at least one non-empty cookie after login
}

type Option func(*Service)

func WithHTTPClient(h *http.Client) Option { return func(s *Service) { s.http = h } }

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("cfg must not be nil")
	}
	if log == nil {
		return nil, errors.New("logger must not be nil")
	}
	log.Debug("Using 'cookie' authenticator: session cookie-based authentication")
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	cfg.APIs.Operate.BaseURL = common.DefaultVal(cfg.APIs.Operate.BaseURL, cfg.APIs.Camunda.BaseURL)
	cfg.APIs.Tasklist.BaseURL = common.DefaultVal(cfg.APIs.Tasklist.BaseURL, cfg.APIs.Camunda.BaseURL)

	s := &Service{cfg: cfg, http: httpClient, log: log}
	for _, opt := range opts {
		opt(s)
	}

	if s.http.Jar == nil {
		jar, _ := cookiejar.New(nil)
		s.http.Jar = jar
	}

	baseUrl, err := url.Parse(cfg.Auth.Cookie.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	s.baseURL = baseUrl
	return s, nil
}

func (s *Service) Name() string { return "cookie" }

func (s *Service) IsAuthenticated() bool { return s.isAuth }

func (s *Service) Init(ctx context.Context) error {
	if s.isAuth {
		return nil
	}

	s.log.Debug(fmt.Sprintf("initializing session cookie auth at %s", s.baseURL.Host))
	loginURL := *s.baseURL
	loginURL.Path = strings.TrimRight(loginURL.Path, "/") + "/api/login"

	query := loginURL.Query()
	query.Set("username", common.DefaultVal(s.cfg.Auth.Cookie.Username, "demo"))
	query.Set("password", common.DefaultVal(s.cfg.Auth.Cookie.Password, "demo"))
	loginURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		hint := ""
		if strings.Contains(err.Error(), "connect: connection refused") {
			hint = " (is the server running?)"
		}
		return fmt.Errorf("login request: %w%s", err, hint)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("login failed: %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Verify we received at least one cookie for the base host.
	cookies := s.http.Jar.Cookies(s.baseURL)
	if len(cookies) == 0 {
		// Some setups mark cookies Secure; re-check https scheme.
		if s.baseURL.Scheme == "http" {
			httpsURL := *s.baseURL
			httpsURL.Scheme = "https"
			if len(s.http.Jar.Cookies(&httpsURL)) > 0 {
				return fmt.Errorf("session cookie is Secure; switch BaseURL to https://%s", s.baseURL.Host)
			}
		}
		return errors.New("login succeeded but no session cookie stored at " + s.baseURL.Host)
	}

	s.isAuth = true
	return nil
}

// Editor adds standard headers and ensures login happened before non-login calls.
// Use this with your generated clients’ RequestEditor hook, or call before building requests.
func (s *Service) Editor() authenticator.RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		sameHost := strings.EqualFold(req.URL.Host, s.baseURL.Host)
		isLogin := strings.Contains(req.URL.Path, "/api/login")
		if sameHost && !isLogin && !s.isAuth {
			return errors.New("cookie auth: not authenticated; call Init first")
		}
		req.Header.Set("Accept", "application/json")
		// Cookies are attached automatically by s.http.Jar for same-host requests.
		return nil
	}
}

func (s *Service) ClearCache() {
	s.isAuth = false
	if s.http != nil && s.http.Jar != nil {
		s.http.Jar.SetCookies(s.baseURL, nil)
	}
}
