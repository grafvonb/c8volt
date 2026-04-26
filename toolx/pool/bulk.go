// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package pool

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

// Reports wraps a collection of reports and provides total counts
type Reports[T Reporter] struct {
	Items []T
}

// Totals returns the total count, successful count, and failed count
func (r Reports[T]) Totals() (total int, oks int, noks int) {
	for _, item := range r.Items {
		if item.OK() {
			oks++
		}
	}
	total = len(r.Items)
	noks = total - oks
	return
}

// ExecuteBulkOperation executes a bulk operation on a slice of keys with parallel workers
// Returns a slice of reports and an optional error
func ExecuteBulkOperation[R Reporter](ctx context.Context, keys []string, wantedWorkers int, failFast bool, operationName string, log *slog.Logger, opts []foptions.FacadeOption, fn func(context.Context, string, ...foptions.FacadeOption) (R, error)) ([]R, error) {
	cCfg := foptions.ApplyFacadeOptions(opts)
	ukeys := toolx.UniqueSlice(keys)
	lk := len(ukeys)

	workers := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	log.Info(fmt.Sprintf("%s requested for %d unique key(s) using %d worker(s)", operationName, len(ukeys), workers))

	rs, err := ExecuteSlice[string, R](ctx, ukeys, workers, failFast, func(ctx context.Context, key string, _ int) (R, error) {
		return fn(ctx, key, opts...)
	})

	if !cCfg.NoWait {
		total := len(rs)
		oks := 0
		noks := 0
		for _, r := range rs {
			if r.OK() {
				oks++
			}
		}
		noks = total - oks
		log.Info(fmt.Sprintf("%s completed: %d succeeded, %d failed", operationName, oks, noks))
	}

	return rs, err
}
