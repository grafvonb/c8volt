// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
)

// cleanupSmokeTestResources removes the owned instance scope before evaluating every exact deployed definition key.
func (s *Service) cleanupSmokeTestResources(ctx context.Context, result d.SmokeTestResult, opts ...services.CallOption) (d.SmokeTestCleanupResult, error) {
	cleanup := d.SmokeTestCleanupResult{
		NoCleanup:         result.Request.NoCleanup,
		DefinitionResults: smokeTestInitialDefinitionResults(result.Deployment.ProcessDefinitions),
	}
	if result.Request.NoCleanup {
		cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.RetainedProcessInstanceKeys = append(typex.Keys(nil), result.Run.ProcessInstanceKeys...)
		for i := range cleanup.DefinitionResults {
			cleanup.DefinitionResults[i].Disposition = d.SmokeTestDefinitionCleanupDispositionRetained
			cleanup.DefinitionResults[i].Reason = "cleanup disabled (--no-cleanup)"
		}
		smokeTestApplyMainDefinitionProjection(&cleanup, result.Deployment)
		cleanup.Errors = append(cleanup.Errors, result.Deployment.Errors...)
		cleanup.DefinitionSummary = smokeTestDefinitionCleanupSummary(cleanup.DefinitionResults, result.Deployment.DefinitionScopeComplete, cleanup.Errors)
		if len(cleanup.Errors) > 0 {
			return cleanup, errors.New(strings.Join(cleanup.Errors, "; "))
		}
		return cleanup, nil
	}

	piCleanup, cleanupScope, err := smokeTestCleanupProcessInstances(ctx, s.piAPI, s.log, result.Run.ProcessInstanceKeys, result.Request.Workers, opts...)
	cleanup.ProcessInstanceCleanup = piCleanup
	if err != nil {
		cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.Errors = append(cleanup.Errors, err.Error())
		cleanup.Errors = append(cleanup.Errors, result.Deployment.Errors...)
		smokeTestMarkUnattemptedDefinitionCleanup(&cleanup, "process-instance cleanup failed: "+err.Error())
		cleanup.DefinitionSummary = smokeTestDefinitionCleanupSummary(cleanup.DefinitionResults, result.Deployment.DefinitionScopeComplete, cleanup.Errors)
		return cleanup, fmt.Errorf("cleanup smoke-test process instances: %w", err)
	}

	if smokeTestHasDefinitionMutationTarget(cleanup.DefinitionResults) && (s.piAPI == nil || s.pdAPI == nil || s.resourceAPI == nil) {
		err := fmt.Errorf("%w: smoke-test process-definition cleanup requires process-instance, process-definition, and resource services", d.ErrValidation)
		cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusFailed
		cleanup.ProcessDefinitionEligibility.Errors = []string{err.Error()}
		cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.Errors = append(cleanup.Errors, err.Error())
		cleanup.Errors = append(cleanup.Errors, result.Deployment.Errors...)
		smokeTestMarkUnattemptedDefinitionCleanup(&cleanup, err.Error())
		cleanup.DefinitionSummary = smokeTestDefinitionCleanupSummary(cleanup.DefinitionResults, result.Deployment.DefinitionScopeComplete, cleanup.Errors)
		return cleanup, err
	}

	cfg := services.ApplyCallOptions(opts)
	var workflowErrs []error
	var unattemptedReason string
	for i := range cleanup.DefinitionResults {
		item := &cleanup.DefinitionResults[i]
		if ctxErr := ctx.Err(); ctxErr != nil {
			item.Reason = ctxErr.Error()
			unattemptedReason = "cleanup stopped: " + ctxErr.Error()
			workflowErrs = append(workflowErrs, ctxErr)
			break
		}
		if item.Definition.ProcessDefinitionKey == "" {
			item.Disposition = d.SmokeTestDefinitionCleanupDispositionFailed
			item.Reason = "missing process-definition key"
			item.Eligibility.Status = d.OpsWorkflowStepStatusFailed
			item.Eligibility.Errors = []string{item.Reason}
			workflowErrs = append(workflowErrs, errors.New(item.Reason))
			if cfg.FailFast {
				unattemptedReason = "cleanup stopped after process-definition identity failure (--fail-fast)"
				break
			}
			continue
		}
		deployment := smokeTestDefinitionProjection(item.Definition)
		eligibility, eligibilityErr := smokeTestProcessDefinitionCleanupEligibility(ctx, s.piAPI, deployment, cleanupScope, opts...)
		item.Eligibility = eligibility
		if eligibilityErr != nil {
			item.Disposition = d.SmokeTestDefinitionCleanupDispositionFailed
			item.Reason = eligibilityErr.Error()
			workflowErrs = append(workflowErrs, fmt.Errorf("process definition %s eligibility: %w", item.Definition.ProcessDefinitionKey, eligibilityErr))
			if cfg.FailFast {
				unattemptedReason = "cleanup stopped after process-definition eligibility failure (--fail-fast)"
				break
			}
			continue
		}
		if !eligibility.Eligible {
			item.Disposition = d.SmokeTestDefinitionCleanupDispositionRetained
			item.Reason = smokeTestDefinitionRetentionReason(eligibility)
			continue
		}
		definitionCleanup, cleanupErr := smokeTestCleanupProcessDefinition(ctx, s.resourceAPI, s.pdAPI, s.piAPI, s.log, item.Definition.ProcessDefinitionKey, result.Request.Workers, opts...)
		item.Cleanup = definitionCleanup
		switch {
		case cleanupErr != nil:
			item.Disposition = d.SmokeTestDefinitionCleanupDispositionFailed
			item.Reason = smokeTestDefinitionCleanupFailureReason(definitionCleanup, cleanupErr)
			workflowErrs = append(workflowErrs, fmt.Errorf("process definition %s cleanup: %w", item.Definition.ProcessDefinitionKey, cleanupErr))
		case definitionCleanup.Confirmed:
			item.Disposition = d.SmokeTestDefinitionCleanupDispositionConfirmed
		case definitionCleanup.Submitted:
			item.Disposition = d.SmokeTestDefinitionCleanupDispositionSubmitted
		default:
			item.Disposition = d.SmokeTestDefinitionCleanupDispositionFailed
			item.Reason = smokeTestDefinitionCleanupFailureReason(definitionCleanup, nil)
			workflowErrs = append(workflowErrs, errors.New(item.Reason))
		}
		if cleanupErr != nil && cfg.FailFast {
			unattemptedReason = "cleanup stopped after process-definition deletion failure (--fail-fast)"
			break
		}
	}
	if unattemptedReason != "" {
		smokeTestMarkUnattemptedDefinitionCleanup(&cleanup, unattemptedReason)
	}

	for _, scopeErr := range result.Deployment.Errors {
		workflowErrs = append(workflowErrs, errors.New(scopeErr))
	}
	workflowErr := errors.Join(workflowErrs...)
	if workflowErr != nil {
		cleanup.Errors = append(cleanup.Errors, workflowErr.Error())
	}
	smokeTestApplyMainDefinitionProjection(&cleanup, result.Deployment)
	cleanup.DefinitionSummary = smokeTestDefinitionCleanupSummary(cleanup.DefinitionResults, result.Deployment.DefinitionScopeComplete, cleanup.Errors)
	if workflowErr != nil {
		return cleanup, fmt.Errorf("cleanup smoke-test process definitions: %w", workflowErr)
	}
	return cleanup, nil
}

// smokeTestMarkUnattemptedDefinitionCleanup makes every known but unscheduled definition actionable.
func smokeTestMarkUnattemptedDefinitionCleanup(cleanup *d.SmokeTestCleanupResult, reason string) {
	for i := range cleanup.DefinitionResults {
		item := &cleanup.DefinitionResults[i]
		if item.Disposition == d.SmokeTestDefinitionCleanupDispositionUnattempted && item.Reason == "" {
			item.Reason = reason
		}
	}
}

// smokeTestDefinitionCleanupFailureReason preserves rejection evidence when no helper error was returned.
func smokeTestDefinitionCleanupFailureReason(cleanup d.SmokeTestProcessDefinitionCleanupResult, err error) string {
	if err != nil {
		return err.Error()
	}
	for _, item := range cleanup.Items {
		if item.Ok {
			continue
		}
		if item.Status != "" {
			return "cleanup rejected: " + item.Status
		}
	}
	return "cleanup returned no accepted deletion response"
}

// smokeTestInitialDefinitionResults preserves deployment order before any cleanup can return early.
func smokeTestInitialDefinitionResults(definitions []d.SmokeTestDeployedDefinition) []d.SmokeTestDefinitionCleanupItem {
	items := make([]d.SmokeTestDefinitionCleanupItem, len(definitions))
	for i, definition := range definitions {
		items[i] = d.SmokeTestDefinitionCleanupItem{
			Definition:  definition,
			Disposition: d.SmokeTestDefinitionCleanupDispositionUnattempted,
			Eligibility: d.SmokeTestCleanupEligibility{Status: d.OpsWorkflowStepStatusSkipped},
			Cleanup:     d.SmokeTestProcessDefinitionCleanupResult{Status: d.OpsWorkflowStepStatusSkipped},
		}
	}
	return items
}

// smokeTestHasDefinitionMutationTarget reports whether deployment evidence includes an exact deletable key.
func smokeTestHasDefinitionMutationTarget(items []d.SmokeTestDefinitionCleanupItem) bool {
	for _, item := range items {
		if item.Definition.ProcessDefinitionKey != "" {
			return true
		}
	}
	return false
}

// smokeTestDefinitionProjection adapts one collection entry to the existing exact-key eligibility helper.
func smokeTestDefinitionProjection(definition d.SmokeTestDeployedDefinition) d.SmokeTestDeploymentResult {
	return d.SmokeTestDeploymentResult{
		BpmnProcessID:            definition.BpmnProcessID,
		ProcessDefinitionKey:     definition.ProcessDefinitionKey,
		ProcessDefinitionVersion: definition.ProcessDefinitionVersion,
		TenantID:                 definition.TenantID,
	}
}

// smokeTestDefinitionRetentionReason supplies compact evidence for safety-based retention.
func smokeTestDefinitionRetentionReason(eligibility d.SmokeTestCleanupEligibility) string {
	if len(eligibility.Blockers) == 0 {
		return "cleanup safety check did not authorize deletion"
	}
	return "unrelated process instances: " + strings.Join(eligibility.Blockers, ", ")
}

// smokeTestApplyMainDefinitionProjection keeps legacy singular fields tied to the exact runnable fixture.
func smokeTestApplyMainDefinitionProjection(cleanup *d.SmokeTestCleanupResult, deployment d.SmokeTestDeploymentResult) {
	for _, item := range cleanup.DefinitionResults {
		if item.Definition.BpmnProcessID != deployment.BpmnProcessID {
			continue
		}
		cleanup.ProcessDefinitionEligibility = item.Eligibility
		cleanup.ProcessDefinitionCleanup = item.Cleanup
		if item.Disposition == d.SmokeTestDefinitionCleanupDispositionRetained {
			cleanup.RetainedProcessDefinitionKey = item.Definition.ProcessDefinitionKey
			cleanup.RetainedBpmnProcessID = item.Definition.BpmnProcessID
			cleanup.RetainedTenantID = item.Definition.TenantID
			if cleanup.RetainedTenantID == "" {
				cleanup.RetainedTenantID = deployment.TenantID
			}
		}
		return
	}
}

// smokeTestDefinitionCleanupSummary derives exclusive counts and aggregate precedence from collection facts.
func smokeTestDefinitionCleanupSummary(items []d.SmokeTestDefinitionCleanupItem, scopeComplete bool, cleanupErrors []string) *d.SmokeTestDefinitionCleanupSummary {
	summary := &d.SmokeTestDefinitionCleanupSummary{Total: len(items)}
	for _, item := range items {
		switch item.Disposition {
		case d.SmokeTestDefinitionCleanupDispositionConfirmed:
			summary.Confirmed++
		case d.SmokeTestDefinitionCleanupDispositionSubmitted:
			summary.Submitted++
		case d.SmokeTestDefinitionCleanupDispositionRetained:
			summary.Retained++
		case d.SmokeTestDefinitionCleanupDispositionFailed:
			summary.Failed++
		default:
			summary.Unattempted++
		}
	}
	switch {
	case len(cleanupErrors) > 0 || !scopeComplete || summary.Failed > 0 || summary.Unattempted > 0 || summary.Total == 0:
		summary.Status = d.OpsWorkflowStepStatusFailed
	case summary.Retained > 0:
		summary.Status = d.OpsWorkflowStepStatusSkipped
	case summary.Submitted > 0:
		summary.Status = d.OpsWorkflowStepStatusSubmitted
	default:
		summary.Status = d.OpsWorkflowStepStatusConfirmed
	}
	return summary
}
