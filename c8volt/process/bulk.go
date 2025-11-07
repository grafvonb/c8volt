package process

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/options"
)

func (c *client) CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, parallel int, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	c.log.Debug(fmt.Sprintf("running %d process instances using %d workers with fail-fast=%t", n, parallel, cCfg.FailFast))

	if n <= 0 {
		return nil, nil
	}
	if parallel <= 0 {
		parallel = 1
	}
	if parallel > n {
		parallel = n
	}

	out := make([]ProcessInstance, n)
	errs := make([]error, n)

	// Derive a cancellable context, can stop in fail-fast mode
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create job channel as work queue
	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(parallel)

	var sawErr atomic.Bool
	worker := func() {
		defer wg.Done()
		for i := range jobs {
			select {
			case <-ctx.Done():
				if cCfg.FailFast && errs[i] == nil {
					errs[i] = ctx.Err()
				}
				continue
			default:
			}
			pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), options.MapFacadeOptionsToCallOptions(opts)...)
			if err != nil {
				errs[i] = ferrors.FromDomain(err)
				if cCfg.FailFast && !sawErr.Load() {
					sawErr.Store(true)
					cancel() // stop in-flight quickly and block new scheduling
				}
				continue
			}
			out[i] = fromDomainProcessInstanceCreation(pic)
		}
	}
	// Start workers
	for w := 0; w < parallel; w++ {
		go worker()
	}

produce:
	// Distribute jobs
	for i := 0; i < n; i++ {
		if sawErr.Load() && cCfg.FailFast {
			break produce
		}
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	var aggErr error
	for _, err := range errs {
		if err != nil {
			aggErr = errors.Join(aggErr, err)
		}
	}
	return out, aggErr
}
