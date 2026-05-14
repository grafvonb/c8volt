// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx/logging"
)

type IncidentWaiter interface {
	GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error)
	SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
}

func WaitForIncidentResolved(ctx context.Context, s IncidentWaiter, cfg *config.Config, log *slog.Logger, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for incident %s resolve", key))
	defer stopActivity()
	backoff := cfg.App.Backoff
	start := time.Now()
	if backoff.Timeout > 0 {
		deadline := time.Now().Add(backoff.Timeout)
		if dl, ok := ctx.Deadline(); !ok || deadline.Before(dl) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()
		}
	}

	attempts := 0
	delay := backoff.InitialDelay
	for {
		if err := ctx.Err(); err != nil {
			status := fmt.Sprintf("incident %s wait stopped; attempts %d, elapsed %s, reason context error", key, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: key, Ok: false, Status: status}, fmt.Errorf("%w: %s", err, status)
		}
		attempts++
		incident, err := s.GetIncident(ctx, key, opts...)
		if err == nil && !incidentIsActive(incident) {
			status := fmt.Sprintf("incident %s resolved; checks %d, elapsed %s", key, attempts, time.Since(start))
			if attempts == 1 {
				status = fmt.Sprintf("incident %s resolved", key)
			}
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: key, Ok: true, Status: status}, nil
		}
		if err != nil {
			if isIncidentAbsentErr(err) {
				status := fmt.Sprintf("incident %s absent; checks %d, elapsed %s", key, attempts, time.Since(start))
				log.Debug(status)
				return d.IncidentResolutionResponse{Key: key, Ok: true, Status: status}, nil
			}
			status := fmt.Sprintf("incident %s wait stopped; attempts %d, elapsed %s, reason error", key, attempts, time.Since(start))
			log.Error(status)
			return d.IncidentResolutionResponse{Key: key, Ok: false, Status: status}, fmt.Errorf("%w: %s", err, status)
		}
		logging.InfoIfVerbose(fmt.Sprintf("incident %s waiting; state %s, attempt %d", key, incident.State, attempts), log, cCfg.Verbose)
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			status := fmt.Sprintf("incident %s wait exceeded retries; max %d, attempts %d, elapsed %s", key, backoff.MaxRetries, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: key, Ok: false, Status: status}, errors.New(status)
		}
		select {
		case <-time.After(delay):
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			status := fmt.Sprintf("incident %s wait stopped; attempts %d, elapsed %s, reason context done", key, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: key, Ok: false, Status: status}, fmt.Errorf("%w: %s", ctx.Err(), status)
		}
	}
}

func WaitForProcessInstanceIncidentsResolved(ctx context.Context, s IncidentWaiter, cfg *config.Config, log *slog.Logger, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for pi %s incidents resolve", processInstanceKey))
	defer stopActivity()
	backoff := cfg.App.Backoff
	start := time.Now()
	if backoff.Timeout > 0 {
		deadline := time.Now().Add(backoff.Timeout)
		if dl, ok := ctx.Deadline(); !ok || deadline.Before(dl) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()
		}
	}

	targets := stringSet(incidentKeys)
	attempts := 0
	delay := backoff.InitialDelay
	for {
		if err := ctx.Err(); err != nil {
			status := fmt.Sprintf("pi %s incident wait stopped; attempts %d, elapsed %s, reason context error", processInstanceKey, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: status}, fmt.Errorf("%w: %s", err, status)
		}
		attempts++
		incidents, err := s.SearchProcessInstanceIncidents(ctx, processInstanceKey, opts...)
		if err != nil {
			status := fmt.Sprintf("pi %s incident wait stopped; attempts %d, elapsed %s, reason error", processInstanceKey, attempts, time.Since(start))
			log.Error(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: status}, fmt.Errorf("%w: %s", err, status)
		}
		active := activeTargetIncidentKeys(processInstanceKey, targets, incidents)
		if len(active) == 0 {
			status := fmt.Sprintf("pi %s incidents resolved; checks %d, elapsed %s", processInstanceKey, attempts, time.Since(start))
			if attempts == 1 {
				status = fmt.Sprintf("pi %s incidents resolved", processInstanceKey)
			}
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: true, Status: status}, nil
		}
		logging.InfoIfVerbose(fmt.Sprintf("pi %s incidents waiting; active %v, attempt %d", processInstanceKey, active, attempts), log, cCfg.Verbose)
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			status := fmt.Sprintf("pi %s incident wait exceeded retries; max %d, attempts %d, elapsed %s", processInstanceKey, backoff.MaxRetries, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: status}, errors.New(status)
		}
		select {
		case <-time.After(delay):
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			status := fmt.Sprintf("pi %s incident wait stopped; attempts %d, elapsed %s, reason context done", processInstanceKey, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: status}, fmt.Errorf("%w: %s", ctx.Err(), status)
		}
	}
}

func IncidentIsActive(incident d.ProcessInstanceIncidentDetail) bool {
	return incidentIsActive(incident)
}

func incidentIsActive(incident d.ProcessInstanceIncidentDetail) bool {
	state := strings.ToUpper(strings.TrimSpace(incident.State))
	return state == "" || state == "ACTIVE" || state == "PENDING"
}

func activeTargetIncidentKeys(processInstanceKey string, targets map[string]struct{}, incidents []d.ProcessInstanceIncidentDetail) []string {
	active := make([]string, 0, len(targets))
	for _, incident := range incidents {
		if incident.ProcessInstanceKey != processInstanceKey {
			continue
		}
		if _, ok := targets[incident.IncidentKey]; !ok || !incidentIsActive(incident) {
			continue
		}
		active = append(active, incident.IncidentKey)
	}
	return active
}

func stringSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}

func isIncidentAbsentErr(err error) bool {
	if errors.Is(err, d.ErrNotFound) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist")
}
