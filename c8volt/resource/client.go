package resource

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/toolx/poller"
)

type client struct {
	api  rsvc.API
	papi process.API
	log  *slog.Logger
}

func New(api rsvc.API, papi process.API, log *slog.Logger) API {
	return &client{api: api, papi: papi, log: log}
}

func (c *client) DeployProcessDefinition(ctx context.Context, units []DeploymentUnitData, opts ...options.FacadeOption) ([]ProcessDefinitionDeployment, error) {
	pdd, err := c.api.Deploy(ctx, toDeploymentUnitDatas(units), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return fromProcessDefinitionDeployment(pdd), nil
}

func (c *client) DeleteProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	if !cCfg.NoStateCheck {
		filter := process.ProcessInstanceFilter{ProcessDefinitionKey: key, State: process.StateActive}
		pis, err := c.papi.SearchProcessInstances(ctx, filter, 1, opts...)
		if err != nil {
			return DeleteReport{Key: key, Ok: false}, ferr.FromDomain(err)
		}
		if len(pis.Items) > 0 {
			if cCfg.Force {
				c.log.Info(fmt.Sprintf("cancelling active process instance(s) for process definition %s before deletion", key))
				pis, err = c.papi.SearchProcessInstances(ctx, filter, consts.MaxPISearchSize, opts...)
				if err != nil {
					return DeleteReport{Key: key, Ok: false}, ferr.FromDomain(err)
				}
				var keys []string
				for _, pi := range pis.Items {
					keys = append(keys, pi.Key)
				}
				roots, collected, err := c.papi.DryRunCancelOrDeleteGetPIKeys(context.Background(), keys, opts...)
				if err != nil {
					return DeleteReport{Key: key, Ok: false}, fmt.Errorf("validating process instance keys for cancellation: %w", err)
				}
				c.log.Debug(fmt.Sprintf("found %d process instance(s) to cancel (requested %d, root %d) for process definition %s", len(collected), len(keys), len(roots), key))
				_, err = c.papi.CancelProcessInstances(ctx, roots, len(roots), opts...)
				if err != nil {
					return DeleteReport{Key: key, Ok: false}, fmt.Errorf("cancelling active process instances for process definition %s before deletion failed: %w", key, err)
				}
			} else {
				return DeleteReport{Key: key, Ok: false}, fmt.Errorf("cannot delete process definition %s with active process instances; user --force to cancel them automatically", key)
			}
		}
	}
	if cCfg.AllowInconsistent {
		err := c.api.Delete(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
		if err != nil {
			return DeleteReport{Key: key, Ok: false}, ferr.FromDomain(err)
		}
		/*
			if !cCfg.NoWait {
				if err := c.waitForProcessDefinitionRemoval(ctx, key, opts...); err != nil {
					return DeleteReport{Key: key, Ok: false}, fmt.Errorf("waiting for process definition %s removal failed: %w", key, err)
				}
			}
		*/
	}
	return DeleteReport{Key: key, Ok: true}, nil
}

//nolint:unused
func (c *client) waitForProcessDefinitionRemoval(ctx context.Context, key string, opts ...options.FacadeOption) error {
	poll := func(ctx context.Context) (poller.JobPollStatus, error) {
		_, err := c.papi.GetProcessDefinition(ctx, key, opts...)
		if err != nil {
			if errors.Is(err, ferr.ErrNotFound) {
				return poller.JobPollStatus{
					Success: true,
					Message: fmt.Sprintf("process definition %s no longer listed", key),
				}, nil
			}
			return poller.JobPollStatus{}, err
		}
		return poller.JobPollStatus{
			Success: false,
			Message: fmt.Sprintf("process definition %s still listed", key),
		}, nil
	}
	return poller.WaitForCompletion(ctx, c.log, poller.DefaultCompletionTimeout, true, poll)
}
