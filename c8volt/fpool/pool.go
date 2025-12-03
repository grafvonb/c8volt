package fpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

func ExecuteNTimes[T any](ctx context.Context, n int, parallel int, failFast bool, fn func(context.Context, int) (T, error)) ([]T, error) {
	if n <= 0 {
		return nil, nil
	}
	if parallel <= 0 {
		parallel = 1
	}
	if parallel > n {
		parallel = n
	}

	out := make([]T, n)
	errs := make([]error, n)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(parallel)

	var sawErr atomic.Bool

	worker := func() {
		defer wg.Done()
		for i := range jobs {
			select {
			case <-ctx.Done():
				if failFast && errs[i] == nil {
					errs[i] = ctx.Err()
				}
				continue
			default:
			}

			res, err := fn(ctx, i)
			if err != nil {
				errs[i] = err
				if failFast && !sawErr.Load() {
					sawErr.Store(true)
					cancel()
				}
				continue
			}
			out[i] = res
		}
	}

	for w := 0; w < parallel; w++ {
		go worker()
	}

produce:
	for i := 0; i < n; i++ {
		if failFast && sawErr.Load() {
			break produce
		}
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	var agg error
	for _, e := range errs {
		if e != nil {
			agg = errors.Join(agg, e)
		}
	}
	return out, agg
}

// ExecuteSlice maps a slice of inputs with concurrency, same semantics
func ExecuteSlice[In any, Out any](ctx context.Context, in []In, parallel int, failFast bool, fn func(context.Context, In, int) (Out, error)) ([]Out, error) {
	return ExecuteNTimes[Out](ctx, len(in), parallel, failFast, func(ctx context.Context, i int) (Out, error) {
		return fn(ctx, in[i], i)
	})
}
