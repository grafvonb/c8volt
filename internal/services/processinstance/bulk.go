// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

func CreateNProcessInstances(ctx context.Context, api API, log *slog.Logger, data d.ProcessInstanceData, n int, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstanceCreation, error) {
	cfg := services.ApplyCallOptions(opts)
	nw := toolx.DetermineNoOfWorkers(n, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("creating %d process instances using %d workers", n, nw), log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("creating %d process instance(s)", n))
	defer stopActivity()
	pics, err := pool.ExecuteNTimes[d.ProcessInstanceCreation](ctx, n, nw, cfg.FailFast, func(ctx context.Context, _ int) (d.ProcessInstanceCreation, error) {
		return api.CreateProcessInstance(ctx, data, opts...)
	})
	if !cfg.NoWait {
		log.Info(fmt.Sprintf("creation of %d process instances completed", n))
	}
	return pics, err
}

func CancelProcessInstances(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, wantedWorkers int, affectedCount int, opts ...services.CallOption) ([]d.Reporter, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	if affectedCount > lk {
		logging.InfoIfVerbose(fmt.Sprintf("cancelling process instances requested for %d affected instance(s) across %d root key(s) using %d worker(s)", affectedCount, lk, nw), log, cfg.Verbose)
	} else {
		logging.InfoIfVerbose(fmt.Sprintf("cancelling process instances requested for %d unique key(s) using %d worker(s)", lk, nw), log, cfg.Verbose)
	}
	stopActivity := logging.StartActivity(ctx, processInstanceBulkActivity("cancelling", lk, affectedCount))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[string, d.Reporter](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.Reporter, error) {
		resp, _, err := api.CancelProcessInstance(ctx, key, opts...)
		return d.Reporter{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, err
	})
	if !cfg.NoWait {
		t, oks, noks := reporterTotals(rs)
		if affectedCount > t {
			log.Info(fmt.Sprintf("cancelling %d process instance(s) completed via %d root request(s): %d root request(s) succeeded or already cancelled/terminated, %d failed", affectedCount, t, oks, noks))
		} else {
			log.Info(fmt.Sprintf("cancelling %d process instance(s) completed: %d succeeded or already cancelled/terminated, %d failed", t, oks, noks))
		}
	}
	return rs, err
}

func DeleteProcessInstances(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, wantedWorkers int, affectedCount int, opts ...services.CallOption) ([]d.Reporter, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	if affectedCount > lk {
		logging.InfoIfVerbose(fmt.Sprintf("deleting process instances requested for %d affected instance(s) across %d root key(s) using %d worker(s)", affectedCount, lk, nw), log, cfg.Verbose)
	} else {
		logging.InfoIfVerbose(fmt.Sprintf("deleting process instances requested for %d unique key(s) using %d worker(s)", lk, nw), log, cfg.Verbose)
	}
	stopActivity := logging.StartActivity(ctx, processInstanceBulkActivity("deleting", lk, affectedCount))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[string, d.Reporter](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.Reporter, error) {
		resp, err := api.DeleteProcessInstance(ctx, key, opts...)
		return d.Reporter{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, err
	})
	if !cfg.NoWait {
		t, oks, noks := reporterTotals(rs)
		if hasStatusCode(rs, http.StatusConflict) {
			affected := affectedCount
			if affected < t {
				affected = t
			}
			log.Info(fmt.Sprintf("cannot delete expanded process-instance scope of %d process instance(s): one or more affected process instances are not in a terminated state; use --force flag to cancel and then delete them", affected))
		}
		if affectedCount > t {
			log.Info(fmt.Sprintf("deleting %d process instance(s) completed via %d root request(s): %d root request(s) succeeded, %d failed", affectedCount, t, oks, noks))
		} else {
			log.Info(fmt.Sprintf("deleting %d process instances completed: %d succeeded, %d failed", t, oks, noks))
		}
	}
	return rs, err
}

func GetProcessInstances(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	ukeys := keys.Unique()
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("getting %d process instance(s)", len(ukeys)))
	defer stopActivity()
	return api.GetProcessInstances(ctx, ukeys, wantedWorkers, opts...)
}

func processInstanceBulkActivity(verb string, rootCount int, affectedCount int) string {
	if affectedCount > rootCount {
		return fmt.Sprintf("%s %d process instance(s) via %d root request(s)", verb, affectedCount, rootCount)
	}
	return fmt.Sprintf("%s %d process instance(s)", verb, rootCount)
}

func reporterTotals(items []d.Reporter) (total, oks, noks int) {
	for _, item := range items {
		if item.Ok {
			oks++
		}
	}
	total = len(items)
	noks = total - oks
	return total, oks, noks
}

func hasStatusCode(items []d.Reporter, statusCode int) bool {
	for _, item := range items {
		if item.StatusCode == statusCode {
			return true
		}
	}
	return false
}
