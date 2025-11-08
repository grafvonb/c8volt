package process

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/fpool"
	"github.com/grafvonb/c8volt/toolx"
)

func (c *client) CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n, parallel int, opts ...foptions.FacadeOption) ([]ProcessInstance, error) {
	cCfg := foptions.ApplyFacadeOptions(opts)

	workers := toolx.DetermineNoOfWorkers(n, parallel)
	c.log.Info(fmt.Sprintf("creating %d process instances using %d workers", n, workers))
	pics, err := fpool.ExecuteNTimes[ProcessInstance](ctx, n, workers, cCfg.FailFast, func(ctx context.Context, _ int) (ProcessInstance, error) {
		pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), foptions.MapFacadeOptionsToCallOptions(opts)...)
		if err != nil {
			return ProcessInstance{}, ferrors.FromDomain(err)
		}
		return fromDomainProcessInstanceCreation(pic), nil
	})
	if !cCfg.NoWait {
		c.log.Info(fmt.Sprintf("creation of %d process instances completed", n))
	}
	return pics, err
}

func (c *client) CancelProcessInstances(ctx context.Context, keys []string, parallel int, failFast bool, opts ...foptions.FacadeOption) (CancelReports, error) {
	cCfg := foptions.ApplyFacadeOptions(opts)
	ukeys := toolx.UniqueSlice(keys)

	workers := toolx.DetermineNoOfWorkers(len(keys), parallel)
	c.log.Info(fmt.Sprintf("cancelling process instances requested for %d unique key(s) using %d worker(s)", len(ukeys), workers))
	rs, err := fpool.ExecuteSlice[string, CancelReport](ctx, ukeys, workers, failFast, func(ctx context.Context, key string, _ int) (CancelReport, error) {
		return c.CancelProcessInstance(ctx, key, opts...)
	})
	r := CancelReports{
		Items: rs,
	}
	if !cCfg.NoWait {
		t, oks, noks := r.Totals()
		c.log.Info(fmt.Sprintf("cancelling %d process instances completed: %d succeeded or already cancelled/teminated, %d failed", t, oks, noks))
	}
	return r, err
}

func (c *client) DeleteProcessInstances(ctx context.Context, keys []string, parallel int, failFast bool, opts ...foptions.FacadeOption) (DeleteReports, error) {
	cCfg := foptions.ApplyFacadeOptions(opts)
	ukeys := toolx.UniqueSlice(keys)

	workers := toolx.DetermineNoOfWorkers(len(keys), parallel)
	c.log.Info(fmt.Sprintf("deleting process instances requested for %d unique key(s) using %d worker(s)", len(ukeys), workers))
	rs, err := fpool.ExecuteSlice[string, DeleteReport](ctx, ukeys, workers, failFast, func(ctx context.Context, key string, _ int) (DeleteReport, error) {
		return c.DeleteProcessInstance(ctx, key, opts...)
	})
	r := DeleteReports{
		Items: rs,
	}
	if !cCfg.NoWait {
		t, oks, noks := r.Totals()
		c.log.Info(fmt.Sprintf("deleting %d process instances completed: %d succeeded, %d failed", t, oks, noks))
	}
	return r, err
}
