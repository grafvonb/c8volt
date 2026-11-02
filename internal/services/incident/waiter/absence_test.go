// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"errors"
	"fmt"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// errorIncidentWaiter provides one classified lookup failure without a backend.
type errorIncidentWaiter struct {
	fakeIncidentWaiter
	err   error
	calls int
}

// GetIncident returns the injected lookup error and counts observations.
func (f *errorIncidentWaiter) GetIncident(context.Context, string, ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
	f.calls++
	return d.ProcessInstanceIncidentDetail{}, f.err
}

// TestWaitForIncidentResolvedRequiresClassifiedAbsence prevents HTTP diagnostic text
// from turning lookup failure into successful resolution.
func TestWaitForIncidentResolvedRequiresClassifiedAbsence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		err    error
		absent bool
	}{
		{"not found", d.ErrNotFound, true},
		{"wrapped not found", fmt.Errorf("lookup: %w", d.ErrNotFound), true},
		{"bad request port", fmt.Errorf("%w: 400 GET http://owned.invalid:40401/incidents/123", d.ErrBadRequest), false},
		{"forbidden key", fmt.Errorf("%w: 403 GET http://owned.invalid/incidents/404123", d.ErrForbidden), false},
		{"unauthorized body", fmt.Errorf("%w: configuration not found", d.ErrUnauthorized), false},
		{"unclassified absence text", errors.New("dependency does not exist"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &errorIncidentWaiter{err: tc.err}
			result, err := WaitForIncidentResolved(context.Background(), reader, incidentWaiterTestConfig(), incidentWaiterTestLogger(), "123")
			if tc.absent {
				require.NoError(t, err)
				require.True(t, result.Ok)
			} else {
				require.ErrorIs(t, err, tc.err)
				require.False(t, result.Ok)
			}
			require.Equal(t, 1, reader.calls)
		})
	}
}
