package process

import (
	"context"
	"fmt"
	"net/http"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	types "github.com/grafvonb/c8volt/typex"
)

// CreateNProcessInstances starts n process instances from the same data using a bounded worker pool.
// wantedWorkers is capped by the repository worker policy unless WithNoWorkerLimit is present in opts.
func (c *client) CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, wantedWorkers int, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	cCfg := options.ApplyFacadeOptions(opts)

	nw := toolx.DetermineNoOfWorkers(n, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("creating %d process instances using %d workers", n, nw), c.log, cCfg.Verbose)
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

// CancelProcessInstances cancels the unique process-instance keys with bounded parallelism.
// wantedWorkers is the requested concurrency; duplicate keys are removed before work is scheduled.
func (c *client) CancelProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (CancelReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	if cCfg.AffectedProcessInstanceCount > lk {
		logging.InfoIfVerbose(fmt.Sprintf("cancelling process instances requested for %d affected instance(s) across %d root key(s) using %d worker(s)", cCfg.AffectedProcessInstanceCount, lk, nw), c.log, cCfg.Verbose)
	} else {
		logging.InfoIfVerbose(fmt.Sprintf("cancelling process instances requested for %d unique key(s) using %d worker(s)", lk, nw), c.log, cCfg.Verbose)
	}
	rs, err := pool.ExecuteSlice[string, CancelReport](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (CancelReport, error) {
		cr, _, cerr := c.CancelProcessInstance(ctx, key, opts...)
		return cr, cerr
	})
	r := CancelReports{
		Items: rs,
	}
	if !cCfg.NoWait {
		t, oks, noks := r.Totals()
		if cCfg.AffectedProcessInstanceCount > t {
			c.log.Info(fmt.Sprintf("cancelling %d process instance(s) completed via %d root request(s): %d root request(s) succeeded or already cancelled/terminated, %d failed", cCfg.AffectedProcessInstanceCount, t, oks, noks))
		} else {
			c.log.Info(fmt.Sprintf("cancelling %d process instance(s) completed: %d succeeded or already cancelled/terminated, %d failed", t, oks, noks))
		}
	}
	return r, err
}

// DeleteProcessInstances deletes the unique process-instance keys with bounded parallelism.
// opts may enable fail-fast or remove the normal worker cap.
func (c *client) DeleteProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DeleteReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	if cCfg.AffectedProcessInstanceCount > lk {
		logging.InfoIfVerbose(fmt.Sprintf("deleting process instances requested for %d affected instance(s) across %d root key(s) using %d worker(s)", cCfg.AffectedProcessInstanceCount, lk, nw), c.log, cCfg.Verbose)
	} else {
		logging.InfoIfVerbose(fmt.Sprintf("deleting process instances requested for %d unique key(s) using %d worker(s)", lk, nw), c.log, cCfg.Verbose)
	}
	rs, err := pool.ExecuteSlice[string, DeleteReport](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (DeleteReport, error) {
		return c.DeleteProcessInstance(ctx, key, opts...)
	})
	r := DeleteReports{
		Items: rs,
	}
	if !cCfg.NoWait {
		t, oks, noks := r.Totals()
		if hasStatusCode(r.Items, http.StatusConflict) {
			affected := cCfg.AffectedProcessInstanceCount
			if affected < t {
				affected = t
			}
			c.log.Info(fmt.Sprintf("cannot delete expanded process-instance scope of %d process instance(s): one or more affected process instances are not in a terminated state; use --force flag to cancel and then delete them", affected))
		}
		if cCfg.AffectedProcessInstanceCount > t {
			c.log.Info(fmt.Sprintf("deleting %d process instance(s) completed via %d root request(s): %d root request(s) succeeded, %d failed", cCfg.AffectedProcessInstanceCount, t, oks, noks))
		} else {
			c.log.Info(fmt.Sprintf("deleting %d process instances completed: %d succeeded, %d failed", t, oks, noks))
		}
	}
	return r, err
}

func hasStatusCode(items []DeleteReport, statusCode int) bool {
	for _, item := range items {
		if item.StatusCode == statusCode {
			return true
		}
	}
	return false
}

// WaitForProcessInstancesState waits for each unique key to reach one of the desired states.
// desired is mapped to the internal domain state set before the versioned service performs polling.
func (c *client) WaitForProcessInstancesState(ctx context.Context, keys types.Keys, desired States, wantedWorkers int, opts ...options.FacadeOption) (StateReports, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("waiting for %d unique process instance(s) to reach desired state(s) %v using %d worker(s)", lk, desired, nw), c.log, cCfg.Verbose)
	got, err := c.piApi.WaitForProcessInstancesState(ctx, ukeys, toolx.MapSlice(desired, func(s State) d.State { return d.State(s) }), nw, options.MapFacadeOptionsToCallOptions(opts)...)
	srs := MapStateResponsesToReports(got)
	if err != nil {
		return srs, ferr.FromDomain(err)
	}
	return srs, nil
}

// GetProcessInstances fetches unique process-instance keys using the internal service bulk lookup path.
// wantedWorkers is forwarded to the service so version-specific implementations can choose their concurrency strategy.
func (c *client) GetProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstances, error) {
	_ = options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	pis, err := c.piApi.GetProcessInstances(ctx, ukeys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstances(pis), nil
}
