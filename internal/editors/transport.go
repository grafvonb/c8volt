package editors

import (
	"net/http"

	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
)

type authTransport struct {
	base   http.RoundTripper
	editor authenticator.RequestEditor
}

func (t *authTransport) rt() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	return http.DefaultTransport
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.editor != nil {
		if err := t.editor(req.Context(), req); err != nil {
			return nil, err
		}
	}
	return t.rt().RoundTrip(req)
}

func WithAuth(httpClient *http.Client, editor authenticator.RequestEditor) *http.Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	httpClient.Transport = &authTransport{
		base:   httpClient.Transport,
		editor: editor,
	}
	return httpClient
}
