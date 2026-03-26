package common

import (
	"fmt"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

func RequirePayload[T any](httpResp *http.Response, body []byte, payload *T) (*T, error) {
	if err := httpc.HttpStatusErr(httpResp, body); err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(body))
	}
	return payload, nil
}
