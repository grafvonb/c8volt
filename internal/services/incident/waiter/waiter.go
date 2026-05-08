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
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for incident %s to resolve", key))
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
			status := fmt.Sprintf("stopped waiting for incident %s after %d attempts in %s due to context error", key, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: key, Ok: false, Status: status}, fmt.Errorf("%w: %s", err, status)
		}
		attempts++
		incident, err := s.GetIncident(ctx, key, opts...)
		if err == nil && !incidentIsActive(incident) {
			status := fmt.Sprintf("incident %s resolved after %d checks in %s", key, attempts, time.Since(start))
			if attempts == 1 {
				status = fmt.Sprintf("incident %s is already resolved", key)
			}
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: key, Ok: true, Status: status}, nil
		}
		if err != nil {
			if isIncidentAbsentErr(err) {
				status := fmt.Sprintf("incident %s no longer exists after %d checks in %s", key, attempts, time.Since(start))
				log.Debug(status)
				return d.IncidentResolutionResponse{Key: key, Ok: true, Status: status}, nil
			}
			status := fmt.Sprintf("stopped waiting for incident %s after %d attempts in %s due to error", key, attempts, time.Since(start))
			log.Error(status)
			return d.IncidentResolutionResponse{Key: key, Ok: false, Status: status}, fmt.Errorf("%w: %s", err, status)
		}
		logging.InfoIfVerbose(fmt.Sprintf("incident %s currently in state %s; waiting... (attempt #%d)", key, incident.State, attempts), log, cCfg.Verbose)
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			status := fmt.Sprintf("exceeded max_retries (%d) waiting for incident %s to resolve after %d attempts in %s", backoff.MaxRetries, key, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: key, Ok: false, Status: status}, errors.New(status)
		}
		select {
		case <-time.After(delay):
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			status := fmt.Sprintf("stopped waiting for incident %s after %d attempts in %s due to context done", key, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: key, Ok: false, Status: status}, fmt.Errorf("%w: %s", ctx.Err(), status)
		}
	}
}

func WaitForProcessInstanceIncidentsResolved(ctx context.Context, s IncidentWaiter, cfg *config.Config, log *slog.Logger, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	cCfg := services.ApplyCallOptions(opts)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for process instance %s incidents to resolve", processInstanceKey))
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
			status := fmt.Sprintf("stopped waiting for process instance %s incidents after %d attempts in %s due to context error", processInstanceKey, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: status}, fmt.Errorf("%w: %s", err, status)
		}
		attempts++
		incidents, err := s.SearchProcessInstanceIncidents(ctx, processInstanceKey, opts...)
		if err != nil {
			status := fmt.Sprintf("stopped waiting for process instance %s incidents after %d attempts in %s due to error", processInstanceKey, attempts, time.Since(start))
			log.Error(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: status}, fmt.Errorf("%w: %s", err, status)
		}
		active := activeTargetIncidentKeys(processInstanceKey, targets, incidents)
		if len(active) == 0 {
			status := fmt.Sprintf("process instance %s incidents resolved after %d checks in %s", processInstanceKey, attempts, time.Since(start))
			if attempts == 1 {
				status = fmt.Sprintf("process instance %s incidents are already resolved", processInstanceKey)
			}
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: true, Status: status}, nil
		}
		logging.InfoIfVerbose(fmt.Sprintf("process instance %s still has active incident(s) %v; waiting... (attempt #%d)", processInstanceKey, active, attempts), log, cCfg.Verbose)
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			status := fmt.Sprintf("exceeded max_retries (%d) waiting for process instance %s incidents to resolve after %d attempts in %s", backoff.MaxRetries, processInstanceKey, attempts, time.Since(start))
			log.Debug(status)
			return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: status}, errors.New(status)
		}
		select {
		case <-time.After(delay):
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			status := fmt.Sprintf("stopped waiting for process instance %s incidents after %d attempts in %s due to context done", processInstanceKey, attempts, time.Since(start))
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
