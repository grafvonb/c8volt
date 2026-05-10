// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"log/slog"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
)

func WaitForProcessInstancesState(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, desired d.States, wantedWorkers int, opts ...services.CallOption) (d.StateResponses, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("waiting for %d unique process instance(s) to reach desired state(s) %v using %d worker(s)", lk, desired, nw), log, cfg.Verbose)
	return api.WaitForProcessInstancesState(ctx, ukeys, desired, nw, opts...)
}

func WaitForProcessInstancesExpectation(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, request d.ProcessInstanceExpectationRequest, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceExpectationResponses, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("waiting for %d unique process instance(s) to satisfy expectation(s) using %d worker(s)", lk, nw), log, cfg.Verbose)
	return api.WaitForProcessInstancesExpectation(ctx, ukeys, request, nw, opts...)
}
