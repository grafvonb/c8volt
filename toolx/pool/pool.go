package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// ExecuteNTimes runs fn for indices [0,n) with at most wantedWorkers concurrent goroutines,
// cancelling all work on context cancellation or the first error when failFast is true.
// It normalizes worker counts (clamping to [1,n]), allocates result/error slots up front,
// and returns a slice of outputs along with a joined error capturing every failure.
func ExecuteNTimes[T any](ctx context.Context, n int, wantedWorkers int, failFast bool, fn func(context.Context, int) (T, error)) ([]T, error) {
	if n <= 0 {
		return nil, nil
	}
	if wantedWorkers <= 0 {
		wantedWorkers = 1
	}
	if wantedWorkers > n {
		wantedWorkers = n
	}

	out := make([]T, n)
	errs := make([]error, n)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Buffered to let producers enqueue up to worker-count tasks without blocking,
	// smoothing bursts and keeping memory bounded.
	jobs := make(chan int, wantedWorkers)
	var wg sync.WaitGroup
	wg.Add(wantedWorkers)

	var sawErr atomic.Bool

	// Worker consumes indices, short-circuits on ctx errors, and records results.
	worker := func() {
		defer wg.Done()
		for i := range jobs {
			if err := ctx.Err(); err != nil {
				if failFast && errs[i] == nil {
					errs[i] = err
				}
				continue
			}

			res, err := fn(ctx, i)
			if err != nil {
				errs[i] = err
				if failFast && sawErr.CompareAndSwap(false, true) {
					cancel()
				}
				continue
			}
			out[i] = res
		}
	}

	for w := 0; w < wantedWorkers; w++ {
		go worker()
	}

produce:
	// Feed job indices until either fail-fast trips or the context is canceled.
	for i := range n {
		if failFast && sawErr.Load() {
			break produce
		}
		select {
		case <-ctx.Done():
			break produce
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()

	// Join all collected errors so callers can inspect every failure.
	var agg error
	for _, e := range errs {
		if e != nil {
			agg = errors.Join(agg, e)
		}
	}
	return out, agg
}

// ExecuteSlice maps a slice of inputs with concurrency, same semantics
func ExecuteSlice[In any, Out any](ctx context.Context, in []In, wantedWorkers int, failFast bool, fn func(context.Context, In, int) (Out, error)) ([]Out, error) {
	return ExecuteNTimes[Out](ctx, len(in), wantedWorkers, failFast, func(ctx context.Context, i int) (Out, error) {
		return fn(ctx, in[i], i)
	})
}
