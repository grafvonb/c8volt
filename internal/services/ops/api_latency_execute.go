// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/grafvonb/c8volt/embedded"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
)

// ExecuteAPILatencyTest performs active diagnostic preflight and bounded mutation stages.
func (s *Service) ExecuteAPILatencyTest(ctx context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
	cfg := services.ApplyCallOptions(opts)
	if request.Progress == nil {
		request.Progress = cfg.Progress
	}
	request.Mode = d.APILatencyModeActive
	started := request.StartedAt
	if started.IsZero() {
		started = apiLatencyNow()
		request.StartedAt = started
	}
	result, err := s.preflightAPILatencyActive(ctx, request, opts...)
	if err != nil {
		return result, err
	}
	if request.DryRun {
		return result, nil
	}
	return s.executeAPILatencyActiveStages(ctx, result, opts...)
}

// preflightAPILatencyActive validates active-version capabilities before any deployment or process creation.
func (s *Service) preflightAPILatencyActive(ctx context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
	version := s.version
	if version == "" {
		version = toolx.CurrentCamundaVersion
	}
	plan, planErr := PlanAPILatency(request)
	result := d.APILatencyResult{
		SchemaVersion: d.APILatencySchemaVersion,
		Request:       request,
		Plan:          plan,
		Notices:       append([]string(nil), plan.Notices...),
		Limitations:   append([]string(nil), plan.Limitations...),
		Outcome:       d.APILatencyOutcomePlanned,
	}
	if planErr != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		return result, planErr
	}
	runID, err := newAPILatencyRunID()
	if err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		return result, err
	}
	fixture, err := apiLatencyFixtureForVersion(version)
	if err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		result.Plan.RunID = runID
		return result, err
	}
	result.Plan.RunID = runID
	result.Plan.Fixture = &fixture
	result.Context = d.APILatencyRunContext{
		CommandName:    request.CommandName,
		SchemaVersion:  d.APILatencySchemaVersion,
		CamundaVersion: version.String(),
		StartedAt:      request.StartedAt,
		Tenant:         request.TenantID,
	}
	topology, err := s.preflightAPILatencyTopology(ctx, opts...)
	if err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		result.Plan.Cleanup = apiLatencyBlockedCleanupPlan(result.Plan.Cleanup, fmt.Sprintf("active API latency topology preflight failed with %s classification", ClassifyAPILatencyError(err)))
		return finishAPILatencyActivePreflight(result, err)
	}
	result.Topology = topologyAPILatencyEvidence(topology)
	if notice, err := apiLatencyObservedVersionNotice(version, topology.GatewayVersion); err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		result.Plan.Cleanup = apiLatencyBlockedCleanupPlan(result.Plan.Cleanup, err.Error())
		return finishAPILatencyActivePreflight(result, err)
	} else if notice != "" {
		result.Notices = append(result.Notices, notice)
		result.Plan.Notices = append(result.Plan.Notices, notice)
	}
	if err := applyAPILatencyActiveVersionCapability(version, &result.Plan); err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		return finishAPILatencyActivePreflight(result, err)
	}
	result.Stages = BuildAPILatencyStageResults(result.Plan, nil)
	return finishAPILatencyActivePreflight(result, nil)
}

// preflightAPILatencyTopology performs the required connectivity check for active planning.
func (s *Service) preflightAPILatencyTopology(ctx context.Context, opts ...services.CallOption) (d.Topology, error) {
	if s.clusterAPI == nil {
		return d.Topology{}, fmt.Errorf("%w: active API latency test requires cluster topology service", d.ErrPrecondition)
	}
	topology, err := s.clusterAPI.GetClusterTopology(ctx, opts...)
	if err != nil {
		return d.Topology{}, fmt.Errorf("active API latency topology preflight: %w", err)
	}
	return topology, nil
}

// apiLatencyFixtureForVersion selects the existing version-matched SimpleUserTask fixture.
func apiLatencyFixtureForVersion(version toolx.CamundaVersion) (d.APILatencyFixturePlan, error) {
	normalized, err := toolx.NormalizeCamundaVersion(version.String())
	if err != nil {
		return d.APILatencyFixturePlan{}, fmt.Errorf("%w: unsupported API latency fixture version %q", d.ErrPrecondition, version)
	}
	prefix, ok := toolx.ProductionFixturePrefix(normalized)
	if !ok {
		return d.APILatencyFixturePlan{}, fmt.Errorf("%w: unsupported API latency fixture version %q", d.ErrPrecondition, version)
	}
	processID := prefix + "SimpleUserTask"
	fsPath := "processdefinitions/" + processID + ".bpmn"
	if _, err := fs.Stat(embedded.FS, fsPath); err != nil {
		return d.APILatencyFixturePlan{}, fmt.Errorf("%w: embedded API latency fixture not found: %s", d.ErrPrecondition, fsPath)
	}
	return d.APILatencyFixturePlan{
		CamundaVersion: normalized.String(),
		File:           filepath.ToSlash(filepath.Join("embedded", fsPath)),
		BpmnProcessID:  processID,
		Available:      true,
	}, nil
}

// newAPILatencyRunID returns a 128-bit local correlation identity encoded for stable reports.
func newAPILatencyRunID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("%w: generate active API latency run id: %v", d.ErrPrecondition, err)
	}
	return hex.EncodeToString(raw[:]), nil
}

// apiLatencyObservedVersionNotice compares configured and observed compatible release lines when topology reports one.
func apiLatencyObservedVersionNotice(configured toolx.CamundaVersion, observed string) (string, error) {
	if strings.TrimSpace(observed) == "" {
		return fmt.Sprintf("configured Camunda %s; observed gateway version unavailable from topology", configured.String()), nil
	}
	observedLine, ok := apiLatencyObservedVersionLine(observed)
	if !ok {
		return "", fmt.Errorf("%w: observed gateway version %q is not a supported Camunda 8.7-8.10 line", d.ErrPrecondition, observed)
	}
	if observedLine != configured {
		return "", fmt.Errorf("%w: configured Camunda %s does not match observed gateway %s", d.ErrPrecondition, configured.String(), observed)
	}
	return fmt.Sprintf("configured Camunda %s matches observed gateway %s", configured.String(), observed), nil
}

// apiLatencyObservedVersionLine normalizes patch-level gateway versions to configured compatibility lines.
func apiLatencyObservedVersionLine(observed string) (toolx.CamundaVersion, bool) {
	value := strings.TrimPrefix(strings.TrimSpace(strings.ToLower(observed)), "v")
	for _, version := range []toolx.CamundaVersion{toolx.V810, toolx.V89, toolx.V88, toolx.V87} {
		line := version.String()
		if value == line || strings.HasPrefix(value, line+".") || strings.HasPrefix(value, line+"-") {
			return version, true
		}
	}
	return "", false
}

// applyAPILatencyActiveVersionCapability records ownership and cleanup gates that must pass before mutation.
func applyAPILatencyActiveVersionCapability(version toolx.CamundaVersion, plan *d.APILatencyPlan) error {
	if plan.Cleanup == nil {
		plan.Cleanup = &d.APILatencyCleanupPlan{}
	}
	plan.Cleanup.Supported = toolx.SupportsFullProcessDefinitionHistoryDeletion(version)
	switch {
	case version == toolx.V87:
		plan.Cleanup.BlockReason = "Camunda 8.7 cannot guarantee exact active latency ownership keys"
		return fmt.Errorf("%w: %s", d.ErrUnsupported, plan.Cleanup.BlockReason)
	case plan.Cleanup.Requested && !plan.Cleanup.Supported:
		plan.Cleanup.BlockReason = fmt.Sprintf("Camunda %s cannot completely clean up active latency process-definition history; rerun with --no-cleanup only if intentional retention is acceptable", version.String())
		return fmt.Errorf("%w: %s", d.ErrUnsupported, plan.Cleanup.BlockReason)
	default:
		return nil
	}
}

// apiLatencyBlockedCleanupPlan preserves requested cleanup intent while attaching a pre-mutation block reason.
func apiLatencyBlockedCleanupPlan(plan *d.APILatencyCleanupPlan, reason string) *d.APILatencyCleanupPlan {
	if plan == nil {
		plan = &d.APILatencyCleanupPlan{}
	}
	plan.BlockReason = reason
	return plan
}

// finishAPILatencyActivePreflight stamps terminal timing and synchronizes plan notices onto the result.
func finishAPILatencyActivePreflight(result d.APILatencyResult, err error) (d.APILatencyResult, error) {
	finished := apiLatencyNow()
	result.Context.FinishedAt = finished
	if !result.Context.StartedAt.IsZero() {
		result.Context.Duration = finished.Sub(result.Context.StartedAt).String()
	}
	result.Notices = append([]string(nil), result.Plan.Notices...)
	result.Limitations = append([]string(nil), result.Plan.Limitations...)
	if err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
	}
	return result, err
}

type apiLatencyActiveCycle struct {
	measurements []d.APILatencyMeasurement
	visibility   *d.APILatencyVisibilityResult
}

// executeAPILatencyActiveStages runs the mutation-bearing active diagnostic after a successful immutable preflight.
func (s *Service) executeAPILatencyActiveStages(ctx context.Context, result d.APILatencyResult, opts ...services.CallOption) (d.APILatencyResult, error) {
	if err := s.requireAPILatencyActiveDependencies(); err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		return finishAPILatencyActiveResult(result, err)
	}
	deployment, deploymentMeasurement, err := s.deployAPILatencyFixture(ctx, result.Plan, opts...)
	measurements := []d.APILatencyMeasurement{deploymentMeasurement}
	ownership := newAPILatencyOwnership(result.Plan)
	ownership.DeploymentSubmitted = true
	ownershipRegistry := newAPILatencyOwnershipRegistry(ownership)
	result.Ownership = ownershipRegistry.snapshot()
	if err != nil {
		result.Stages = BuildAPILatencyStageResults(result.Plan, measurements)
		result.Findings = EvaluateAPILatencyFindings(result.Topology, result.Stages)
		result.Ownership = ownershipRegistry.snapshot()
		result.Outcome = d.APILatencyOutcomeFailed
		return finishAPILatencyActiveResult(result, fmt.Errorf("deploy API latency fixture: %w", err))
	}
	pdKey, err := apiLatencyDeploymentProcessDefinitionKey(deployment, result.Plan.Fixture)
	if err != nil {
		result.Stages = BuildAPILatencyStageResults(result.Plan, measurements)
		result.Findings = EvaluateAPILatencyFindings(result.Topology, result.Stages)
		result.Ownership = ownershipRegistry.snapshot()
		result.Outcome = d.APILatencyOutcomeFailed
		return finishAPILatencyActiveResult(result, err)
	}
	ownershipRegistry.registerProcessDefinitionKey(pdKey)
	result.Ownership = ownershipRegistry.snapshot()

	cfg := services.ApplyCallOptions(opts)
	var mu sync.Mutex
	var inFlightWrites int64
	recordVisibility := func(item d.APILatencyVisibilityResult) {
		mu.Lock()
		result.Visibility = append(result.Visibility, item)
		mu.Unlock()
	}

	for _, stage := range result.Plan.Stages {
		reportAPILatencyStageProgress(result.Request.Progress, "executing active API latency test", stage, 0, 0)
		cycles, stageErr := pool.ExecuteNTimes(ctx, stage.PrimarySamples, stage.WorkerCount, cfg.FailFast, func(ctx context.Context, _ int) (apiLatencyActiveCycle, error) {
			return s.measureAPILatencyActiveCycle(ctx, stage.Index, result.Request, pdKey, &inFlightWrites, ownershipRegistry.registerProcessInstanceKey, opts...)
		})
		done := 0
		failed := 0
		for _, cycle := range cycles {
			if len(cycle.measurements) == 0 {
				failed++
				continue
			}
			done++
			measurements = append(measurements, cycle.measurements...)
			if cycle.visibility != nil {
				recordVisibility(*cycle.visibility)
			}
		}
		reportAPILatencyStageProgress(result.Request.Progress, "executing active API latency test", stage, done, failed)
		if stageErr == nil && done < stage.PrimarySamples && ctx.Err() != nil {
			stageErr = ctx.Err()
		}
		if stageErr != nil {
			result.Stages = BuildAPILatencyStageResults(result.Plan, measurements)
			result.Findings = EvaluateAPILatencyFindings(result.Topology, result.Stages)
			result.Ownership = ownershipRegistry.snapshot()
			result.Outcome = apiLatencyInterruptedOrPartialOutcome(ctx)
			cleanupErr := s.finalizeAPILatencyCleanup(ctx, &result, opts...)
			if cleanupErr != nil {
				stageErr = errors.Join(stageErr, cleanupErr)
			}
			return finishAPILatencyActiveResult(result, stageErr)
		}
	}

	result.Stages = BuildAPILatencyStageResults(result.Plan, measurements)
	result.Findings = EvaluateAPILatencyFindings(result.Topology, result.Stages)
	result.Ownership = ownershipRegistry.snapshot()
	if cleanupErr := s.finalizeAPILatencyCleanup(ctx, &result, opts...); cleanupErr != nil {
		result.Outcome = d.APILatencyOutcomePartial
		return finishAPILatencyActiveResult(result, cleanupErr)
	}
	if result.Request.NoCleanup {
		result.Outcome = d.APILatencyOutcomeCompletedRetained
	} else {
		result.Outcome = d.APILatencyOutcomeCompleted
	}
	return finishAPILatencyActiveResult(result, nil)
}

// requireAPILatencyActiveDependencies fails before mutation when required active services are absent.
func (s *Service) requireAPILatencyActiveDependencies() error {
	switch {
	case s.resourceAPI == nil:
		return fmt.Errorf("%w: active API latency test requires resource service", d.ErrPrecondition)
	case s.piAPI == nil:
		return fmt.Errorf("%w: active API latency test requires process-instance service", d.ErrPrecondition)
	default:
		return nil
	}
}

// deployAPILatencyFixture submits the selected fixture without waiting for exporter visibility.
func (s *Service) deployAPILatencyFixture(ctx context.Context, plan d.APILatencyPlan, opts ...services.CallOption) (d.Deployment, d.APILatencyMeasurement, error) {
	units, err := apiLatencyDeploymentUnits(plan.Fixture)
	if err != nil {
		return d.Deployment{}, newAPILatencyUnavailableMeasurement(1, d.APILatencyCategoryFixtureDeploy, d.APILatencyClassificationUnavailable), err
	}
	deployOpts := append([]services.CallOption{}, opts...)
	deployOpts = append(deployOpts, services.WithNoWait())
	deployment, measurement := measureAPILatencyCall(ctx, 1, d.APILatencyCategoryFixtureDeploy, d.APILatencyMeasurementKindSetup, func(ctx context.Context) (d.Deployment, error) {
		return s.resourceAPI.Deploy(ctx, units, deployOpts...)
	})
	return deployment, measurement, err
}

// apiLatencyDeploymentUnits reads the exact SimpleUserTask fixture selected during active preflight.
func apiLatencyDeploymentUnits(fixture *d.APILatencyFixturePlan) ([]d.DeploymentUnitData, error) {
	if fixture == nil || strings.TrimSpace(fixture.File) == "" {
		return nil, fmt.Errorf("%w: active API latency fixture is not planned", d.ErrPrecondition)
	}
	fsPath := strings.TrimPrefix(fixture.File, "embedded/")
	data, err := fs.ReadFile(embedded.FS, fsPath)
	if err != nil {
		return nil, fmt.Errorf("%w: read active API latency fixture %s: %v", d.ErrPrecondition, fixture.File, err)
	}
	return []d.DeploymentUnitData{{
		Name:        fsPath,
		ContentType: "application/xml",
		Data:        data,
	}}, nil
}

// apiLatencyDeploymentProcessDefinitionKey extracts the exact definition key returned for the selected fixture.
func apiLatencyDeploymentProcessDefinitionKey(deployment d.Deployment, fixture *d.APILatencyFixturePlan) (string, error) {
	if fixture == nil {
		return "", fmt.Errorf("%w: active API latency fixture is not planned", d.ErrPrecondition)
	}
	var first string
	for _, unit := range deployment.Units {
		pd := unit.ProcessDefinition
		if pd.ProcessDefinitionKey == "" {
			continue
		}
		if first == "" {
			first = pd.ProcessDefinitionKey
		}
		if pd.ProcessDefinitionId == fixture.BpmnProcessID || pd.ResourceName == strings.TrimPrefix(fixture.File, "embedded/") {
			return pd.ProcessDefinitionKey, nil
		}
	}
	if first != "" {
		return first, nil
	}
	return "", fmt.Errorf("%w: active API latency deployment did not return an exact process-definition key", d.ErrPrecondition)
}

// newAPILatencyOwnership initializes the active run registry from the immutable plan.
func newAPILatencyOwnership(plan d.APILatencyPlan) *d.APILatencyOwnership {
	ownership := &d.APILatencyOwnership{RunID: plan.RunID}
	if plan.Fixture != nil {
		ownership.FixtureName = plan.Fixture.File
		ownership.BpmnProcessID = plan.Fixture.BpmnProcessID
	}
	return ownership
}

// measureAPILatencyActiveCycle creates one instance, probes reads, and polls exact-key visibility within the stage worker.
func (s *Service) measureAPILatencyActiveCycle(ctx context.Context, stageIndex int, request d.APILatencyRequest, pdKey string, inFlightWrites *int64, registerKey func(string), opts ...services.CallOption) (apiLatencyActiveCycle, error) {
	measurements := make([]d.APILatencyMeasurement, 0, 2+request.Backoff.MaxRetries+1)
	if err := ctx.Err(); err != nil {
		return apiLatencyActiveCycle{measurements: measurements}, err
	}
	data := d.ProcessInstanceData{ProcessDefinitionSpecificId: pdKey, TenantId: request.TenantID}
	createOpts := append([]services.CallOption{}, opts...)
	createOpts = append(createOpts, services.WithNoWait())
	created, createMeasurement := measureAPILatencyCall(ctx, stageIndex, d.APILatencyCategoryProcessInstanceCreate, d.APILatencyMeasurementKindPrimary, func(ctx context.Context) (d.ProcessInstanceCreation, error) {
		atomic.AddInt64(inFlightWrites, 1)
		defer atomic.AddInt64(inFlightWrites, -1)
		return s.piAPI.CreateProcessInstance(ctx, data, createOpts...)
	})
	measurements = append(measurements, createMeasurement)
	registerKey(created.Key)

	overlapped := atomic.LoadInt64(inFlightWrites) > 0
	_, readMeasurement := measureAPILatencyCall(ctx, stageIndex, d.APILatencyCategoryConcurrentRead, d.APILatencyMeasurementKindDerived, func(ctx context.Context) ([]d.ProcessInstance, error) {
		return s.piAPI.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{ProcessDefinitionKey: pdKey}, 1, opts...)
	})
	readMeasurement.OverlappedWrite = overlapped
	measurements = append(measurements, readMeasurement)

	visibility, visibilityMeasurements := s.measureAPILatencyVisibility(ctx, stageIndex, created.Key, pdKey, request.Backoff, opts...)
	measurements = append(measurements, visibilityMeasurements...)
	return apiLatencyActiveCycle{measurements: measurements, visibility: &visibility}, ctx.Err()
}

// measureAPILatencyVisibility polls exact-key search visibility within the previewed attempt ceiling.
func (s *Service) measureAPILatencyVisibility(ctx context.Context, stageIndex int, key string, pdKey string, backoff d.APILatencyBackoff, opts ...services.CallOption) (d.APILatencyVisibilityResult, []d.APILatencyMeasurement) {
	limit := APILatencyVisibilityAttemptLimit(backoff)
	result := d.APILatencyVisibilityResult{ProcessInstanceKey: key, AttemptLimit: limit}
	if key == "" {
		result.FinalClassification = d.APILatencyClassificationUnavailable
		return result, []d.APILatencyMeasurement{newAPILatencyUnavailableMeasurement(stageIndex, d.APILatencyCategorySearchVisibility, d.APILatencyClassificationUnavailable)}
	}
	started := apiLatencyNow()
	measurements := make([]d.APILatencyMeasurement, 0, limit)
	delay := backoff.InitialDelay
	for attempt := 1; attempt <= limit; attempt++ {
		items, measurement := measureAPILatencyCall(ctx, stageIndex, d.APILatencyCategorySearchVisibility, d.APILatencyMeasurementKindDerived, func(ctx context.Context) ([]d.ProcessInstance, error) {
			return s.piAPI.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{Key: key, ProcessDefinitionKey: pdKey}, 1, opts...)
		})
		if measurement.Classification == d.APILatencyClassificationSuccess && !apiLatencyProcessInstanceVisible(items, key) {
			measurement.Outcome = d.APILatencyMeasurementUnavailable
			measurement.Classification = d.APILatencyClassificationNotFound
		}
		measurements = append(measurements, measurement)
		result.Attempts = attempt
		result.FinalClassification = measurement.Classification
		if measurement.Classification == d.APILatencyClassificationSuccess {
			result.Visible = true
			result.Duration = apiLatencyNow().Sub(started)
			return result, measurements
		}
		if attempt < limit && delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				result.Duration = apiLatencyNow().Sub(started)
				return result, measurements
			case <-timer.C:
			}
			delay = apiLatencyNextBackoffDelay(backoff, delay)
		}
	}
	result.Duration = apiLatencyNow().Sub(started)
	return result, measurements
}

// apiLatencyProcessInstanceVisible requires the exact created key to appear in search results.
func apiLatencyProcessInstanceVisible(items []d.ProcessInstance, key string) bool {
	for _, item := range items {
		if item.Key == key {
			return true
		}
	}
	return false
}

// finishAPILatencyActiveResult stamps terminal timing after mutation-bearing execution.
func finishAPILatencyActiveResult(result d.APILatencyResult, err error) (d.APILatencyResult, error) {
	finished := apiLatencyNow()
	result.Context.FinishedAt = finished
	if !result.Context.StartedAt.IsZero() {
		result.Context.Duration = finished.Sub(result.Context.StartedAt).String()
	}
	return result, err
}
