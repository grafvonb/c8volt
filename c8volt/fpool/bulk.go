package fpool

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/toolx"
)

// Reporter is an interface for operation reports that can indicate success/failure
type Reporter interface {
	OK() bool
}

// CalculateTotals returns the total count, successful count, and failed count from a slice of reporters
func CalculateTotals[R Reporter](items []R) (total int, oks int, noks int) {
	for _, item := range items {
		if item.OK() {
			oks++
		}
	}
	total = len(items)
	noks = total - oks
	return
}

// ExecuteBulkOperation executes a bulk operation on a slice of keys with parallel workers
// Returns a slice of reports and an optional error
func ExecuteBulkOperation[R Reporter](
	ctx context.Context,
	keys []string,
	parallel int,
	failFast bool,
	operationName string,
	log *slog.Logger,
	opts []foptions.FacadeOption,
	fn func(context.Context, string, ...foptions.FacadeOption) (R, error),
) ([]R, error) {
	cCfg := foptions.ApplyFacadeOptions(opts)
	ukeys := toolx.UniqueSlice(keys)

	workers := toolx.DetermineNoOfWorkers(len(keys), parallel)
	log.Info(fmt.Sprintf("%s requested for %d unique key(s) using %d worker(s)", operationName, len(ukeys), workers))

	rs, err := ExecuteSlice[string, R](ctx, ukeys, workers, failFast, func(ctx context.Context, key string, _ int) (R, error) {
		return fn(ctx, key, opts...)
	})

	if !cCfg.NoWait {
		_, oks, noks := CalculateTotals(rs)
		log.Info(fmt.Sprintf("%s completed: %d succeeded, %d failed", operationName, oks, noks))
	}

	return rs, err
}
