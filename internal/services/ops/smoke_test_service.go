// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/embedded"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

const smokeTestReportSchemaVersion = "ops.smoke-test.v1"

// ExecuteSmokeTest validates the shared smoke-test request shape and returns the foundational workflow envelope.
func (s *Service) ExecuteSmokeTest(ctx context.Context, request d.SmokeTestRequest, opts ...services.CallOption) (d.SmokeTestResult, error) {
	started := request.StartedAt
	if started.IsZero() {
		started = time.Now().UTC()
		request.StartedAt = started
	}
	request = withSmokeTestOptionControls(request, opts...)
	result := newSmokeTestResult(request)

	if err := validateSmokeTestRequest(request); err != nil {
		result.Plan.Status = d.OpsWorkflowStepStatusFailed
		result.Plan.Errors = []string{err.Error()}
		result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, err)
	}

	if request.DryRun {
		return s.executeSmokeTestDryRun(ctx, result, opts...)
	}

	return s.executeSmokeTestDeployment(ctx, result, opts...)
}

func (s *Service) executeSmokeTestDeployment(ctx context.Context, result d.SmokeTestResult, opts ...services.CallOption) (d.SmokeTestResult, error) {
	version := s.version
	if version == "" {
		version = toolx.CurrentCamundaVersion
	}
	fixture, err := smokeTestFixtureForVersion(version)
	if err != nil {
		result.Plan.Status = d.OpsWorkflowStepStatusFailed
		result.Plan.CamundaVersion = version.String()
		result.Plan.Errors = []string{err.Error()}
		result.Plan.PlannedSteps = smokeTestPlannedStepsWithStatuses(result.Request, d.OpsWorkflowStepStatusSkipped, "", nil, d.OpsWorkflowStepStatusFailed, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped)
		result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, err)
	}

	result.Fixture = fixture
	connectivityStatus := d.OpsWorkflowStepStatusSkipped
	connectivityMessage := "cluster topology check not configured"
	if s.clusterAPI != nil {
		if _, err := s.clusterAPI.GetClusterTopology(ctx, opts...); err != nil {
			result.Plan.Status = d.OpsWorkflowStepStatusFailed
			result.Plan.CamundaVersion = version.String()
			result.Plan.Fixture = fixture
			result.Plan.PlannedSteps = smokeTestPlannedStepsWithStatuses(result.Request, d.OpsWorkflowStepStatusFailed, err.Error(), []string{err.Error()}, d.OpsWorkflowStepStatusPlanned, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped)
			result.Plan.Errors = []string{err.Error()}
			result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
			result.Run.Status = d.OpsWorkflowStepStatusSkipped
			result.Walk.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
			return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, fmt.Errorf("smoke-test connectivity validation: %w", err))
		}
		connectivityStatus = d.OpsWorkflowStepStatusConfirmed
		connectivityMessage = "cluster topology retrieved"
	}

	result.Plan = d.SmokeTestPlan{
		Status:           d.OpsWorkflowStepStatusPlanned,
		CamundaVersion:   version.String(),
		Fixture:          fixture,
		CleanupRequested: !result.Request.NoCleanup,
		PlannedSteps:     smokeTestPlannedSteps(result.Request, connectivityStatus, connectivityMessage, nil),
	}
	if s.resourceAPI == nil {
		err := fmt.Errorf("%w: smoke-test deployment requires resource service", d.ErrValidation)
		result.Plan.Status = d.OpsWorkflowStepStatusFailed
		result.Plan.Errors = []string{err.Error()}
		result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, err)
	}

	unit, err := smokeTestDeploymentUnit(fixture)
	if err != nil {
		result.Plan.Status = d.OpsWorkflowStepStatusFailed
		result.Plan.Errors = []string{err.Error()}
		result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, err)
	}

	proofOpts := smokeTestProofOptions(opts...)
	deployment, err := s.resourceAPI.Deploy(ctx, []d.DeploymentUnitData{unit}, proofOpts...)
	if err != nil {
		result.Deployment = d.SmokeTestDeploymentResult{
			Status:        d.OpsWorkflowStepStatusFailed,
			FixtureFile:   fixture.File,
			BpmnProcessID: fixture.BpmnProcessID,
			Errors:        []string{err.Error()},
		}
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Plan.PlannedSteps = smokeTestPlannedStepsWithStatuses(result.Request, connectivityStatus, connectivityMessage, nil, d.OpsWorkflowStepStatusPlanned, d.OpsWorkflowStepStatusFailed, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped)
		return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, fmt.Errorf("deploy smoke-test fixture: %w", err))
	}

	result.Deployment = smokeTestDeploymentResult(fixture, deployment, d.OpsWorkflowStepStatusConfirmed)
	if s.piAPI == nil {
		err := fmt.Errorf("%w: smoke-test run requires process-instance service", d.ErrValidation)
		result.Plan.PlannedSteps = smokeTestPlannedStepsWithStatuses(result.Request, connectivityStatus, connectivityMessage, nil, d.OpsWorkflowStepStatusConfirmed, result.Deployment.Status, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped)
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Run.RequestedCount = result.Request.Count
		result.Run.Errors = []string{err.Error()}
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomePartiallyFailed, err)
	}

	result.Run, err = smokeTestCreateProcessInstances(ctx, s.piAPI, s.log, result.Request, result.Deployment, proofOpts...)
	if err != nil {
		result.Plan.PlannedSteps = smokeTestPlannedStepsWithStatuses(result.Request, connectivityStatus, connectivityMessage, nil, d.OpsWorkflowStepStatusConfirmed, result.Deployment.Status, result.Run.Status, d.OpsWorkflowStepStatusSkipped, d.OpsWorkflowStepStatusSkipped)
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomePartiallyFailed, fmt.Errorf("start smoke-test process instances: %w", err))
	}

	result.Walk, err = smokeTestWalkCreatedFamilies(ctx, s.piAPI, result.Run.ProcessInstanceKeys, result.Request.Workers, proofOpts...)
	if err != nil {
		result.Plan.PlannedSteps = smokeTestPlannedStepsWithStatuses(result.Request, connectivityStatus, connectivityMessage, nil, d.OpsWorkflowStepStatusConfirmed, result.Deployment.Status, result.Run.Status, result.Walk.Status, d.OpsWorkflowStepStatusSkipped)
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomePartiallyFailed, fmt.Errorf("walk smoke-test process-instance families: %w", err))
	}

	cleanupOpts := smokeTestCleanupOptions(opts...)
	result.Cleanup, err = s.cleanupSmokeTestResources(ctx, result, cleanupOpts...)
	result.Plan.PlannedSteps = smokeTestPlannedStepsWithStatuses(result.Request, connectivityStatus, connectivityMessage, nil, d.OpsWorkflowStepStatusConfirmed, result.Deployment.Status, result.Run.Status, result.Walk.Status, smokeTestCleanupStepStatus(result.Cleanup))
	if err != nil {
		return finishSmokeTestResult(result, d.SmokeTestOutcomePartiallyFailed, err)
	}
	if result.Request.NoCleanup {
		return finishSmokeTestResult(result, d.SmokeTestOutcomePassedCleanupSkipped, nil)
	}
	return finishSmokeTestResult(result, d.SmokeTestOutcomePassed, nil)
}

func smokeTestCreateProcessInstances(ctx context.Context, api pisvc.API, log *slog.Logger, request d.SmokeTestRequest, deployment d.SmokeTestDeploymentResult, opts ...services.CallOption) (d.SmokeTestRunResult, error) {
	out := d.SmokeTestRunResult{
		Status:         d.OpsWorkflowStepStatusSubmitted,
		RequestedCount: request.Count,
	}
	data := smokeTestProcessInstanceData(deployment)
	created, err := pisvc.CreateNProcessInstances(ctx, api, log, data, request.Count, request.Workers, opts...)
	out.Items = make([]d.SmokeTestRunItem, 0, len(created))
	for _, item := range created {
		if item.Key == "" {
			continue
		}
		out.ProcessInstanceKeys = append(out.ProcessInstanceKeys, item.Key)
		out.Items = append(out.Items, d.SmokeTestRunItem{
			ProcessInstanceKey: item.Key,
			Status:             d.OpsWorkflowStepStatusConfirmed,
		})
	}
	out.CreatedCount = len(out.ProcessInstanceKeys)
	if err != nil {
		out.Status = d.OpsWorkflowStepStatusFailed
		out.Errors = []string{err.Error()}
		return out, err
	}
	out.Status = d.OpsWorkflowStepStatusConfirmed
	return out, nil
}

func smokeTestProcessInstanceData(deployment d.SmokeTestDeploymentResult) d.ProcessInstanceData {
	data := d.ProcessInstanceData{
		TenantId: deployment.TenantID,
	}
	if strings.TrimSpace(deployment.ProcessDefinitionKey) != "" {
		data.ProcessDefinitionSpecificId = deployment.ProcessDefinitionKey
		return data
	}
	data.BpmnProcessId = deployment.BpmnProcessID
	return data
}

func smokeTestWalkCreatedFamilies(ctx context.Context, api pisvc.API, keys []string, wantedWorkers int, opts ...services.CallOption) (d.SmokeTestWalkResult, error) {
	out := d.SmokeTestWalkResult{Status: d.OpsWorkflowStepStatusSubmitted}
	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(keys), wantedWorkers, cfg.NoWorkerLimit)
	items, err := pool.ExecuteSlice[string, d.SmokeTestWalkItem](ctx, keys, workers, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.SmokeTestWalkItem, error) {
		result, walkErr := api.FamilyResult(ctx, key, opts...)
		item := smokeTestWalkItemFromTraversal(key, result)
		if walkErr != nil {
			item.Status = d.OpsWorkflowStepStatusFailed
			item.Error = walkErr.Error()
			return item, walkErr
		}
		item.Status = d.OpsWorkflowStepStatusConfirmed
		return item, nil
	})
	out.Items = items
	if err != nil {
		out.Status = d.OpsWorkflowStepStatusFailed
		out.Errors = []string{err.Error()}
		return out, err
	}
	out.Status = d.OpsWorkflowStepStatusConfirmed
	return out, nil
}

func smokeTestWalkItemFromTraversal(key string, result pitraversal.Result) d.SmokeTestWalkItem {
	return d.SmokeTestWalkItem{
		ProcessInstanceKey: key,
		Summary: d.SmokeTestTraversalSummary{
			ProcessInstanceKey:     key,
			RootProcessInstanceKey: result.RootKey,
			FamilyKeys:             append(typex.Keys(nil), result.Keys...),
			MissingAncestors:       smokeTestMissingAncestors(result.MissingAncestors),
			Warning:                result.Warning,
			Outcome:                d.TraversalOutcome(result.Outcome),
		},
	}
}

func smokeTestMissingAncestors(items []pitraversal.MissingAncestor) []d.MissingAncestor {
	out := make([]d.MissingAncestor, 0, len(items))
	for _, item := range items {
		out = append(out, d.MissingAncestor{
			Key:      item.Key,
			StartKey: item.StartKey,
		})
	}
	return out
}

func smokeTestProofOptions(opts ...services.CallOption) []services.CallOption {
	cfg := services.ApplyCallOptions(opts)
	out := make([]services.CallOption, 0, 5)
	if cfg.FailFast {
		out = append(out, services.WithFailFast())
	}
	if cfg.NoWorkerLimit {
		out = append(out, services.WithNoWorkerLimit())
	}
	if cfg.Verbose {
		out = append(out, services.WithVerbose())
	}
	if !cfg.Verbose {
		out = append(out, services.WithSuppressWorkflowDetailLogs(), services.WithSuppressProcessInstanceDetailLogs())
	}
	return out
}

func smokeTestCleanupOptions(opts ...services.CallOption) []services.CallOption {
	cfg := services.ApplyCallOptions(opts)
	out := make([]services.CallOption, 0, 6)
	if cfg.FailFast {
		out = append(out, services.WithFailFast())
	}
	if cfg.NoWorkerLimit {
		out = append(out, services.WithNoWorkerLimit())
	}
	if cfg.Verbose {
		out = append(out, services.WithVerbose())
	}
	if cfg.NoWait {
		out = append(out, services.WithNoWait())
	}
	return append(out, services.WithForce(), services.WithSuppressProcessInstanceDetailLogs())
}

func (s *Service) cleanupSmokeTestResources(ctx context.Context, result d.SmokeTestResult, opts ...services.CallOption) (d.SmokeTestCleanupResult, error) {
	cleanup := d.SmokeTestCleanupResult{NoCleanup: result.Request.NoCleanup}
	if result.Request.NoCleanup {
		cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.RetainedProcessInstanceKeys = append(typex.Keys(nil), result.Run.ProcessInstanceKeys...)
		cleanup.RetainedProcessDefinitionKey = result.Deployment.ProcessDefinitionKey
		cleanup.RetainedBpmnProcessID = result.Deployment.BpmnProcessID
		cleanup.RetainedTenantID = result.Deployment.TenantID
		return cleanup, nil
	}
	piCleanup, cleanupScope, err := smokeTestCleanupProcessInstances(ctx, s.piAPI, s.log, result.Run.ProcessInstanceKeys, result.Request.Workers, opts...)
	cleanup.ProcessInstanceCleanup = piCleanup
	if err != nil {
		cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.Errors = []string{err.Error()}
		return cleanup, fmt.Errorf("cleanup smoke-test process instances: %w", err)
	}
	if result.Deployment.ProcessDefinitionKey != "" && (s.pdAPI == nil || s.resourceAPI == nil) {
		err := fmt.Errorf("%w: smoke-test process-definition cleanup requires process-definition and resource services", d.ErrValidation)
		cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusFailed
		cleanup.ProcessDefinitionEligibility.Errors = []string{err.Error()}
		cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.Errors = []string{err.Error()}
		return cleanup, err
	}

	eligibility, err := smokeTestProcessDefinitionCleanupEligibility(ctx, s.piAPI, result.Deployment, cleanupScope, opts...)
	cleanup.ProcessDefinitionEligibility = eligibility
	if err != nil {
		cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		cleanup.Errors = []string{err.Error()}
		return cleanup, err
	}
	pdCleanup, err := smokeTestCleanupProcessDefinition(ctx, s.resourceAPI, s.pdAPI, s.piAPI, s.log, result.Deployment.ProcessDefinitionKey, result.Request.Workers, opts...)
	cleanup.ProcessDefinitionCleanup = pdCleanup
	if err != nil {
		cleanup.Errors = []string{err.Error()}
		return cleanup, fmt.Errorf("cleanup smoke-test process definition: %w", err)
	}
	return cleanup, nil
}

func smokeTestCleanupProcessInstances(ctx context.Context, api pisvc.API, log *slog.Logger, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.SmokeTestProcessInstanceCleanupResult, typex.Keys, error) {
	out := d.SmokeTestProcessInstanceCleanupResult{Status: d.OpsWorkflowStepStatusSkipped, NoWait: services.ApplyCallOptions(opts).NoWait}
	if len(keys) == 0 {
		return out, nil, nil
	}
	plan, err := pisvc.DryRunCancelOrDeletePlan(ctx, api, keys, wantedWorkers, opts...)
	if err != nil {
		out.Status = d.OpsWorkflowStepStatusFailed
		out.Errors = []string{err.Error()}
		return out, nil, err
	}
	roots := plan.Roots.Unique()
	affected := len(plan.Collected.Unique())
	if affected == 0 {
		affected = len(roots)
	}
	deleteOpts := append([]services.CallOption{}, opts...)
	deleteOpts = append(deleteOpts, services.WithAffectedProcessInstanceCount(affected))
	reports, err := pisvc.DeleteProcessInstances(ctx, api, log, roots, wantedWorkers, affected, deleteOpts...)
	out.Submitted = len(reports) > 0
	out.SubmittedKeys = append(typex.Keys(nil), roots...)
	out.Items = append([]d.Reporter(nil), reports...)
	out.Status = deletionStatusForReports(reports, out.NoWait, err)
	out.Confirmed = out.Status == d.OpsWorkflowStepStatusConfirmed
	if err != nil {
		out.Errors = []string{err.Error()}
		return out, plan.Collected.Unique(), err
	}
	return out, plan.Collected.Unique(), nil
}

func smokeTestProcessDefinitionCleanupEligibility(ctx context.Context, api pisvc.API, deployment d.SmokeTestDeploymentResult, ownedKeys typex.Keys, opts ...services.CallOption) (d.SmokeTestCleanupEligibility, error) {
	out := d.SmokeTestCleanupEligibility{Status: d.OpsWorkflowStepStatusSkipped}
	if deployment.ProcessDefinitionKey == "" {
		return out, nil
	}
	unrelated, err := pdsvc.FindUnrelatedProcessInstancesForDefinition(ctx, api, deployment.ProcessDefinitionKey, deployment.BpmnProcessID, ownedKeys, opts...)
	if err != nil {
		out.Status = d.OpsWorkflowStepStatusFailed
		out.Errors = []string{err.Error()}
		return out, err
	}
	if len(unrelated) > 0 {
		out.Status = d.OpsWorkflowStepStatusBlocked
		out.Blockers = smokeTestCleanupBlockers(unrelated)
		err := fmt.Errorf("%w: process-definition cleanup blocked by unrelated process instance(s): %s", d.ErrPrecondition, strings.Join(out.Blockers, ", "))
		out.Errors = []string{err.Error()}
		return out, err
	}
	out.Status = d.OpsWorkflowStepStatusConfirmed
	out.Eligible = true
	return out, nil
}

func smokeTestCleanupBlockers(items []d.ProcessInstance) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.Key != "" {
			out = append(out, item.Key)
		}
	}
	return out
}

func smokeTestCleanupProcessDefinition(ctx context.Context, resourceAPI pdsvc.ResourceDeleteAPI, pdAPI pdsvc.API, piAPI pisvc.API, log *slog.Logger, key string, wantedWorkers int, opts ...services.CallOption) (d.SmokeTestProcessDefinitionCleanupResult, error) {
	out := d.SmokeTestProcessDefinitionCleanupResult{Status: d.OpsWorkflowStepStatusSkipped, NoWait: services.ApplyCallOptions(opts).NoWait}
	if key == "" {
		return out, nil
	}
	responses, err := pdsvc.DeleteProcessDefinitions(ctx, resourceAPI, pdAPI, piAPI, log, typex.Keys{key}, wantedWorkers, opts...)
	out.Submitted = len(responses) > 0
	out.SubmittedProcessDefinitionKey = key
	out.Items = append([]d.ResourceDeleteResponse(nil), responses...)
	out.Status = smokeTestProcessDefinitionCleanupStatus(responses, out.NoWait, err)
	out.Confirmed = out.Status == d.OpsWorkflowStepStatusConfirmed
	if err != nil {
		out.Errors = []string{err.Error()}
		return out, err
	}
	return out, nil
}

func smokeTestProcessDefinitionCleanupStatus(items []d.ResourceDeleteResponse, noWait bool, err error) d.OpsWorkflowStepStatus {
	if err != nil || !allResourceDeleteResponsesOK(items) {
		return d.OpsWorkflowStepStatusFailed
	}
	if noWait {
		return d.OpsWorkflowStepStatusSubmitted
	}
	return d.OpsWorkflowStepStatusConfirmed
}

func smokeTestCleanupStepStatus(cleanup d.SmokeTestCleanupResult) d.OpsWorkflowStepStatus {
	for _, status := range []d.OpsWorkflowStepStatus{
		cleanup.ProcessInstanceCleanup.Status,
		cleanup.ProcessDefinitionEligibility.Status,
		cleanup.ProcessDefinitionCleanup.Status,
	} {
		if status == d.OpsWorkflowStepStatusFailed || status == d.OpsWorkflowStepStatusBlocked {
			return status
		}
	}
	if cleanup.NoCleanup {
		return d.OpsWorkflowStepStatusSkipped
	}
	if cleanup.ProcessInstanceCleanup.Status == d.OpsWorkflowStepStatusSubmitted || cleanup.ProcessDefinitionCleanup.Status == d.OpsWorkflowStepStatusSubmitted {
		return d.OpsWorkflowStepStatusSubmitted
	}
	return d.OpsWorkflowStepStatusConfirmed
}

func (s *Service) executeSmokeTestDryRun(ctx context.Context, result d.SmokeTestResult, opts ...services.CallOption) (d.SmokeTestResult, error) {
	version := s.version
	if version == "" {
		version = toolx.CurrentCamundaVersion
	}
	fixture, err := smokeTestFixtureForVersion(version)
	if err != nil {
		result.Plan.Status = d.OpsWorkflowStepStatusFailed
		result.Plan.CamundaVersion = version.String()
		result.Plan.Errors = []string{err.Error()}
		result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
		result.Run.Status = d.OpsWorkflowStepStatusSkipped
		result.Walk.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
		result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
		return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, err)
	}

	connectivityStatus := d.OpsWorkflowStepStatusSkipped
	connectivityMessage := "cluster topology check not configured"
	if s.clusterAPI != nil {
		if _, err := s.clusterAPI.GetClusterTopology(ctx, opts...); err != nil {
			result.Plan.Status = d.OpsWorkflowStepStatusFailed
			result.Plan.CamundaVersion = version.String()
			result.Plan.Fixture = fixture
			result.Fixture = fixture
			result.Plan.PlannedSteps = smokeTestPlannedSteps(result.Request, d.OpsWorkflowStepStatusFailed, err.Error(), []string{err.Error()})
			result.Plan.Errors = []string{err.Error()}
			result.Deployment.Status = d.OpsWorkflowStepStatusSkipped
			result.Run.Status = d.OpsWorkflowStepStatusSkipped
			result.Walk.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessInstanceCleanup.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessDefinitionEligibility.Status = d.OpsWorkflowStepStatusSkipped
			result.Cleanup.ProcessDefinitionCleanup.Status = d.OpsWorkflowStepStatusSkipped
			return finishSmokeTestResult(result, d.SmokeTestOutcomeFailed, fmt.Errorf("smoke-test connectivity validation: %w", err))
		}
		connectivityStatus = d.OpsWorkflowStepStatusConfirmed
		connectivityMessage = "cluster topology retrieved"
	}

	result.Fixture = fixture
	result.Plan = d.SmokeTestPlan{
		Status:           d.OpsWorkflowStepStatusPlanned,
		CamundaVersion:   version.String(),
		Fixture:          fixture,
		CleanupRequested: !result.Request.NoCleanup,
		PlannedSteps:     smokeTestPlannedSteps(result.Request, connectivityStatus, connectivityMessage, nil),
	}
	result.Deployment = d.SmokeTestDeploymentResult{
		Status:        d.OpsWorkflowStepStatusPlanned,
		FixtureFile:   fixture.File,
		BpmnProcessID: fixture.BpmnProcessID,
	}
	result.Run = d.SmokeTestRunResult{
		Status:         d.OpsWorkflowStepStatusPlanned,
		RequestedCount: result.Request.Count,
	}
	result.Walk.Status = d.OpsWorkflowStepStatusPlanned
	cleanupStatus := d.OpsWorkflowStepStatusPlanned
	if result.Request.NoCleanup {
		cleanupStatus = d.OpsWorkflowStepStatusSkipped
	}
	result.Cleanup.ProcessInstanceCleanup.Status = cleanupStatus
	result.Cleanup.ProcessDefinitionEligibility.Status = cleanupStatus
	result.Cleanup.ProcessDefinitionCleanup.Status = cleanupStatus
	return finishSmokeTestResult(result, d.SmokeTestOutcomePlanned, nil)
}

func smokeTestFixtureForVersion(version toolx.CamundaVersion) (d.EmbeddedSmokeTestFixture, error) {
	normalized, err := toolx.NormalizeCamundaVersion(version.String())
	if err != nil {
		return d.EmbeddedSmokeTestFixture{}, fmt.Errorf("%w: unsupported smoke-test fixture version %q", d.ErrPrecondition, version)
	}
	processID := normalized.FilePrefix() + "MultipleSubProcessesParentProcess"
	fsPath := "processdefinitions/" + processID + ".bpmn"
	if _, err := fs.Stat(embedded.FS, fsPath); err != nil {
		return d.EmbeddedSmokeTestFixture{}, fmt.Errorf("%w: embedded smoke-test fixture not found: %s", d.ErrPrecondition, fsPath)
	}
	return d.EmbeddedSmokeTestFixture{
		CamundaVersion: normalized.String(),
		File:           filepath.ToSlash(filepath.Join("embedded", fsPath)),
		BpmnProcessID:  processID,
		Available:      true,
	}, nil
}

func smokeTestDeploymentUnit(fixture d.EmbeddedSmokeTestFixture) (d.DeploymentUnitData, error) {
	fsPath := strings.TrimPrefix(fixture.File, "embedded/")
	data, err := fs.ReadFile(embedded.FS, fsPath)
	if err != nil {
		return d.DeploymentUnitData{}, fmt.Errorf("%w: read embedded smoke-test fixture %s: %v", d.ErrPrecondition, fixture.File, err)
	}
	return d.DeploymentUnitData{
		Name:        fsPath,
		ContentType: "application/xml",
		Data:        data,
	}, nil
}

func smokeTestDeploymentResult(fixture d.EmbeddedSmokeTestFixture, deployment d.Deployment, status d.OpsWorkflowStepStatus) d.SmokeTestDeploymentResult {
	out := d.SmokeTestDeploymentResult{
		Status:        status,
		FixtureFile:   fixture.File,
		BpmnProcessID: fixture.BpmnProcessID,
		TenantID:      deployment.TenantId,
	}
	if len(deployment.Units) > 0 {
		pd := deployment.Units[0].ProcessDefinition
		if pd.ProcessDefinitionId != "" {
			out.BpmnProcessID = pd.ProcessDefinitionId
		}
		out.ProcessDefinitionKey = pd.ProcessDefinitionKey
		out.ProcessDefinitionVersion = pd.ProcessDefinitionVersion
		if pd.TenantId != "" {
			out.TenantID = pd.TenantId
		}
	}
	return out
}

func smokeTestPlannedSteps(request d.SmokeTestRequest, connectivityStatus d.OpsWorkflowStepStatus, connectivityMessage string, connectivityErrors []string) []d.WorkflowStepResult {
	cleanupStatus := d.OpsWorkflowStepStatusPlanned
	if request.NoCleanup {
		cleanupStatus = d.OpsWorkflowStepStatusSkipped
	}
	return smokeTestPlannedStepsWithStatuses(request, connectivityStatus, connectivityMessage, connectivityErrors, d.OpsWorkflowStepStatusPlanned, d.OpsWorkflowStepStatusPlanned, d.OpsWorkflowStepStatusPlanned, d.OpsWorkflowStepStatusPlanned, cleanupStatus)
}

func smokeTestPlannedStepsWithStatuses(request d.SmokeTestRequest, connectivityStatus d.OpsWorkflowStepStatus, connectivityMessage string, connectivityErrors []string, fixtureStatus d.OpsWorkflowStepStatus, deploymentStatus d.OpsWorkflowStepStatus, runStatus d.OpsWorkflowStepStatus, walkStatus d.OpsWorkflowStepStatus, cleanupStatus d.OpsWorkflowStepStatus) []d.WorkflowStepResult {
	if connectivityStatus == "" {
		connectivityStatus = d.OpsWorkflowStepStatusPlanned
	}
	if fixtureStatus == "" {
		fixtureStatus = d.OpsWorkflowStepStatusPlanned
	}
	if deploymentStatus == "" {
		deploymentStatus = d.OpsWorkflowStepStatusPlanned
	}
	if runStatus == "" {
		runStatus = d.OpsWorkflowStepStatusPlanned
	}
	if walkStatus == "" {
		walkStatus = d.OpsWorkflowStepStatusPlanned
	}
	steps := []d.WorkflowStepResult{
		{Name: "connectivity", Status: connectivityStatus, Message: connectivityMessage, Errors: append([]string(nil), connectivityErrors...)},
		{Name: "fixture", Status: fixtureStatus, Message: "select version-matched embedded fixture"},
		{Name: "deployment", Status: deploymentStatus, Message: "deploy selected fixture"},
		{Name: "run", Status: runStatus, Message: fmt.Sprintf("start %d process instance(s)", request.Count)},
		{Name: "walk", Status: walkStatus, Message: "walk each created process-instance family"},
	}
	cleanupMessage := "clean up created process instances and eligible process definition"
	if request.NoCleanup {
		cleanupMessage = "retain created resources because --no-cleanup is set"
	}
	steps = append(steps, d.WorkflowStepResult{Name: "cleanup", Status: cleanupStatus, Message: cleanupMessage})
	reportStatus := d.OpsWorkflowStepStatusSkipped
	reportMessage := "no audit report requested"
	if strings.TrimSpace(request.ReportFile) != "" {
		reportStatus = d.OpsWorkflowStepStatusPlanned
		reportMessage = "write audit report"
	}
	return append(steps, d.WorkflowStepResult{Name: "report", Status: reportStatus, Message: reportMessage})
}

// validateSmokeTestRequest enforces local request invariants before any later workflow step can mutate state.
func validateSmokeTestRequest(request d.SmokeTestRequest) error {
	if request.Count < 1 {
		return fmt.Errorf("%w: count must be a positive integer", d.ErrValidation)
	}
	switch request.ReportFormat {
	case "", "markdown", "json":
	default:
		return fmt.Errorf("%w: report-format must be markdown or json", d.ErrValidation)
	}
	if request.ReportFormat != "" && request.ReportFile == "" {
		return fmt.Errorf("%w: report-format requires report-file", d.ErrValidation)
	}
	return nil
}

// withSmokeTestOptionControls folds reusable call options into the persisted smoke-test request controls.
func withSmokeTestOptionControls(request d.SmokeTestRequest, opts ...services.CallOption) d.SmokeTestRequest {
	cfg := services.ApplyCallOptions(opts)
	request.NoWait = request.NoWait || cfg.NoWait
	request.FailFast = request.FailFast || cfg.FailFast
	request.NoWorkerLimit = request.NoWorkerLimit || cfg.NoWorkerLimit
	return request
}

// newSmokeTestResult initializes the common report envelope before validation or workflow execution.
func newSmokeTestResult(request d.SmokeTestRequest) d.SmokeTestResult {
	cleanupRequested := !request.NoCleanup
	return d.SmokeTestResult{
		Request: request,
		Plan: d.SmokeTestPlan{
			CleanupRequested: cleanupRequested,
		},
		Cleanup: d.SmokeTestCleanupResult{
			NoCleanup: request.NoCleanup,
		},
		Report: d.SmokeTestAuditReport{
			SchemaVersion:    smokeTestReportSchemaVersion,
			CommandName:      request.CommandName,
			StartedAt:        request.StartedAt,
			DryRun:           request.DryRun,
			CleanupRequested: cleanupRequested,
			NoCleanup:        request.NoCleanup,
			AutoConfirm:      request.AutoConfirm,
			Automation:       request.Automation,
			NoWait:           request.NoWait,
			Outcome:          d.SmokeTestOutcomeFailed,
		},
		Outcome: d.SmokeTestOutcomeFailed,
	}
}

// finishSmokeTestResult snapshots final status and mirrors step data into the audit report.
func finishSmokeTestResult(result d.SmokeTestResult, outcome d.SmokeTestOutcome, err error) (d.SmokeTestResult, error) {
	finished := time.Now().UTC()
	result.Outcome = outcome
	result.Report.Outcome = outcome
	result.Report.FinishedAt = finished
	if !result.Request.StartedAt.IsZero() {
		result.Report.Duration = finished.Sub(result.Request.StartedAt).String()
	}
	result.Report.Plan = result.Plan
	result.Report.Fixture = result.Fixture
	result.Report.Deployment = result.Deployment
	result.Report.Run = result.Run
	result.Report.Walk = result.Walk
	result.Report.Cleanup = result.Cleanup
	if err != nil {
		result.Errors = appendIfMissing(result.Errors, err.Error())
	}
	result.Report.Errors = append([]string(nil), result.Errors...)
	return result, err
}
