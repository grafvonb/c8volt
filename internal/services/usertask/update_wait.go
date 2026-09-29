// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// waitForUserTaskVariableUpdate polls the complete effective-variable view
// until every requested name is complete at its frozen scope and value.
func waitForUserTaskVariableUpdate(ctx context.Context, api API, cfg *config.Config, task d.UserTaskVariablePlan, opts ...services.CallOption) error {
	if api == nil {
		return fmt.Errorf("%w: user-task variable confirmation requires a user-task service", d.ErrPrecondition)
	}
	backoff := configuredUserTaskUpdateBackoff(cfg)
	if backoff.Timeout > 0 {
		deadline := time.Now().Add(backoff.Timeout)
		if current, ok := ctx.Deadline(); !ok || deadline.Before(current) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()
		}
	}
	delay := backoff.InitialDelay
	if delay <= 0 {
		delay = 500 * time.Millisecond
	}
	attempts := 0
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("confirm variables for user task %s: %w", task.UserTaskKey, err)
		}
		attempts++
		variables, err := SearchUserTaskEffectiveVariables(ctx, api, task.UserTaskKey, opts...)
		if err != nil {
			return fmt.Errorf("confirm variables for user task %s: %w", task.UserTaskKey, err)
		}
		missing, err := unconfirmedUserTaskVariables(task, variables)
		if err != nil {
			return fmt.Errorf("confirm variables for user task %s: %w", task.UserTaskKey, err)
		}
		if len(missing) == 0 {
			return nil
		}
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			return fmt.Errorf("%w: variables %v not confirmed at planned scopes for user task %s", d.ErrUpstream, missing, task.UserTaskKey)
		}
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("confirm variables for user task %s: %w", task.UserTaskKey, ctx.Err())
		}
	}
}

// unconfirmedUserTaskVariables returns stable requested names whose complete
// effective observation does not match both the frozen scope and value.
func unconfirmedUserTaskVariables(task d.UserTaskVariablePlan, observed []d.ProcessInstanceVariable) ([]string, error) {
	byName := make(map[string]d.ProcessInstanceVariable, len(observed))
	for _, variable := range observed {
		byName[variable.Name] = variable
	}
	type expectedVariable struct {
		name     string
		scopeKey string
		value    any
	}
	expected := make([]expectedVariable, 0, len(task.Additions)+len(task.Changes)+len(task.UnchangedRequested))
	for _, value := range task.Additions {
		expected = append(expected, expectedVariable{name: value.Name, scopeKey: value.ScopeKey, value: value.Value})
	}
	for _, value := range task.Changes {
		expected = append(expected, expectedVariable{name: value.Name, scopeKey: value.ScopeKey, value: value.After})
	}
	for _, value := range task.UnchangedRequested {
		expected = append(expected, expectedVariable{name: value.Name, scopeKey: value.ScopeKey, value: value.Value})
	}
	missing := make([]string, 0)
	for _, wanted := range expected {
		actual, exists := byName[wanted.name]
		if exists && strings.TrimSpace(actual.ScopeKey) == "" {
			return nil, fmt.Errorf("%w: variable %q has no scope key", d.ErrMalformedResponse, wanted.name)
		}
		if exists && !actual.APITruncated {
			if _, ok := normalizeObservedJSONValue(actual.Value); !ok {
				return nil, fmt.Errorf("%w: variable %q contains invalid complete JSON", d.ErrMalformedResponse, wanted.name)
			}
		}
		if !exists || actual.APITruncated || actual.ScopeKey != wanted.scopeKey || !normalizedJSONValuesEqual(wanted.value, actual.Value) {
			missing = append(missing, wanted.name)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

// configuredUserTaskUpdateBackoff safely obtains confirmation timing for
// services constructed without a repository configuration.
func configuredUserTaskUpdateBackoff(cfg *config.Config) config.BackoffConfig {
	if cfg == nil {
		return config.BackoffConfig{}
	}
	return cfg.App.Backoff
}
