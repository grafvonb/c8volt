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
