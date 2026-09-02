// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
)

var apiLatencyNow = func() time.Time { return time.Now().UTC() }

// analyseAPILatencyReadOnly executes bounded read-only sample stages through existing service APIs.
func (s *Service) analyseAPILatencyReadOnly(ctx context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
	cfg := services.ApplyCallOptions(opts)
	if request.Progress == nil {
		request.Progress = cfg.Progress
	}
	request.Mode = d.APILatencyModeReadOnly
	plan, err := PlanAPILatency(request)
	result := d.APILatencyResult{
		SchemaVersion: d.APILatencySchemaVersion,
		Request:       request,
		Plan:          plan,
		Notices:       append([]string(nil), plan.Notices...),
		Limitations:   append([]string(nil), plan.Limitations...),
		Outcome:       d.APILatencyOutcomeCompleted,
	}
	if err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		return result, err
	}
	if err := s.requireAPILatencyReadOnlyDependencies(); err != nil {
		result.Outcome = d.APILatencyOutcomeFailed
		return result, err
	}

	started := request.StartedAt
	if started.IsZero() {
		started = apiLatencyNow()
	}
	result.Context = d.APILatencyRunContext{
		CommandName:    request.CommandName,
		SchemaVersion:  d.APILatencySchemaVersion,
		CamundaVersion: s.version.String(),
		StartedAt:      started,
		Profile:        "",
		Tenant:         request.TenantID,
	}

	var measurements []d.APILatencyMeasurement
	var topologyEvidence d.APILatencyTopologyEvidence
	for _, stage := range plan.Stages {
		reportAPILatencyStageProgress(request.Progress, "measuring read-only API latency", stage, 0, 0)
		cycles, stageErr := pool.ExecuteNTimes(ctx, stage.PrimarySamples, stage.WorkerCount, cfg.FailFast, func(ctx context.Context, _ int) (apiLatencyReadOnlyCycle, error) {
			return s.measureAPILatencyReadOnlyCycle(ctx, stage.Index, opts...)
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
			topologyEvidence = mergeAPILatencyTopologyEvidence(topologyEvidence, cycle.topology)
		}
		reportAPILatencyStageProgress(request.Progress, "measuring read-only API latency", stage, done, failed)
		if stageErr == nil && done < stage.PrimarySamples && ctx.Err() != nil {
			stageErr = ctx.Err()
		}
		if stageErr != nil {
			result.Stages = BuildAPILatencyStageResults(plan, measurements)
			result.Topology = topologyEvidence
			result.Findings = EvaluateAPILatencyFindings(result.Topology, result.Stages)
			result.Outcome = apiLatencyInterruptedOrPartialOutcome(ctx)
			result.Context.FinishedAt = apiLatencyNow()
			result.Context.Duration = result.Context.FinishedAt.Sub(started).String()
			return result, stageErr
		}
	}
	result.Stages = BuildAPILatencyStageResults(plan, measurements)
	result.Topology = topologyEvidence
	result.Findings = EvaluateAPILatencyFindings(result.Topology, result.Stages)
	result.Context.FinishedAt = apiLatencyNow()
	result.Context.Duration = result.Context.FinishedAt.Sub(started).String()
	return result, nil
}

type apiLatencyReadOnlyCycle struct {
	measurements []d.APILatencyMeasurement
	topology     d.APILatencyTopologyEvidence
}

// requireAPILatencyReadOnlyDependencies fails before measuring when required read-only services are absent.
func (s *Service) requireAPILatencyReadOnlyDependencies() error {
	switch {
	case s.clusterAPI == nil:
		return fmt.Errorf("%w: api latency analysis requires cluster topology service", d.ErrPrecondition)
	case s.pdAPI == nil:
		return fmt.Errorf("%w: api latency analysis requires process-definition service", d.ErrPrecondition)
	case s.piAPI == nil:
		return fmt.Errorf("%w: api latency analysis requires process-instance service", d.ErrPrecondition)
	default:
		return nil
	}
}

// measureAPILatencyReadOnlyCycle measures one topology/search sample and bounded search-derived keyed reads.
func (s *Service) measureAPILatencyReadOnlyCycle(ctx context.Context, stageIndex int, opts ...services.CallOption) (apiLatencyReadOnlyCycle, error) {
	measurements := make([]d.APILatencyMeasurement, 0, 5)
	var topology d.Topology
	var definitions []d.ProcessDefinition
	var instances []d.ProcessInstance

	if err := ctx.Err(); err != nil {
		return apiLatencyReadOnlyCycle{measurements: measurements}, err
	}
	topology, measurement := measureAPILatencyCall(ctx, stageIndex, d.APILatencyCategoryTopologyRead, d.APILatencyMeasurementKindPrimary, func(ctx context.Context) (d.Topology, error) {
		return s.clusterAPI.GetClusterTopology(ctx, opts...)
	})
	measurements = append(measurements, measurement)

	if err := ctx.Err(); err != nil {
		return apiLatencyReadOnlyCycle{measurements: measurements, topology: topologyAPILatencyEvidence(topology)}, err
	}
	definitions, measurement = measureAPILatencyCall(ctx, stageIndex, d.APILatencyCategoryProcessDefinitionSearch, d.APILatencyMeasurementKindPrimary, func(ctx context.Context) ([]d.ProcessDefinition, error) {
		return s.pdAPI.SearchProcessDefinitions(ctx, d.ProcessDefinitionFilter{}, 1, opts...)
	})
	measurements = append(measurements, measurement)

	if err := ctx.Err(); err != nil {
		return apiLatencyReadOnlyCycle{measurements: measurements, topology: topologyAPILatencyEvidence(topology)}, err
	}
	instances, measurement = measureAPILatencyCall(ctx, stageIndex, d.APILatencyCategoryProcessInstanceSearch, d.APILatencyMeasurementKindPrimary, func(ctx context.Context) ([]d.ProcessInstance, error) {
		return s.piAPI.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{}, 1, opts...)
	})
	measurements = append(measurements, measurement)

	measurements = append(measurements, s.measureAPILatencyProcessDefinitionRead(ctx, stageIndex, firstAPILatencyProcessDefinitionKey(definitions), opts...))
	measurements = append(measurements, s.measureAPILatencyProcessInstanceRead(ctx, stageIndex, firstAPILatencyProcessInstanceKey(instances), opts...))
	return apiLatencyReadOnlyCycle{measurements: measurements, topology: topologyAPILatencyEvidence(topology)}, ctx.Err()
}

// measureAPILatencyProcessDefinitionRead records unavailable evidence when search yields no reusable definition key.
func (s *Service) measureAPILatencyProcessDefinitionRead(ctx context.Context, stageIndex int, key string, opts ...services.CallOption) d.APILatencyMeasurement {
	if key == "" {
		return newAPILatencyUnavailableMeasurement(stageIndex, d.APILatencyCategoryProcessDefinitionRead, d.APILatencyClassificationUnavailable)
	}
	_, measurement := measureAPILatencyCall(ctx, stageIndex, d.APILatencyCategoryProcessDefinitionRead, d.APILatencyMeasurementKindDerived, func(ctx context.Context) (d.ProcessDefinition, error) {
		return s.pdAPI.GetProcessDefinition(ctx, key, opts...)
	})
	return measurement
}

// measureAPILatencyProcessInstanceRead records the v8.7 direct-read capability gap without calling the PI service.
func (s *Service) measureAPILatencyProcessInstanceRead(ctx context.Context, stageIndex int, key string, opts ...services.CallOption) d.APILatencyMeasurement {
	if key == "" {
		return newAPILatencyUnavailableMeasurement(stageIndex, d.APILatencyCategoryProcessInstanceRead, d.APILatencyClassificationUnavailable)
	}
	if s.version == toolx.V87 {
		return newAPILatencyUnavailableMeasurement(stageIndex, d.APILatencyCategoryProcessInstanceRead, d.APILatencyClassificationUnsupported)
	}
	_, measurement := measureAPILatencyCall(ctx, stageIndex, d.APILatencyCategoryProcessInstanceRead, d.APILatencyMeasurementKindDerived, func(ctx context.Context) (d.ProcessInstance, error) {
		return s.piAPI.GetProcessInstance(ctx, key, opts...)
	})
	return measurement
}

// measureAPILatencyCall wraps one logical service call with timing and sanitized classification.
func measureAPILatencyCall[T any](ctx context.Context, stageIndex int, category d.APILatencyMeasurementCategory, kind d.APILatencyMeasurementKind, call func(context.Context) (T, error)) (T, d.APILatencyMeasurement) {
	started := apiLatencyNow()
	value, err := call(ctx)
	duration := apiLatencyNow().Sub(started)
	classification := ClassifyAPILatencyError(err)
	return value, d.APILatencyMeasurement{
		StageIndex:     stageIndex,
		Category:       category,
		Kind:           kind,
		StartedAt:      started,
		Duration:       duration,
		Outcome:        apiLatencyMeasurementOutcome(classification),
		Classification: classification,
	}
}

// newAPILatencyUnavailableMeasurement records missing or unsupported derived evidence as a bounded result.
func newAPILatencyUnavailableMeasurement(stageIndex int, category d.APILatencyMeasurementCategory, classification d.APILatencyClassification) d.APILatencyMeasurement {
	return d.APILatencyMeasurement{
		StageIndex:     stageIndex,
		Category:       category,
		Kind:           d.APILatencyMeasurementKindDerived,
		StartedAt:      apiLatencyNow(),
		Outcome:        d.APILatencyMeasurementUnavailable,
		Classification: classification,
	}
}

// apiLatencyMeasurementOutcome maps safe classes onto measurement outcomes without exposing raw errors.
func apiLatencyMeasurementOutcome(classification d.APILatencyClassification) d.APILatencyMeasurementOutcome {
	switch classification {
	case d.APILatencyClassificationSuccess:
		return d.APILatencyMeasurementSucceeded
	case d.APILatencyClassificationTimeout:
		return d.APILatencyMeasurementTimedOut
	case d.APILatencyClassificationUnavailable, d.APILatencyClassificationUnsupported, d.APILatencyClassificationNotFound:
		return d.APILatencyMeasurementUnavailable
	default:
		return d.APILatencyMeasurementFailed
	}
}

// firstAPILatencyProcessDefinitionKey selects the measured search key used for derived direct reads.
func firstAPILatencyProcessDefinitionKey(items []d.ProcessDefinition) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].Key
}

// firstAPILatencyProcessInstanceKey selects the measured search key used for derived direct reads.
func firstAPILatencyProcessInstanceKey(items []d.ProcessInstance) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].Key
}

// apiLatencyInterruptedOrPartialOutcome distinguishes caller cancellation from other incomplete read-only stages.
func apiLatencyInterruptedOrPartialOutcome(ctx context.Context) d.APILatencyOutcome {
	if ctx.Err() != nil {
		return d.APILatencyOutcomeInterrupted
	}
	return d.APILatencyOutcomePartial
}

// reportAPILatencyStageProgress emits aggregate stage progress without per-key details.
func reportAPILatencyStageProgress(progress func(d.OpsProgressEvent), phase string, stage d.APILatencyStagePlan, done int, failed int) {
	if progress == nil {
		return
	}
	progress(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindFrozenScope,
		FrozenScope: &d.OpsFrozenScopeProgress{
			Phase:        phase,
			CoreResource: fmt.Sprintf("stage %d sample cycle(s)", stage.Index),
			Done:         done,
			Total:        stage.PrimarySamples,
			Errors:       failed,
		},
	})
}

// mergeAPILatencyTopologyEvidence preserves any known topology evidence collected during the run.
func mergeAPILatencyTopologyEvidence(current d.APILatencyTopologyEvidence, next d.APILatencyTopologyEvidence) d.APILatencyTopologyEvidence {
	if !next.HealthKnown {
		return current
	}
	return next
}

// topologyAPILatencyEvidence extracts safe broker, partition, health, and leadership facts.
func topologyAPILatencyEvidence(topology d.Topology) d.APILatencyTopologyEvidence {
	out := d.APILatencyTopologyEvidence{
		BrokerCount: int(topology.ClusterSize),
	}
	if out.BrokerCount == 0 {
		out.BrokerCount = len(topology.Brokers)
	}
	partitionHealth := map[int32]d.PartitionHealth{}
	partitionHasLeader := map[int32]bool{}
	for _, broker := range topology.Brokers {
		for _, partition := range broker.Partitions {
			out.HealthKnown = true
			partitionHealth[partition.PartitionId] = partition.Health
			if strings.EqualFold(string(partition.Role), "leader") {
				partitionHasLeader[partition.PartitionId] = true
			}
		}
	}
	if topology.PartitionsCount > 0 {
		out.PartitionCount = int(topology.PartitionsCount)
	} else {
		out.PartitionCount = len(partitionHealth)
	}
	for partitionID, health := range partitionHealth {
		if string(health) != "" && !strings.EqualFold(string(health), "healthy") {
			out.UnhealthyPartitions = append(out.UnhealthyPartitions, int(partitionID))
		}
		if !partitionHasLeader[partitionID] {
			out.LeaderlessPartitions = append(out.LeaderlessPartitions, int(partitionID))
		}
	}
	sort.Ints(out.UnhealthyPartitions)
	sort.Ints(out.LeaderlessPartitions)
	return out
}
