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
	rs, err := fpool.ExecuteBulkOperation[CancelReport](
		ctx, keys, parallel, failFast,
		"cancelling process instances",
		c.log, opts,
		c.CancelProcessInstance,
	)
	return CancelReports{Items: rs}, err
}

func (c *client) DeleteProcessInstances(ctx context.Context, keys []string, parallel int, failFast bool, opts ...foptions.FacadeOption) (DeleteReports, error) {
	rs, err := fpool.ExecuteBulkOperation[DeleteReport](
		ctx, keys, parallel, failFast,
		"deleting process instances",
		c.log, opts,
		c.DeleteProcessInstance,
	)
	return DeleteReports{Items: rs}, err
}
