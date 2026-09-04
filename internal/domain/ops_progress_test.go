// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestNewOpsETASampleWindowRequiresMinimumSamplesAndExactTotal verifies ETA is withheld until the timing sample is trustworthy.
func TestNewOpsETASampleWindowRequiresMinimumSamplesAndExactTotal(t *testing.T) {
	startedAt := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)

	subThreshold := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(500*time.Millisecond), 3, 10, OpsDefaultETAMinimumSamples)
	require.True(t, subThreshold.MinimumSamplesMet)
	require.Zero(t, subThreshold.Elapsed)
	require.Nil(t, subThreshold.Rate)
	require.Nil(t, subThreshold.Remaining)

	tooFew := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(2*time.Second), 2, 10, OpsDefaultETAMinimumSamples)
	require.False(t, tooFew.MinimumSamplesMet)
	require.NotNil(t, tooFew.Rate)
	require.Nil(t, tooFew.Remaining)

	unknownTotal := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(3*time.Second), 3, 0, OpsDefaultETAMinimumSamples)
	require.True(t, unknownTotal.MinimumSamplesMet)
	require.NotNil(t, unknownTotal.Rate)
	require.Nil(t, unknownTotal.Remaining)

	exactTotal := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(3*time.Second), 3, 12, OpsDefaultETAMinimumSamples)
	require.True(t, exactTotal.MinimumSamplesMet)
	require.Equal(t, 3*time.Second, exactTotal.Elapsed)
	require.NotNil(t, exactTotal.Rate)
	require.InDelta(t, 1.0, *exactTotal.Rate, 0.001)
	require.NotNil(t, exactTotal.Remaining)
	require.Equal(t, 9*time.Second, exactTotal.Remaining.Round(time.Second))
}

// TestNewOpsETASampleWindowOmitsRemainingWhenComplete verifies completed frozen scopes do not show stale ETA.
func TestNewOpsETASampleWindowOmitsRemainingWhenComplete(t *testing.T) {
	startedAt := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)

	got := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(4*time.Second), 4, 4, OpsDefaultETAMinimumSamples)

	require.True(t, got.MinimumSamplesMet)
	require.NotNil(t, got.Rate)
	require.Nil(t, got.Remaining)
}

// TestOpsPreflightScope_TenantContextJSONContract verifies progress preflight
// can carry the same optional nested tenant-context object as final reports.
func TestOpsPreflightScope_TenantContextJSONContract(t *testing.T) {
	t.Parallel()

	noContext, err := json.Marshal(OpsPreflightScope{Phase: "preflight"})
	require.NoError(t, err)
	require.NotContains(t, string(noContext), "tenantContext")

	ctx, err := NewTenantContext(TenantContextModeDiscovery, TenantContextInput{
		Filter:             TenantContextFilterNamed,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs:  []string{"tenant-b", "tenant-a"},
	})
	require.NoError(t, err)

	withContext, err := json.Marshal(OpsPreflightScope{
		Phase:         "preflight",
		TenantContext: &ctx,
	})
	require.NoError(t, err)
	require.JSONEq(t, `{
		"phase": "preflight",
		"consequenceSummary": {},
		"tenantContext": {
			"mode": "discovery",
			"filter": "named",
			"configuredTenantId": "tenant-a",
			"resolvedTenantIds": ["tenant-a", "tenant-b"],
			"unknownTargetCount": 0,
			"crossTenant": true,
			"warnings": [
				{
					"code": "multiple_tenants",
					"message": "affected tenants: tenant-a, tenant-b"
				}
			]
		}
	}`, string(withContext))
}

// TestOpsAuditReports_CarryTenantContext verifies every affected ops audit
// report model has a shared optional tenant-context field.
func TestOpsAuditReports_CarryTenantContext(t *testing.T) {
	t.Parallel()

	ctx, err := NewDiscoveryTenantContext("")
	require.NoError(t, err)

	require.Same(t, &ctx, RetentionAuditReport{TenantContext: &ctx}.TenantContext)
	require.Same(t, &ctx, OrphanPurgeReport{TenantContext: &ctx}.TenantContext)
	require.Same(t, &ctx, IncidentPurgeReport{TenantContext: &ctx}.TenantContext)
	require.Same(t, &ctx, AllProcessDefinitionsPurgeReport{TenantContext: &ctx}.TenantContext)
	require.Same(t, &ctx, OpsRepairAuditReport{TenantContext: &ctx}.TenantContext)
	require.Same(t, &ctx, SmokeTestAuditReport{TenantContext: &ctx}.TenantContext)
}

// TestOpsCompletionProgressJSONContract verifies completion facts expose
// service-owned lifecycle data without rendered command wording.
func TestOpsCompletionProgressJSONContract(t *testing.T) {
	t.Parallel()

	affected := 0
	raw, err := json.Marshal(OpsProgressEvent{
		Kind: OpsProgressEventKindCompletion,
		Completion: &OpsCompletionProgress{
			Phase:            "deleting process-instance trees",
			CoreResource:     "process-instance root tree(s)",
			Total:            4,
			Identity:         "2251799813685251",
			Disposition:      OpsCompletionDispositionFailed,
			FailureDetail:    "context deadline exceeded",
			AffectedResource: "affected process instance(s)",
			AffectedCount:    &affected,
		},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{
		"kind": "completion",
		"completion": {
			"phase": "deleting process-instance trees",
			"coreResource": "process-instance root tree(s)",
			"total": 4,
			"identity": "2251799813685251",
			"disposition": "failed",
			"failureDetail": "context deadline exceeded",
			"affectedResource": "affected process instance(s)",
			"affectedCount": 0
		}
	}`, string(raw))
}

// TestOpsStageProgressJSONContract verifies stage-entry facts expose
// service-owned phase, resource, total, and planned affected scope fields.
func TestOpsStageProgressJSONContract(t *testing.T) {
	t.Parallel()

	total := 2
	plannedAffected := 7
	raw, err := json.Marshal(OpsProgressEvent{
		Kind: OpsProgressEventKindStage,
		Stage: &OpsStageProgress{
			Phase:                "cancel",
			CoreResource:         "process-instance tree(s)",
			Total:                &total,
			PlannedAffectedCount: &plannedAffected,
		},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{
		"kind": "stage",
		"stage": {
			"phase": "cancel",
			"coreResource": "process-instance tree(s)",
			"total": 2,
			"plannedAffectedCount": 7
		}
	}`, string(raw))
}

// TestOpsStageProgressOmitsUnavailableCounts verifies nil stage totals remain
// absent while known zero counts survive JSON serialization.
func TestOpsStageProgressOmitsUnavailableCounts(t *testing.T) {
	t.Parallel()

	zero := 0
	raw, err := json.Marshal(OpsProgressEvent{
		Kind: OpsProgressEventKindStage,
		Stage: &OpsStageProgress{
			Phase:                "drain process instances",
			PlannedAffectedCount: &zero,
		},
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	stage := got["stage"].(map[string]any)
	require.NotContains(t, stage, "total")
	require.Equal(t, float64(0), stage["plannedAffectedCount"])
}

// TestOpsCompletionDispositionJSONContract verifies lifecycle outcomes stay
// generic so services cannot smuggle command-rendered completion verbs.
func TestOpsCompletionDispositionJSONContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		disposition OpsCompletionDisposition
		want        string
	}{
		{
			name:        "accepted no-wait work is submitted",
			disposition: OpsCompletionDispositionSubmitted,
			want:        "submitted",
		},
		{
			name:        "waited work is confirmed",
			disposition: OpsCompletionDispositionConfirmed,
			want:        "confirmed",
		},
		{
			name:        "failed work stays failed",
			disposition: OpsCompletionDispositionFailed,
			want:        "failed",
		},
	}

	renderedVerbs := []string{"cancelled", "canceled", "deleted", "deployed", "repaired", "started", "satisfied"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(OpsCompletionProgress{
				Phase:       "delete",
				Identity:    "2251799813685251",
				Disposition: tt.disposition,
			})
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(raw, &got))
			require.Equal(t, tt.want, got["disposition"])
			for _, renderedVerb := range renderedVerbs {
				require.NotEqual(t, renderedVerb, got["disposition"])
			}
		})
	}
}

// TestOpsCompletionProgressOmitsUnknownAffectedCount verifies nil affected
// counts stay distinct from a trustworthy zero in the wire contract.
func TestOpsCompletionProgressOmitsUnknownAffectedCount(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(OpsProgressEvent{
		Kind: OpsProgressEventKindCompletion,
		Completion: &OpsCompletionProgress{
			Phase:            "submitting process-instance cancellation",
			CoreResource:     "process-instance root tree(s)",
			Identity:         "2251799813685252",
			Disposition:      OpsCompletionDispositionSubmitted,
			AffectedResource: "affected process instance(s)",
		},
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	completion := got["completion"].(map[string]any)
	require.Equal(t, "submitted", completion["disposition"])
	require.NotContains(t, completion, "affectedCount")
}
