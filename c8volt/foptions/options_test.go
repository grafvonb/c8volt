// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package foptions

import (
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestProgressCompletionFactMapsFromServiceOptions verifies the public facade
// callback receives completion facts through the existing service call option.
func TestProgressCompletionFactMapsFromServiceOptions(t *testing.T) {
	t.Parallel()

	var events []ProgressEvent
	opts := MapFacadeOptionsToCallOptions([]FacadeOption{
		WithProgress(func(event ProgressEvent) {
			events = append(events, event)
		}),
	})
	cfg := services.ApplyCallOptions(opts)
	require.NotNil(t, cfg.Progress)

	affected := 0
	cfg.Progress(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindCompletion,
		Completion: &d.OpsCompletionProgress{
			Phase:            "deleting process-instance trees",
			CoreResource:     "process-instance root tree(s)",
			Total:            3,
			Identity:         "2251799813685251",
			Disposition:      d.OpsCompletionDispositionConfirmed,
			FailureDetail:    "",
			AffectedResource: "affected process instance(s)",
			AffectedCount:    &affected,
		},
	})

	require.Len(t, events, 1)
	require.Equal(t, ProgressEventKindCompletion, events[0].Kind)
	require.Equal(t, &CompletionProgress{
		Phase:            "deleting process-instance trees",
		CoreResource:     "process-instance root tree(s)",
		Total:            3,
		Identity:         "2251799813685251",
		Disposition:      CompletionDispositionConfirmed,
		AffectedResource: "affected process instance(s)",
		AffectedCount:    &affected,
	}, events[0].Completion)
}

// TestProgressStageFactMapsFromServiceOptions verifies stage-entry facts reach
// facade-option callbacks while preserving nil, known zero, and pointer copies.
func TestProgressStageFactMapsFromServiceOptions(t *testing.T) {
	t.Parallel()

	var events []ProgressEvent
	cfg := services.ApplyCallOptions(MapFacadeOptionsToCallOptions([]FacadeOption{
		WithProgress(func(event ProgressEvent) {
			events = append(events, event)
		}),
	}))
	require.NotNil(t, cfg.Progress)

	total := 0
	plannedAffected := 5
	cfg.Progress(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindStage,
		Stage: &d.OpsStageProgress{
			Phase:                "cancel",
			CoreResource:         "process-instance tree(s)",
			Total:                &total,
			PlannedAffectedCount: &plannedAffected,
		},
	})
	cfg.Progress(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindStage,
		Stage: &d.OpsStageProgress{
			Phase: "drain process instances",
		},
	})

	require.Len(t, events, 2)
	require.Equal(t, ProgressEventKindStage, events[0].Kind)
	require.Equal(t, &StageProgress{
		Phase:                "cancel",
		CoreResource:         "process-instance tree(s)",
		Total:                &total,
		PlannedAffectedCount: &plannedAffected,
	}, events[0].Stage)
	require.NotSame(t, &total, events[0].Stage.Total)
	require.NotSame(t, &plannedAffected, events[0].Stage.PlannedAffectedCount)

	require.Equal(t, ProgressEventKindStage, events[1].Kind)
	require.NotNil(t, events[1].Stage)
	require.Nil(t, events[1].Stage.Total)
	require.Nil(t, events[1].Stage.PlannedAffectedCount)
}

// TestProgressCompletionDispositionMapsFromServiceOptions verifies the facade
// option adapter preserves service lifecycle facts without operation wording.
func TestProgressCompletionDispositionMapsFromServiceOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   d.OpsCompletionDisposition
		want CompletionDisposition
	}{
		{
			name: "accepted no-wait work is submitted",
			in:   d.OpsCompletionDispositionSubmitted,
			want: CompletionDispositionSubmitted,
		},
		{
			name: "waited work is confirmed",
			in:   d.OpsCompletionDispositionConfirmed,
			want: CompletionDispositionConfirmed,
		},
		{
			name: "failed work stays failed",
			in:   d.OpsCompletionDispositionFailed,
			want: CompletionDispositionFailed,
		},
	}

	renderedVerbs := []CompletionDisposition{"cancelled", "canceled", "deleted", "deployed", "repaired", "started", "satisfied"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got ProgressEvent
			cfg := services.ApplyCallOptions(MapFacadeOptionsToCallOptions([]FacadeOption{
				WithProgress(func(event ProgressEvent) {
					got = event
				}),
			}))

			cfg.Progress(d.OpsProgressEvent{
				Kind: d.OpsProgressEventKindCompletion,
				Completion: &d.OpsCompletionProgress{
					Phase:       "delete",
					Identity:    "2251799813685251",
					Disposition: tt.in,
				},
			})

			require.Equal(t, ProgressEventKindCompletion, got.Kind)
			require.NotNil(t, got.Completion)
			require.Equal(t, tt.want, got.Completion.Disposition)
			for _, renderedVerb := range renderedVerbs {
				require.NotEqual(t, renderedVerb, got.Completion.Disposition)
			}
		})
	}
}

// TestProgressCompletionFactPreservesUnknownAffectedCount verifies nil remains
// unavailable instead of being converted to a zero affected count.
func TestProgressCompletionFactPreservesUnknownAffectedCount(t *testing.T) {
	t.Parallel()

	var got ProgressEvent
	cfg := services.ApplyCallOptions(MapFacadeOptionsToCallOptions([]FacadeOption{
		WithProgress(func(event ProgressEvent) {
			got = event
		}),
	}))

	cfg.Progress(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindCompletion,
		Completion: &d.OpsCompletionProgress{
			Phase:       "submitting process-instance cancellation",
			Identity:    "2251799813685252",
			Disposition: d.OpsCompletionDispositionSubmitted,
		},
	})

	require.Equal(t, ProgressEventKindCompletion, got.Kind)
	require.NotNil(t, got.Completion)
	require.Equal(t, CompletionDispositionSubmitted, got.Completion.Disposition)
	require.Nil(t, got.Completion.AffectedCount)
}
