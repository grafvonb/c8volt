package httpc

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"

	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
)

type LogTransport struct {
	base     http.RoundTripper
	WithBody bool
	Log      *slog.Logger
}

func (t *LogTransport) rt() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	if t.Log == nil {
		t.Log = slog.Default()
	}
	return http.DefaultTransport
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.WithBody {
		// clone body to avoid consuming it
		var bodyCopy []byte
		if req.Body != nil {
			bodyCopy, _ = httputil.DumpRequestOut(req, true)
		} else {
			bodyCopy, _ = httputil.DumpRequestOut(req, false)
		}
		// restore body if needed
		if req.Body != nil && len(bodyCopy) > 0 {
			// DumpRequestOut already reads body, so rebuild it
			req.Body = io.NopCloser(bytes.NewReader(extractBody(bodyCopy)))
		}
		t.Log.Debug(string(bodyCopy))
		return t.rt().RoundTrip(req)
	}
	t.Log.Debug(fmt.Sprintf("calling: %s %s", req.Method, req.URL.String()))
	return t.rt().RoundTrip(req)
}

// helper to extract body part from DumpRequestOut output
func extractBody(dump []byte) []byte {
	parts := bytes.SplitN(dump, []byte("\r\n\r\n"), 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return nil
}

type AuthTransport struct {
	base   http.RoundTripper
	Editor authenticator.RequestEditor
}

func (t *AuthTransport) rt() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	return http.DefaultTransport
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Editor != nil {
		if err := t.Editor(req.Context(), req); err != nil {
			return nil, err
		}
	}
	return t.rt().RoundTrip(req)
}
