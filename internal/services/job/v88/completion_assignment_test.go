// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestCompletionAssignmentUnsupported proves even forged and empty plans reject before dependencies are touched.
func TestCompletionAssignmentUnsupported(t *testing.T) {
	svc := &Service{}
	value := toolx.Ptr("Order")
	_, err := svc.PrepareJobCompletionBatch(context.Background(), d.CompletionRequest{SetBusinessID: value})
	require.ErrorIs(t, err, d.ErrUnsupported)
	_, err = svc.ExecuteJobCompletionBatch(context.Background(), d.PreparedCompletionBatch{SetBusinessID: value, Status: d.CompletionPreparationComplete}, 1)
	require.ErrorIs(t, err, d.ErrUnsupported)
	_, err = svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{Mode: d.JobWorkerOutcomeCompletion, SetBusinessID: value})
	require.ErrorIs(t, err, d.ErrUnsupported)
	_, err = svc.submitBulkCompletion(context.Background(), d.Job{}, d.CompletionPayload{SetBusinessID: value})
	require.ErrorIs(t, err, d.ErrUnsupported)
}
