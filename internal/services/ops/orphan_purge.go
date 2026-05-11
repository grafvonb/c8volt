// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"log/slog"
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

	discovery := pisvc.OrphanDiscovery{
		Filter: request.Selection,
	}
	if request.DiscoveredKeys != nil {
		discovery.Keys = request.DiscoveredKeys.Unique()
	} else {
		var err error
		discovery, err = pisvc.DiscoverOrphanProcessInstances(ctx, s.piAPI, pisvc.OrphanDiscoveryRequest{
			Filter:    request.Selection,
			BatchSize: request.BatchSize,
			Limit:     request.Limit,
		}, opts...)
		if err != nil {
			result.Discovery.Status = d.OpsWorkflowStepStatusFailed
			result.Discovery.Errors = []string{err.Error()}
			return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomeFailed, err)
		}
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
		RequiresConfirmation: !request.DryRun,
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

	cfg := services.ApplyCallOptions(opts)
	if !cfg.Force && len(plan.RequiresCancelBeforeDelete) > 0 {
		err = fmt.Errorf("refusing to delete orphan process-instance scope: %d affected process instance(s) are not in a final state; no delete request was submitted; use --force to cancel the entire affected scope before delete", len(plan.RequiresCancelBeforeDelete))
		result.Deletion.Status = d.OpsWorkflowStepStatusBlocked
		result.Deletion.Errors = []string{err.Error()}
		return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomeFailed, err)
	}

	result.DeleteRequested = true
	reports, err := pisvc.DeleteProcessInstances(ctx, s.piAPI, slog.Default(), plan.Roots, request.Workers, len(plan.Collected), opts...)
	result.Deletion = d.DeletionResult{
		Status:    deletionStatusForReports(reports, cfg.NoWait, err),
		Items:     reports,
		Errors:    deletionErrors(err),
		Submitted: len(reports) > 0,
		Confirmed: err == nil && !cfg.NoWait && allReportsOK(reports),
	}
	if err != nil {
		return finishOrphanPurgeResult(result, deletionOutcomeForReports(reports), fmt.Errorf("delete orphan process instances: %w", err))
	}
	return finishOrphanPurgeResult(result, deletionOutcomeForReports(reports), nil)
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

func deletionStatusForReports(reports []d.Reporter, noWait bool, err error) d.OpsWorkflowStepStatus {
	if err != nil || !allReportsOK(reports) {
		return d.OpsWorkflowStepStatusFailed
	}
	if noWait {
		return d.OpsWorkflowStepStatusSubmitted
	}
	return d.OpsWorkflowStepStatusConfirmed
}

func deletionOutcomeForReports(reports []d.Reporter) d.OrphanPurgeOutcome {
	if len(reports) == 0 {
		return d.OrphanPurgeOutcomeFailed
	}
	ok := 0
	for _, report := range reports {
		if report.Ok {
			ok++
		}
	}
	switch ok {
	case len(reports):
		return d.OrphanPurgeOutcomeDeleted
	case 0:
		return d.OrphanPurgeOutcomeFailed
	default:
		return d.OrphanPurgeOutcomePartiallyFailed
	}
}

func deletionErrors(err error) []string {
	if err == nil {
		return nil
	}
	return []string{err.Error()}
}

func allReportsOK(reports []d.Reporter) bool {
	if len(reports) == 0 {
		return false
	}
	for _, report := range reports {
		if !report.Ok {
			return false
		}
	}
	return true
}
