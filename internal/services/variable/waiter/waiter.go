// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// VariableWaiter searches process-instance variables for confirmation loops.
type VariableWaiter interface {
	SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error)
}

// WaitForProcessInstanceVariables polls until requested process-scope variable values are visible.
func WaitForProcessInstanceVariables(ctx context.Context, s VariableWaiter, cfg *config.Config, key string, requested map[string]any, opts ...services.CallOption) ([]string, error) {
	backoff := cfg.App.Backoff
	if backoff.Timeout > 0 {
		deadline := time.Now().Add(backoff.Timeout)
		if dl, ok := ctx.Deadline(); !ok || deadline.Before(dl) {
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
			return requestedVariableNames(requested), err
		}
		attempts++
		variables, err := s.SearchProcessInstanceVariables(ctx, key, opts...)
		if err != nil {
			return nil, err
		}
		missing := MissingRequestedVariables(key, requested, variables)
		if len(missing) == 0 {
			return nil, nil
		}
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			return missing, nil
		}
		select {
		case <-time.After(delay):
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			return missing, ctx.Err()
		}
	}
}

// MissingRequestedVariables returns requested names whose process-scope values are absent or different.
func MissingRequestedVariables(key string, requested map[string]any, observed []d.ProcessInstanceVariable) []string {
	byName := make(map[string]d.ProcessInstanceVariable, len(observed))
	for _, variable := range processScopeVariablesForInstance(key, observed) {
		byName[variable.Name] = variable
	}
	missing := requestedVariableNames(requested)
	out := missing[:0]
	for _, name := range missing {
		variable, ok := byName[name]
		if !ok || !normalizedJSONValuesEqual(requested[name], variable.Value) {
			out = append(out, name)
		}
	}
	return out
}

// processScopeVariablesForInstance keeps variables owned by the process instance scope.
func processScopeVariablesForInstance(key string, variables []d.ProcessInstanceVariable) []d.ProcessInstanceVariable {
	out := make([]d.ProcessInstanceVariable, 0, len(variables))
	for _, variable := range variables {
		if variable.ProcessInstanceKey == key && variable.ScopeKey == key {
			out = append(out, variable)
		}
	}
	return out
}

// requestedVariableNames returns requested variable names in stable order.
func requestedVariableNames(requested map[string]any) []string {
	names := make([]string, 0, len(requested))
	for name := range requested {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// normalizedJSONValuesEqual compares requested values with raw Camunda JSON values.
func normalizedJSONValuesEqual(requested any, observedRaw string) bool {
	var observed any
	if err := json.Unmarshal([]byte(observedRaw), &observed); err != nil {
		if s, ok := requested.(string); ok {
			return observedRaw == s
		}
		return false
	}
	requestedNormalized, ok := normalizeJSONValue(requested)
	if !ok {
		return false
	}
	observedNormalized, ok := normalizeJSONValue(observed)
	if !ok {
		return false
	}
	return reflect.DeepEqual(requestedNormalized, observedNormalized)
}

// normalizeJSONValue round-trips a value through JSON using stable numeric handling.
func normalizeJSONValue(v any) (any, bool) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	var out any
	if err := decoder.Decode(&out); err != nil {
		return nil, false
	}
	return out, true
}
