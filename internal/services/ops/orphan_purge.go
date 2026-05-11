// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

const orphanPurgeReportSchemaVersion = "ops.orphan-process-instances.v1"

func (s *Service) PurgeOrphanProcessInstances(ctx context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error) {
	started := request.StartedAt
	if started.IsZero() {
		started = time.Now().UTC()
		request.StartedAt = started
	}
	result := newOrphanPurgeResult(request)

	discovery, err := pisvc.DiscoverOrphanProcessInstances(ctx, s.piAPI, pisvc.OrphanDiscoveryRequest{
		Filter:    request.Selection,
		BatchSize: request.BatchSize,
		Limit:     request.Limit,
	}, opts...)
	if err != nil {
		result.Discovery.Status = d.OpsWorkflowStepStatusFailed
		result.Discovery.Errors = []string{err.Error()}
		return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomeFailed, err)
	}

	result.Discovery = d.OrphanDiscoveryResult{
		Status:  d.OpsWorkflowStepStatusPlanned,
		Filters: discovery.Filter,
		Keys:    discovery.Keys,
		Count:   len(discovery.Keys),
	}
	if len(discovery.Keys) == 0 {
		result.DeletionPlan.Status = d.OpsWorkflowStepStatusSkipped
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomePlanned, nil)
	}

	plan, err := pisvc.DryRunCancelOrDeletePlan(ctx, s.piAPI, discovery.Keys, request.Workers, opts...)
	result.DeletionPlan = d.DeletionPlan{
		Status:               d.OpsWorkflowStepStatusPlanned,
		RequestedKeys:        discovery.Keys,
		AffectedKeys:         plan.Collected,
		RootKeys:             plan.Roots,
		RequiresConfirmation: false,
		DryRunPreview:        plan,
	}
	if err != nil {
		result.DeletionPlan.Status = d.OpsWorkflowStepStatusFailed
		result.DeletionPlan.Errors = []string{err.Error()}
		return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomeFailed, fmt.Errorf("orphan purge delete-plan validation: %w", err))
	}

	if request.DryRun {
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomePlanned, nil)
	}

	result.Deletion.Status = d.OpsWorkflowStepStatusBlocked
	err = fmt.Errorf("%w: confirmed orphan process-instance purge is not implemented yet", d.ErrUnsupported)
	result.Errors = append(result.Errors, err.Error())
	return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomeFailed, err)
}

func newOrphanPurgeResult(request d.OrphanPurgeRequest) d.OrphanPurgeResult {
	return d.OrphanPurgeResult{
		Request: request,
		Report: d.OrphanPurgeReport{
			SchemaVersion:    orphanPurgeReportSchemaVersion,
			CommandName:      request.CommandName,
			StartedAt:        request.StartedAt,
			DryRun:           request.DryRun,
			AutoConfirm:      request.AutoConfirm,
			Automation:       request.Automation,
			SelectionFilters: request.Selection,
			Outcome:          d.OrphanPurgeOutcomeFailed,
		},
		Outcome: d.OrphanPurgeOutcomeFailed,
	}
}

func finishOrphanPurgeResult(result d.OrphanPurgeResult, outcome d.OrphanPurgeOutcome, err error) (d.OrphanPurgeResult, error) {
	finished := time.Now().UTC()
	result.Outcome = outcome
	result.Report.Outcome = outcome
	result.Report.FinishedAt = finished
	if !result.Request.StartedAt.IsZero() {
		result.Report.Duration = finished.Sub(result.Request.StartedAt).String()
	}
	result.Report.Discovery = result.Discovery
	result.Report.DeletionPlan = result.DeletionPlan
	result.Report.Deletion = result.Deletion
	result.Report.DeleteRequested = result.DeleteRequested
	result.Report.Errors = append([]string(nil), result.Errors...)
	if err != nil {
		msg := err.Error()
		result.Errors = appendIfMissing(result.Errors, msg)
		result.Report.Errors = appendIfMissing(result.Report.Errors, msg)
	}
	return result, err
}

func appendIfMissing(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}
