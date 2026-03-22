package payload

import (
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
)

func RequireSingleResource(resource d.Resource, body []byte) (d.Resource, error) {
	if resource == (d.Resource{}) {
		return d.Resource{}, fmt.Errorf("%w: 200 OK but empty resource payload; body=%s", d.ErrMalformedResponse, string(body))
	}
	return resource, nil
}
