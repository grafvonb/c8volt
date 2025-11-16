package testx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type AuthServerXsrf struct {
	TS      *httptest.Server
	BaseURL string
}

type XsrfAuthOpts struct {
	SetSessionCookie bool // default true
	SetXSRFToken     bool // default true
}

func StartAuthServerXSRF(t testing.TB, opts XsrfAuthOpts) *AuthServerXsrf {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/xsrf/login/app", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/xsrf/login/") {
			http.NotFound(w, r)
			return
		}
		_ = r.Body.Close()

		if opts.SetSessionCookie {
			http.SetCookie(w, &http.Cookie{
				Name:     "xsrf-session-123456",
				Value:    "ok",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(1 * time.Hour),
			})
		}
		if opts.SetXSRFToken {
			http.SetCookie(w, &http.Cookie{
				Name:     "XSRF-TOKEN",
				Value:    "xsrf-123",
				Path:     "/",
				Secure:   strings.HasPrefix(r.Host, ""),
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(1 * time.Hour),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	t.Log("trying to start AuthServerXsrf...")
	ts := httptest.NewTLSServer(mux)
	t.Log("AuthServerXsrf started")
	return &AuthServerXsrf{TS: ts, BaseURL: ts.URL}
}

func (s *AuthServerXsrf) Close() { s.TS.Close() }
