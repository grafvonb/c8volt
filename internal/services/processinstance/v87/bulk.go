package v87

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/processinstance/waiter"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

func (s *Service) GetProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	cCcfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)

	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCcfg.NoWorkerLimit)
	logging.InfoV(fmt.Sprintf("getting process instances requested for %d unique key(s) using %d worker(s)", lk, nw), s.log, cCcfg.Verbose)
	rs, err := pool.ExecuteSlice[string, d.ProcessInstance](ctx, ukeys, nw, cCcfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstance, error) {
		pi, err := s.GetProcessInstance(ctx, key, opts...)
		return pi, err
	})
	return rs, err
}

func (s *Service) WaitForProcessInstancesState(ctx context.Context, keys typex.Keys, desired d.States, wantedWorkers int, opts ...services.CallOption) (d.StateResponses, error) {
	return waiter.WaitForProcessInstancesState(ctx, s, s.cfg, s.log, keys, desired, wantedWorkers, opts...)
}
