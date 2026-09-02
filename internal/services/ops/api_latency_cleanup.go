// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx/poller"
)

// apiLatencyCleanupCompletionBudget is the bounded independent cleanup opportunity after active execution stops.
var apiLatencyCleanupCompletionBudget = poller.DefaultCompletionTimeout

type apiLatencyOwnershipRegistry struct {
	mu        sync.Mutex
	ownership *d.APILatencyOwnership
	piKeys    map[string]struct{}
}

// newAPILatencyOwnershipRegistry protects active ownership mutations made by concurrent stage workers.
func newAPILatencyOwnershipRegistry(ownership *d.APILatencyOwnership) *apiLatencyOwnershipRegistry {
	if ownership == nil {
		ownership = &d.APILatencyOwnership{}
	}
	registry := &apiLatencyOwnershipRegistry{
		ownership: ownership,
		piKeys:    make(map[string]struct{}, len(ownership.ProcessInstanceKeys)),
	}
	for _, key := range ownership.ProcessInstanceKeys {
		if key == "" {
			continue
		}
		registry.piKeys[key] = struct{}{}
	}
	ownership.ProcessInstanceKeys = apiLatencySortedUniqueKeys(ownership.ProcessInstanceKeys)
	return registry
}

// registerProcessDefinitionKey records the exact deployed process-definition key before dependent work can use it.
func (r *apiLatencyOwnershipRegistry) registerProcessDefinitionKey(key string) {
	if key == "" {
		return
	}
	r.mu.Lock()
	r.ownership.ProcessDefinitionKey = key
	r.mu.Unlock()
}

// registerProcessInstanceKey records exact process-instance cleanup authority from concurrent create calls.
func (r *apiLatencyOwnershipRegistry) registerProcessInstanceKey(key string) {
	if key == "" {
		return
	}
	r.mu.Lock()
	if _, ok := r.piKeys[key]; !ok {
		r.piKeys[key] = struct{}{}
		r.ownership.ProcessInstanceKeys = append(r.ownership.ProcessInstanceKeys, key)
	}
	r.mu.Unlock()
}

// snapshot returns deterministic exact ownership evidence without exposing mutable registry state.
func (r *apiLatencyOwnershipRegistry) snapshot() *d.APILatencyOwnership {
	if r == nil || r.ownership == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := *r.ownership
	out.ProcessInstanceKeys = apiLatencySortedUniqueKeys(out.ProcessInstanceKeys)
	return &out
}

// finalizeAPILatencyCleanup records retained resources or deletes exact owned keys after active execution.
func (s *Service) finalizeAPILatencyCleanup(ctx context.Context, result *d.APILatencyResult, opts ...services.CallOption) error {
	if result == nil || result.Ownership == nil {
		return nil
	}
	if result.Plan.Cleanup != nil && result.Plan.Cleanup.IndependentBudget <= 0 {
		result.Plan.Cleanup.IndependentBudget = apiLatencyCleanupCompletionBudget
	}
	if result.Request.NoCleanup {
		result.Cleanup = retainedAPILatencyCleanupRecords(result.Ownership, result.Plan.Cleanup)
		return nil
	}
	cleanupCtx, cancel := apiLatencyCleanupContext(ctx, result)
	defer cancel()

	var cleanupErr error
	for _, key := range apiLatencySortedUniqueKeys(result.Ownership.ProcessInstanceKeys) {
		if err := cleanupCtx.Err(); err != nil {
			result.Cleanup = append(result.Cleanup, unknownAPILatencyCleanupRecord(d.APILatencyCleanupResourceProcessInstance, key, result.Plan.Cleanup, err))
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("cleanup API latency process instance %s: %w", key, err))
			continue
		}
		record := d.APILatencyCleanupRecord{
			ResourceType:   d.APILatencyCleanupResourceProcessInstance,
			Key:            key,
			Status:         d.APILatencyCleanupStatusDeleted,
			Classification: d.APILatencyClassificationSuccess,
		}
		_, err := s.piAPI.DeleteProcessInstance(cleanupCtx, key, opts...)
		if err != nil {
			record.Status = failedAPILatencyCleanupStatus(err)
			record.Classification = ClassifyAPILatencyError(err)
			record.RecoveryCommand = apiLatencyCleanupRecoveryCommand(d.APILatencyCleanupResourceProcessInstance, key, result.Plan.Cleanup)
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("delete API latency process instance %s: %w", key, err))
		}
		result.Cleanup = append(result.Cleanup, record)
	}
	if result.Ownership.ProcessDefinitionKey != "" {
		if err := cleanupCtx.Err(); err != nil {
			result.Cleanup = append(result.Cleanup, unknownAPILatencyCleanupRecord(d.APILatencyCleanupResourceProcessDefinition, result.Ownership.ProcessDefinitionKey, result.Plan.Cleanup, err))
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("cleanup API latency process definition %s: %w", result.Ownership.ProcessDefinitionKey, err))
			return cleanupErr
		}
		record := d.APILatencyCleanupRecord{
			ResourceType:   d.APILatencyCleanupResourceProcessDefinition,
			Key:            result.Ownership.ProcessDefinitionKey,
			Status:         d.APILatencyCleanupStatusDeleted,
			Classification: d.APILatencyClassificationSuccess,
		}
		_, err := s.resourceAPI.Delete(cleanupCtx, result.Ownership.ProcessDefinitionKey, opts...)
		if err != nil {
			record.Status = failedAPILatencyCleanupStatus(err)
			record.Classification = ClassifyAPILatencyError(err)
			record.RecoveryCommand = apiLatencyCleanupRecoveryCommand(d.APILatencyCleanupResourceProcessDefinition, result.Ownership.ProcessDefinitionKey, result.Plan.Cleanup)
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("delete API latency process definition %s: %w", result.Ownership.ProcessDefinitionKey, err))
		}
		result.Cleanup = append(result.Cleanup, record)
	}
	return cleanupErr
}

// apiLatencyCleanupContext detaches cleanup from caller cancellation while preserving values and enforcing a finite budget.
func apiLatencyCleanupContext(ctx context.Context, result *d.APILatencyResult) (context.Context, context.CancelFunc) {
	budget := apiLatencyCleanupBudget(result)
	return context.WithTimeout(context.WithoutCancel(ctx), budget)
}

// apiLatencyCleanupBudget returns the planned cleanup budget or the repository default when planning omitted it.
func apiLatencyCleanupBudget(result *d.APILatencyResult) time.Duration {
	if result != nil && result.Plan.Cleanup != nil && result.Plan.Cleanup.IndependentBudget > 0 {
		return result.Plan.Cleanup.IndependentBudget
	}
	return apiLatencyCleanupCompletionBudget
}

// failedAPILatencyCleanupStatus treats cleanup context termination as unknown final ownership state.
func failedAPILatencyCleanupStatus(err error) d.APILatencyCleanupStatus {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return d.APILatencyCleanupStatusUnknown
	}
	return d.APILatencyCleanupStatusFailed
}

// unknownAPILatencyCleanupRecord records resources left unattempted after the cleanup budget is exhausted.
func unknownAPILatencyCleanupRecord(resourceType d.APILatencyCleanupResourceType, key string, plan *d.APILatencyCleanupPlan, err error) d.APILatencyCleanupRecord {
	return d.APILatencyCleanupRecord{
		ResourceType:    resourceType,
		Key:             key,
		Status:          d.APILatencyCleanupStatusUnknown,
		Classification:  ClassifyAPILatencyError(err),
		RecoveryCommand: apiLatencyCleanupRecoveryCommand(resourceType, key, plan),
	}
}

// retainedAPILatencyCleanupRecords records explicit no-cleanup resources without treating retention as failure.
func retainedAPILatencyCleanupRecords(ownership *d.APILatencyOwnership, plan *d.APILatencyCleanupPlan) []d.APILatencyCleanupRecord {
	if ownership == nil {
		return nil
	}
	piKeys := apiLatencySortedUniqueKeys(ownership.ProcessInstanceKeys)
	out := make([]d.APILatencyCleanupRecord, 0, len(piKeys)+1)
	for _, key := range piKeys {
		out = append(out, d.APILatencyCleanupRecord{
			ResourceType:    d.APILatencyCleanupResourceProcessInstance,
			Key:             key,
			Status:          d.APILatencyCleanupStatusRetained,
			Classification:  d.APILatencyClassificationUnavailable,
			RecoveryCommand: apiLatencyCleanupRecoveryCommand(d.APILatencyCleanupResourceProcessInstance, key, plan),
		})
	}
	if ownership.ProcessDefinitionKey != "" {
		out = append(out, d.APILatencyCleanupRecord{
			ResourceType:    d.APILatencyCleanupResourceProcessDefinition,
			Key:             ownership.ProcessDefinitionKey,
			Status:          d.APILatencyCleanupStatusRetained,
			Classification:  d.APILatencyClassificationUnavailable,
			RecoveryCommand: apiLatencyCleanupRecoveryCommand(d.APILatencyCleanupResourceProcessDefinition, ownership.ProcessDefinitionKey, plan),
		})
	}
	return out
}

// apiLatencyCleanupRecoveryCommand formats exact-key recovery commands only when the command is capability-safe.
func apiLatencyCleanupRecoveryCommand(resourceType d.APILatencyCleanupResourceType, key string, plan *d.APILatencyCleanupPlan) string {
	switch resourceType {
	case d.APILatencyCleanupResourceProcessInstance:
		return fmt.Sprintf("c8volt delete process-instance --key %s --force --auto-confirm", key)
	case d.APILatencyCleanupResourceProcessDefinition:
		if plan != nil && !plan.Supported {
			return ""
		}
		return fmt.Sprintf("c8volt delete process-definition --key %s --auto-confirm", key)
	default:
		return ""
	}
}

// apiLatencySortedUniqueKeys returns stable exact-key ordering for ownership, retention, and cleanup evidence.
func apiLatencySortedUniqueKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
