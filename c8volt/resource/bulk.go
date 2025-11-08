package resource

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/fpool"
	"github.com/grafvonb/c8volt/toolx"
)

func (c *client) DeleteProcessDefinitions(ctx context.Context, keys []string, parallel int, failFast bool, opts ...foptions.FacadeOption) (DeleteReports, error) {
	cCfg := foptions.ApplyFacadeOptions(opts)
	ukeys := toolx.UniqueSlice(keys)

	workers := toolx.DetermineNoOfWorkers(len(keys), parallel)
	c.log.Info(fmt.Sprintf("deleting process definitions requested for %d unique key(s) using %d worker(s)", len(ukeys), workers))
	rs, err := fpool.ExecuteSlice[string, DeleteReport](ctx, ukeys, workers, failFast, func(ctx context.Context, key string, _ int) (DeleteReport, error) {
		return c.DeleteProcessDefinition(ctx, key, opts...)
	})
	r := DeleteReports{
		Items: rs,
	}
	if !cCfg.NoWait {
		t, oks, noks := r.Totals()
		c.log.Info(fmt.Sprintf("deleting %d process definitions completed: %d succeeded, %d failed", t, oks, noks))
	}
	return r, err
}
