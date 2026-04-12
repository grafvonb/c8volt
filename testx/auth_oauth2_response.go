package testx

import (
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/internal/clients/auth/oauth2"
)

type tokenJSON200 = struct {
	AccessToken  string  `json:"access_token"`
	ExpiresIn    int     `json:"expires_in"`
	IdToken      *string `json:"id_token,omitempty"`
	RefreshToken *string `json:"refresh_token,omitempty"`
	Scope        *string `json:"scope,omitempty"`
	TokenType    string  `json:"token_type"`
}

func TestAuthJSON200Response(t *testing.T, status int, token string, raw string) *oauth2.RequestTokenResponse {
	t.Helper()
	return &oauth2.RequestTokenResponse{
		Body: []byte(raw),
		JSON200: &tokenJSON200{
			AccessToken: token,
			TokenType:   "Bearer",
		},
		HTTPResponse: &http.Response{StatusCode: status},
	}
}
