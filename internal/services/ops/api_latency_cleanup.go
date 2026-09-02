// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

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
	for _, key := range result.Ownership.ProcessInstanceKeys {
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
	out := make([]d.APILatencyCleanupRecord, 0, len(ownership.ProcessInstanceKeys)+1)
	for _, key := range ownership.ProcessInstanceKeys {
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
