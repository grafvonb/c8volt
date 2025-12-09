package resource

import (
	"context"
	"fmt"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	types "github.com/grafvonb/c8volt/typex"
)

func (c *client) DeleteProcessDefinitions(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DeleteReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoV(fmt.Sprintf("deleting process definitions requested for %d unique key(s) using %d worker(s)", lk, nw), c.log, cCfg.Verbose)
	rs, err := pool.ExecuteSlice[string, DeleteReport](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (DeleteReport, error) {
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
