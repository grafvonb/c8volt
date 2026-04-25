package payload

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx/poller"
)

// RequireSingleResource verifies that a successful resource response mapped to a real domain resource.
// body is included in malformed-response errors so callers keep the raw upstream payload for diagnosis.
func RequireSingleResource(resource d.Resource, body []byte) (d.Resource, error) {
	if resource == (d.Resource{}) {
		return d.Resource{}, fmt.Errorf("%w: 200 OK but empty resource payload; body=%s", d.ErrMalformedResponse, string(body))
	}
	return resource, nil
}

// DeploymentProcessDefinitionKeys extracts non-empty process-definition keys from version-specific deployment items.
// deployments is the generated-client deployment slice, and key hides the version-specific shape of each item.
func DeploymentProcessDefinitionKeys[T any](deployments []T, key func(T) string) []string {
	keys := make([]string, 0, len(deployments))
	for _, dep := range deployments {
		k := key(dep)
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

// NewProcessDefinitionVisibilityPoller builds a polling job that waits until deployed process definitions are readable.
// keys are the process-definition keys returned by deployment, and get must return the HTTP response from a single-key
// lookup. A nil response is treated as malformed; 404 and domain not-found errors mean the definition is not visible yet.
func NewProcessDefinitionVisibilityPoller(keys []string, get func(context.Context, string) (*http.Response, error)) func(context.Context) (poller.JobPollStatus, error) {
	return func(ctx context.Context) (poller.JobPollStatus, error) {
		if len(keys) == 0 {
			return poller.JobPollStatus{
				Success: true,
				Message: "no process definitions in deployment; nothing to wait for",
			}, nil
		}
		missing := make([]string, 0)
		for _, k := range keys {
			resp, err := get(ctx, k)
			if err != nil {
				if errors.Is(err, d.ErrNotFound) {
					missing = append(missing, k)
					continue
				}
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: %w", k, err)
			}
			if resp == nil {
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: empty response", k)
			}
			if resp.StatusCode == http.StatusNotFound {
				missing = append(missing, k)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				return poller.JobPollStatus{}, fmt.Errorf("get process definition %q: unexpected status %d", k, resp.StatusCode)
			}
		}
		if len(missing) > 0 {
			return poller.JobPollStatus{
				Success: false,
				Message: fmt.Sprintf("process definitions not visible yet, waiting: %v", missing),
			}, nil
		}
		return poller.JobPollStatus{
			Success: true,
			Message: fmt.Sprintf("process definitions visible: %v", keys),
		}, nil
	}
}
