package oauth2

import (
	"context"
	"io"

	"github.com/grafvonb/c8volt/internal/clients/auth/oauth2"
	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
)

const formContentType = "application/x-www-form-urlencoded"

type GenAuthClient interface {
	RequestTokenWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...oauth2.RequestEditorFn) (*oauth2.RequestTokenResponse, error)
}

var _ authenticator.Authenticator = (*Service)(nil)
