// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"
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
var processInstanceBulkStallProgressThreshold = 2

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
	progress := newProcessInstanceBulkProgressTracker(actionProgressPhase("create"))
	var created atomic.Int64
	stopProgress := func() {}
	if !cfg.SuppressWorkflowDetailLogs {
		stopProgress = startProcessInstanceBulkProgress(ctx, log, "create", n, 0, &completed, progress)
	}
	defer stopProgress()
	pics, err := pool.ExecuteNTimes[d.ProcessInstanceCreation](ctx, n, nw, cfg.FailFast, func(ctx context.Context, _ int) (d.ProcessInstanceCreation, error) {
		work := progress.Start(data.BpmnProcessId)
		defer completed.Add(1)
		defer progress.Done(work)
		pi, err := api.CreateProcessInstance(ctx, data, opts...)
		if err == nil {
			created.Add(1)
		}
		return pi, err
	})
	if !cfg.NoWait && !cfg.SuppressWorkflowDetailLogs {
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
	progress := newProcessInstanceBulkProgressTracker(actionProgressPhase("cancel"))
	stopProgress := func() {}
	if !cfg.SuppressWorkflowDetailLogs {
		stopProgress = startProcessInstanceBulkProgress(ctx, log, "cancel", lk, affectedCount, &completed, progress)
	}
	defer stopProgress()
	rs, err := pool.ExecuteSlice[string, d.Reporter](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.Reporter, error) {
		work := progress.Start(key)
		defer completed.Add(1)
		defer progress.Done(work)
		resp, _, err := api.CancelProcessInstance(ctx, key, opts...)
		return d.Reporter{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, err
	})
	if !cfg.NoWait && !cfg.SuppressWorkflowDetailLogs {
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
	progress := newProcessInstanceBulkProgressTracker(actionProgressPhase("delete"))
	stopProgress := func() {}
	if !cfg.SuppressWorkflowDetailLogs {
		stopProgress = startProcessInstanceBulkProgress(ctx, log, "delete", lk, affectedCount, &completed, progress)
	}
	defer stopProgress()
	rs, err := pool.ExecuteSlice[string, d.Reporter](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.Reporter, error) {
		work := progress.Start(key)
		defer completed.Add(1)
		defer progress.Done(work)
		resp, err := api.DeleteProcessInstance(ctx, key, opts...)
		return d.Reporter{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, err
	})
	if !cfg.NoWait && !cfg.SuppressWorkflowDetailLogs {
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
func startProcessInstanceBulkProgress(ctx context.Context, log *slog.Logger, action string, roots int, affectedCount int, completed *atomic.Int64, tracker *processInstanceBulkProgressTracker) func() {
	if log == nil || roots <= 0 || completed == nil || processInstanceBulkProgressInterval <= 0 {
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(processInstanceBulkProgressInterval)
		defer ticker.Stop()
		lastDoneRoots := -1
		unchangedTicks := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				doneRoots := int(completed.Load())
				if doneRoots >= roots {
					continue
				}
				if doneRoots == lastDoneRoots {
					unchangedTicks++
				} else {
					lastDoneRoots = doneRoots
					unchangedTicks = 0
				}
				if affectedCount > roots {
					log.Info(fmt.Sprintf("pi %s progress; roots %d/%d done, affected %d", action, doneRoots, roots, affectedCount))
				} else {
					log.Info(fmt.Sprintf("pi %s progress; requested %d/%d done", action, doneRoots, roots))
				}
				if unchangedTicks >= processInstanceBulkStallProgressThreshold {
					logOldestProcessInstanceBulkWork(log, action, tracker, unchangedTicks)
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

type processInstanceBulkWork struct {
	id        int64
	key       string
	phase     string
	startedAt time.Time
}

type processInstanceBulkProgressTracker struct {
	mu       sync.Mutex
	nextID   int64
	phase    string
	inFlight map[int64]processInstanceBulkWork
}

func newProcessInstanceBulkProgressTracker(phase string) *processInstanceBulkProgressTracker {
	return &processInstanceBulkProgressTracker{
		phase:    phase,
		inFlight: make(map[int64]processInstanceBulkWork),
	}
}

func (t *processInstanceBulkProgressTracker) Start(key string) processInstanceBulkWork {
	if t == nil {
		return processInstanceBulkWork{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nextID++
	work := processInstanceBulkWork{
		id:        t.nextID,
		key:       key,
		phase:     t.phase,
		startedAt: time.Now(),
	}
	t.inFlight[work.id] = work
	return work
}

func (t *processInstanceBulkProgressTracker) Done(work processInstanceBulkWork) {
	if t == nil || work.id == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.inFlight, work.id)
}

func (t *processInstanceBulkProgressTracker) Oldest(now time.Time, limit int) []processInstanceBulkWork {
	if t == nil || limit <= 0 {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	items := make([]processInstanceBulkWork, 0, len(t.inFlight))
	for _, item := range t.inFlight {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].startedAt.Before(items[j].startedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	for i := range items {
		if items[i].startedAt.After(now) {
			items[i].startedAt = now
		}
	}
	return items
}

func logOldestProcessInstanceBulkWork(log *slog.Logger, action string, tracker *processInstanceBulkProgressTracker, unchangedTicks int) {
	if log == nil || tracker == nil || processInstanceBulkStallProgressThreshold <= 0 || unchangedTicks < processInstanceBulkStallProgressThreshold {
		return
	}
	now := time.Now()
	items := tracker.Oldest(now, 3)
	if len(items) == 0 {
		log.Info(fmt.Sprintf("pi %s progress unchanged; no root completed for %d progress interval(s)", action, unchangedTicks+1))
		return
	}
	for i, item := range items {
		elapsed := now.Sub(item.startedAt).Round(time.Second)
		prefix := "slow root"
		if i > 0 {
			prefix = "also in flight"
		}
		log.Info(fmt.Sprintf("pi %s %s; root %s, phase %s, elapsed %s, no root completed for %d progress interval(s)", action, prefix, item.key, item.phase, elapsed, unchangedTicks+1))
	}
}

func actionProgressPhase(action string) string {
	switch action {
	case "create":
		return "create request"
	case "cancel":
		return "cancel request or wait"
	case "delete":
		return "delete request or wait"
	default:
		return action + " request"
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
