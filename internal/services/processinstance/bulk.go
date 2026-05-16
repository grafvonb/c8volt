// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

var processInstanceBulkProgressInterval = 30 * time.Second

type processInstanceCreator interface {
	CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error)
}

func CreateNProcessInstances(ctx context.Context, api API, log *slog.Logger, data d.ProcessInstanceData, n int, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstanceCreation, error) {
	cfg := services.ApplyCallOptions(opts)
	nw := toolx.DetermineNoOfWorkers(n, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("creating pi: requested %d, workers %d", n, nw), log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("creating %d pi", n))
	defer stopActivity()
	var completed atomic.Int64
	var created atomic.Int64
	stopProgress := startProcessInstanceBulkProgress(ctx, log, "create", n, 0, &completed)
	defer stopProgress()
	pics, err := pool.ExecuteNTimes[d.ProcessInstanceCreation](ctx, n, nw, cfg.FailFast, func(ctx context.Context, _ int) (d.ProcessInstanceCreation, error) {
		defer completed.Add(1)
		pi, err := api.CreateProcessInstance(ctx, data, opts...)
		if err == nil {
			created.Add(1)
		}
		return pi, err
	})
	if !cfg.NoWait {
		ok := int(created.Load())
		log.Info(fmt.Sprintf("creating pi done; requested %d, created %d, failed %d", n, ok, n-ok))
	}
	return pics, err
}

// CreateProcessInstances creates each requested process instance in input order and stops on the first error.
func CreateProcessInstances(ctx context.Context, api processInstanceCreator, datas []d.ProcessInstanceData, opts ...services.CallOption) ([]d.ProcessInstanceCreation, error) {
	pics := make([]d.ProcessInstanceCreation, 0, len(datas))
	for _, data := range datas {
		pic, err := api.CreateProcessInstance(ctx, data, opts...)
		if err != nil {
			return nil, err
		}
		pics = append(pics, pic)
	}
	return pics, nil
}

func CancelProcessInstances(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, wantedWorkers int, affectedCount int, opts ...services.CallOption) ([]d.Reporter, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	if affectedCount > lk {
		logging.InfoIfVerbose(fmt.Sprintf("cancelling pi: affected %d, roots %d, workers %d", affectedCount, lk, nw), log, cfg.Verbose)
	} else {
		logging.InfoIfVerbose(fmt.Sprintf("cancelling pi: requested %d, workers %d", lk, nw), log, cfg.Verbose)
	}
	stopActivity := logging.StartActivity(ctx, processInstanceBulkActivity("cancelling", lk, affectedCount))
	defer stopActivity()
	var completed atomic.Int64
	stopProgress := startProcessInstanceBulkProgress(ctx, log, "cancel", lk, affectedCount, &completed)
	defer stopProgress()
	rs, err := pool.ExecuteSlice[string, d.Reporter](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.Reporter, error) {
		defer completed.Add(1)
		resp, _, err := api.CancelProcessInstance(ctx, key, opts...)
		return d.Reporter{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, err
	})
	if !cfg.NoWait {
		t, oks, noks := reporterTotals(rs)
		if affectedCount > t {
			log.Info(fmt.Sprintf("pi cancel done; roots %d, affected %d, ok %d (cancelled/terminal), failed %d", t, affectedCount, oks, noks))
		} else {
			log.Info(fmt.Sprintf("pi cancel done; requested %d, ok %d (cancelled/terminal), failed %d", t, oks, noks))
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
		logging.InfoIfVerbose(fmt.Sprintf("deleting pi: affected %d, roots %d, workers %d", affectedCount, lk, nw), log, cfg.Verbose)
	} else {
		logging.InfoIfVerbose(fmt.Sprintf("deleting pi: requested %d, workers %d", lk, nw), log, cfg.Verbose)
	}
	stopActivity := logging.StartActivity(ctx, processInstanceBulkActivity("deleting", lk, affectedCount))
	defer stopActivity()
	var completed atomic.Int64
	stopProgress := startProcessInstanceBulkProgress(ctx, log, "delete", lk, affectedCount, &completed)
	defer stopProgress()
	rs, err := pool.ExecuteSlice[string, d.Reporter](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.Reporter, error) {
		defer completed.Add(1)
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
			log.Info(fmt.Sprintf("cannot delete pi scope; affected %d, non-terminal present, use --force", affected))
		}
		if affectedCount > t {
			log.Info(fmt.Sprintf("pi delete done; roots %d, affected %d, ok %d, failed %d", t, affectedCount, oks, noks))
		} else {
			log.Info(fmt.Sprintf("pi delete done; requested %d, ok %d, failed %d", t, oks, noks))
		}
	}
	return rs, err
}

func GetProcessInstances(ctx context.Context, api API, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	ukeys := keys.Unique()
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("getting %d pi", len(ukeys)))
	defer stopActivity()
	return api.GetProcessInstances(ctx, ukeys, wantedWorkers, opts...)
}

// startProcessInstanceBulkProgress emits durable progress while long-running
// root-tree operations are still in flight.
func startProcessInstanceBulkProgress(ctx context.Context, log *slog.Logger, action string, roots int, affectedCount int, completed *atomic.Int64) func() {
	if log == nil || roots <= 0 || completed == nil || processInstanceBulkProgressInterval <= 0 {
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(processInstanceBulkProgressInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				doneRoots := int(completed.Load())
				if doneRoots >= roots {
					continue
				}
				if affectedCount > roots {
					log.Info(fmt.Sprintf("pi %s progress; roots %d/%d done, affected %d", action, doneRoots, roots, affectedCount))
					continue
				}
				log.Info(fmt.Sprintf("pi %s progress; requested %d/%d done", action, doneRoots, roots))
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
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
