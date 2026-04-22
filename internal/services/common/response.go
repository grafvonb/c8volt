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

func RequireSingleProcessInstance(items []d.ProcessInstance, key string) (d.ProcessInstance, error) {
	switch len(items) {
	case 0:
		return d.ProcessInstance{}, ProcessInstanceNotFound(key)
	case 1:
		return items[0], nil
	default:
		return d.ProcessInstance{}, fmt.Errorf("%w: process-instance lookup for key %s returned %d matches", d.ErrMalformedResponse, key, len(items))
	}
}

func ProcessInstanceNotFound(key string) error {
	// Single-resource helpers intentionally keep not-found strict; traversal callers carry partial metadata separately.
	return fmt.Errorf("%w: process instance %s", d.ErrNotFound, key)
}
