// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/grafvonb/c8volt/embedded"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
)

// ExecuteAPILatencyTest performs active diagnostic preflight and returns an immutable dry-run plan.
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
	result.Outcome = d.APILatencyOutcomePartial
	result.Context.FinishedAt = apiLatencyNow()
	result.Context.Duration = result.Context.FinishedAt.Sub(started).String()
	return result, fmt.Errorf("%w: active API latency execution stages are not implemented yet", d.ErrPrecondition)
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
