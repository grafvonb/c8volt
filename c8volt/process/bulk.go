package process

import (
	"context"
	"fmt"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	types "github.com/grafvonb/c8volt/typex"
)

func (c *client) CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, wantedWorkers int, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	cCfg := options.ApplyFacadeOptions(opts)

	nw := toolx.DetermineNoOfWorkers(n, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoV(fmt.Sprintf("creating %d process instances using %d workers", n, nw), c.log, cCfg.Verbose)
	pics, err := pool.ExecuteNTimes[ProcessInstance](ctx, n, nw, cCfg.FailFast, func(ctx context.Context, _ int) (ProcessInstance, error) {
		pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), options.MapFacadeOptionsToCallOptions(opts)...)
		if err != nil {
			return ProcessInstance{}, ferr.FromDomain(err)
		}
		return fromDomainProcessInstanceCreation(pic), nil
	})
	if !cCfg.NoWait {
		c.log.Info(fmt.Sprintf("creation of %d process instances completed", n))
	}
	return pics, err
}

func (c *client) CancelProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (CancelReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoV(fmt.Sprintf("cancelling process instances requested for %d unique key(s) using %d worker(s)", lk, nw), c.log, cCfg.Verbose)
	rs, err := pool.ExecuteSlice[string, CancelReport](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (CancelReport, error) {
		cr, _, cerr := c.CancelProcessInstance(ctx, key, opts...)
		return cr, cerr
	})
	r := CancelReports{
		Items: rs,
	}
	if !cCfg.NoWait {
		t, oks, noks := r.Totals()
		c.log.Info(fmt.Sprintf("cancelling %d process instance(s) completed: %d succeeded or already cancelled/teminated, %d failed", t, oks, noks))
	}
	return r, err
}

func (c *client) DeleteProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DeleteReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoV(fmt.Sprintf("deleting process instances requested for %d unique key(s) using %d worker(s)", lk, nw), c.log, cCfg.Verbose)
	rs, err := pool.ExecuteSlice[string, DeleteReport](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (DeleteReport, error) {
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

func (c *client) WaitForProcessInstancesState(ctx context.Context, keys types.Keys, desired States, wantedWorkers int, opts ...options.FacadeOption) (StateReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoV(fmt.Sprintf("waiting for %d unique process instance(s) to reach desired state(s) %v using %d worker(s)", lk, desired, nw), c.log, cCfg.Verbose)
	got, err := c.piApi.WaitForProcessInstancesState(ctx, ukeys, toolx.MapSlice(desired, func(s State) d.State { return d.State(s) }), nw, options.MapFacadeOptionsToCallOptions(opts)...)
	srs := MapStateResponsesToReports(got)
	if err != nil {
		return srs, ferr.FromDomain(err)
	}
	return srs, nil
}

func (c *client) GetProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstances, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoV(fmt.Sprintf("getting processinstances requested for %d unique key(s) using %d worker(s)", lk, nw), c.log, cCfg.Verbose)
	pis, err := c.piApi.GetProcessInstances(ctx, ukeys, nw, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstances(pis), nil
}
