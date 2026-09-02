// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

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
	if result.Request.NoCleanup {
		result.Cleanup = retainedAPILatencyCleanupRecords(result.Ownership)
		return nil
	}
	var cleanupErr error
	for _, key := range apiLatencySortedUniqueKeys(result.Ownership.ProcessInstanceKeys) {
		record := d.APILatencyCleanupRecord{
			ResourceType:   d.APILatencyCleanupResourceProcessInstance,
			Key:            key,
			Status:         d.APILatencyCleanupStatusDeleted,
			Classification: d.APILatencyClassificationSuccess,
		}
		_, err := s.piAPI.DeleteProcessInstance(ctx, key, opts...)
		if err != nil {
			record.Status = d.APILatencyCleanupStatusFailed
			record.Classification = ClassifyAPILatencyError(err)
			record.RecoveryCommand = fmt.Sprintf("c8volt delete process-instance --key %s --force --auto-confirm", key)
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("delete API latency process instance %s: %w", key, err))
		}
		result.Cleanup = append(result.Cleanup, record)
	}
	if result.Ownership.ProcessDefinitionKey != "" {
		record := d.APILatencyCleanupRecord{
			ResourceType:   d.APILatencyCleanupResourceProcessDefinition,
			Key:            result.Ownership.ProcessDefinitionKey,
			Status:         d.APILatencyCleanupStatusDeleted,
			Classification: d.APILatencyClassificationSuccess,
		}
		_, err := s.resourceAPI.Delete(ctx, result.Ownership.ProcessDefinitionKey, opts...)
		if err != nil {
			record.Status = d.APILatencyCleanupStatusFailed
			record.Classification = ClassifyAPILatencyError(err)
			record.RecoveryCommand = fmt.Sprintf("c8volt delete process-definition --key %s --auto-confirm", result.Ownership.ProcessDefinitionKey)
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("delete API latency process definition %s: %w", result.Ownership.ProcessDefinitionKey, err))
		}
		result.Cleanup = append(result.Cleanup, record)
	}
	return cleanupErr
}

// retainedAPILatencyCleanupRecords records explicit no-cleanup resources without treating retention as failure.
func retainedAPILatencyCleanupRecords(ownership *d.APILatencyOwnership) []d.APILatencyCleanupRecord {
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
			RecoveryCommand: fmt.Sprintf("c8volt delete process-instance --key %s --force --auto-confirm", key),
		})
	}
	if ownership.ProcessDefinitionKey != "" {
		out = append(out, d.APILatencyCleanupRecord{
			ResourceType:    d.APILatencyCleanupResourceProcessDefinition,
			Key:             ownership.ProcessDefinitionKey,
			Status:          d.APILatencyCleanupStatusRetained,
			Classification:  d.APILatencyClassificationUnavailable,
			RecoveryCommand: fmt.Sprintf("c8volt delete process-definition --key %s --auto-confirm", ownership.ProcessDefinitionKey),
		})
	}
	return out
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
