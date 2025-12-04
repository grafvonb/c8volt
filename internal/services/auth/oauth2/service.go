package oauth2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/clients/auth/oauth2"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

type TargetResolver func(*http.Request) string

type Service struct {
	c          GenAuthClient
	cfg        *config.Config
	log        *slog.Logger
	resolve    TargetResolver
	headerName string
	prefix     string

	tokenURL *url.URL

	mu    sync.Mutex
	cache map[string]string
}

type Option func(*Service)

func WithClient(c GenAuthClient) Option {
	return func(s *Service) { s.c = c }
}

func WithTargetResolver(r TargetResolver) Option {
	return func(s *Service) { s.resolve = r }
}

func WithAuthHeader(name, prefix string) Option {
	return func(s *Service) { s.headerName, s.prefix = name, prefix }
}

func New(cfg *config.Config, apiHTTP *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("cfg must not be nil")
	}
	if log == nil {
		return nil, errors.New("logger must not be nil")
	}
	log.Debug("using 'oauth2' authenticator. OAuth2 Client Credentials flow will be used to obtain bearer tokens.")
	if apiHTTP == nil {
		apiHTTP = http.DefaultClient
	}

	// plain client for tokens (no auth transport)
	tu, err := url.Parse(cfg.Auth.OAuth2.TokenURL)
	if err != nil {
		return nil, fmt.Errorf("parse token url: %w", err)
	}
	tokenHTTP := &http.Client{Timeout: apiHTTP.Timeout, Transport: &httpc.LogTransport{Log: log, WithBody: cfg.Log.WithRequestBody}}

	cfg.APIs.Operate.BaseURL = common.DefaultVal(cfg.APIs.Operate.BaseURL, cfg.APIs.Camunda.BaseURL)
	cfg.APIs.Tasklist.BaseURL = common.DefaultVal(cfg.APIs.Tasklist.BaseURL, cfg.APIs.Camunda.BaseURL)

	c, err := oauth2.NewClientWithResponses(tu.String(), oauth2.WithHTTPClient(tokenHTTP))
	if err != nil {
		return nil, fmt.Errorf("init auth client: %w", err)
	}

	s := &Service{
		c:          c,
		cfg:        cfg,
		log:        log,
		resolve:    func(r *http.Request) string { return r.URL.Host },
		headerName: "Authorization",
		prefix:     "Bearer ",
		tokenURL:   tu,
		cache:      make(map[string]string),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Service) Name() string { return "oauth2" }

func (s *Service) Init(_ context.Context) error { return nil }

func (s *Service) Editor() authenticator.RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		// Do NOT add auth to the token request itself.
		if sameURL(req.URL, s.tokenURL) {
			return nil
		}
		var target string
		u := req.URL.String()
		switch {
		case s.cfg.APIs.Camunda.RequireScope && strings.Contains(u, s.cfg.APIs.Camunda.BaseURL):
			target = s.cfg.APIs.Camunda.Key
		case s.cfg.APIs.Tasklist.RequireScope && strings.Contains(u, s.cfg.APIs.Tasklist.BaseURL):
			target = s.cfg.APIs.Tasklist.Key
		case s.cfg.APIs.Operate.RequireScope && strings.Contains(u, s.cfg.APIs.Operate.BaseURL):
			target = s.cfg.APIs.Operate.Key
		}
		tok, err := s.RetrieveTokenForAPI(ctx, target)
		if err != nil {
			return err
		}
		req.Header.Set(s.headerName, s.prefix+tok)
		return nil
	}
}

func sameURL(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Host, b.Host) &&
		path.Clean(a.Path) == path.Clean(b.Path)
}

func (s *Service) ClearCache() {
	s.mu.Lock()
	s.cache = make(map[string]string)
	s.mu.Unlock()
}

func (s *Service) Token(ctx context.Context, target string) (string, error) {
	return s.RetrieveTokenForAPI(ctx, target)
}

func (s *Service) RetrieveTokenForAPI(ctx context.Context, target string) (string, error) {
	if s == nil {
		return "", errors.New("oauth2 service is nil (not wired)")
	}
	s.log.Debug(fmt.Sprintf("looking up bearer token in cache for target: %s", target))
	s.mu.Lock()
	if tok, ok := s.cache[target]; ok && tok != "" {
		s.mu.Unlock()
		s.log.Debug(fmt.Sprintf("found bearer token in cache for target: %s", target))
		return tok, nil
	}
	s.mu.Unlock()

	scope := s.cfg.Auth.OAuth2.Scope(target)
	s.log.Debug(fmt.Sprintf("fetching bearer token for target: %s", target))
	tok, err := s.requestToken(ctx, s.cfg.Auth.OAuth2.ClientID, s.cfg.Auth.OAuth2.ClientSecret, scope)
	if err != nil {
		var t string
		if strings.TrimSpace(target) != "" {
			t = fmt.Sprintf(" for target %s", target)
		}
		return "", fmt.Errorf("retrieve token%s: %w", t, err)
	}

	s.log.Debug(fmt.Sprintf("puting bearer token in cache for target: %s", target))
	s.mu.Lock()
	s.cache[target] = tok
	s.mu.Unlock()
	return tok, nil
}

func (s *Service) requestToken(ctx context.Context, clientID, clientSecret, scope string) (string, error) {
	body := formBody(clientID, clientSecret, scope)
	resp, err := s.c.RequestTokenWithBodyWithResponse(ctx, formContentType, body) // uses plain tokenHTTP
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("nil token response")
	}
	if err = httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return "", err
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}
	return resp.JSON200.AccessToken, nil
}

func formBody(clientID, clientSecret, scope string) io.Reader {
	f := url.Values{}
	f.Set("grant_type", "client_credentials")
	f.Set("client_id", clientID)
	f.Set("client_secret", clientSecret)
	if strings.TrimSpace(scope) != "" {
		f.Set("scope", scope)
	}
	return strings.NewReader(f.Encode())
}
