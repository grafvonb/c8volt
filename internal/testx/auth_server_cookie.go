package testx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/services/common"
)

type AuthServerCookie struct {
	TS      *httptest.Server
	BaseURL string
}

type CookieAuthOpts struct {
	LoginPath  string // default "/api/login"
	CookieName string // default "sessionid"
	SetCookie  bool   // default true
	StatusCode int    // default 200
	ExpectUser struct {
		Name     string // default "demo"
		Password string // default "demo"
	}
}

func StartAuthServerCookie(t testing.TB, opts CookieAuthOpts) *AuthServerCookie {
	t.Helper()
	opts.LoginPath = common.DefaultVal(opts.LoginPath, "/api/login")
	opts.CookieName = common.DefaultVal(opts.CookieName, "sessionid")
	opts.StatusCode = common.DefaultVal(opts.StatusCode, http.StatusOK)
	opts.ExpectUser.Name = common.DefaultVal(opts.ExpectUser.Name, "demo")
	opts.ExpectUser.Password = common.DefaultVal(opts.ExpectUser.Password, "demo")

	mux := http.NewServeMux()
	mux.HandleFunc(opts.LoginPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		q := r.URL.Query()
		user, pass := q.Get("username"), q.Get("password")
		if user != opts.ExpectUser.Name || pass != opts.ExpectUser.Password {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false})
			return
		}

		if opts.SetCookie {
			http.SetCookie(w, &http.Cookie{
				Name:     opts.CookieName,
				Value:    "ok",
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   true,
				Expires:  time.Now().Add(1 * time.Hour),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(opts.StatusCode)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": opts.StatusCode == http.StatusOK})
	})

	t.Log("trying to start AuthServerCookie...")
	ts := httptest.NewTLSServer(mux)
	t.Log("AuthServerCookie started")
	return &AuthServerCookie{TS: ts, BaseURL: ts.URL}
}

func (s *AuthServerCookie) Close() { s.TS.Close() }
