package common

import (
	"context"
	"sync"

	"github.com/grafvonb/c8volt/toolx"
)

// Result holds the outcome for one item.
type Result[T any] struct {
	Index int // original position in the input slice
	Item  T
	Err   error
}

// WorkFunc is the per-item function you want to run.
type WorkFunc[T any] func(ctx context.Context, item T) error

// RunBulk runs fn over items with up to 'parallel' workers.
// - If parallel <= 0, it defaults to min(8, len(items)).
// - Results preserve input order (results[i] corresponds to items[i]).
// - Honors context cancellation; any not-yet-dispatched items are marked with ctx.Err().
func RunBulk[T any](ctx context.Context, items []T, wantedWorkers int, fn WorkFunc[T]) []Result[T] {
	n := len(items)
	results := make([]Result[T], n)
	if n == 0 {
		return results
	}

	nw := toolx.DetermineNoOfWorkers(n, wantedWorkers, false)
	type job struct {
		idx  int
		item T
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	wg.Add(nw)

	// workers
	for w := 0; w < nw; w++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				err := fn(ctx, j.item)
				results[j.idx] = Result[T]{Index: j.idx, Item: j.item, Err: err}
			}
		}()
	}

	// feeder
	go func() {
		defer close(jobs)
		for i, it := range items {
			select {
			case <-ctx.Done():
				// mark remaining as canceled
				for k := i; k < n; k++ {
					results[k] = Result[T]{Index: k, Item: items[k], Err: ctx.Err()}
				}
				return
			case jobs <- job{idx: i, item: it}:
			}
		}
	}()

	wg.Wait()
	return results
}
