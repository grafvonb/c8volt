package resource

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/fpool"
)

func (c *client) DeleteProcessDefinitions(ctx context.Context, keys []string, parallel int, failFast bool, opts ...foptions.FacadeOption) (DeleteReports, error) {
	rs, err := fpool.ExecuteBulkOperation[DeleteReport](
		ctx, keys, parallel, failFast,
		"deleting process definitions",
		c.log, opts,
		c.DeleteProcessDefinition,
	)
	return DeleteReports{Items: rs}, err
}
