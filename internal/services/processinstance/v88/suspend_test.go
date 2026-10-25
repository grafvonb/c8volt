// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestProcessInstanceSuspensionUnsupported verifies the guard precedes client,
// configuration and input access, even for preview and no-wait on an empty service.
func TestProcessInstanceSuspensionUnsupported(t *testing.T) {
	t.Parallel()
	service := &Service{}
	require.ErrorIs(t, service.ValidateProcessInstanceSuspension(), d.ErrUnsupported)
	for _, opts := range [][]services.CallOption{nil, {services.WithDryRun()}, {services.WithNoWait()}, {services.WithDryRun(), services.WithNoWait()}} {
		result, err := service.SuspendProcessInstance(context.Background(), "bad", opts...)
		require.ErrorIs(t, err, d.ErrUnsupported)
		require.Equal(t, "bad", result.Key)
		require.Equal(t, d.ProcessInstanceSuspensionSubmissionNotSubmitted, result.Submission)
		require.Equal(t, d.ProcessInstanceSuspensionConfirmationNotRequested, result.Confirmation)
	}
}
