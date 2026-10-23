// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"fmt"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// NormalizeProcessInstanceSearchFilter validates explicit plural selection before
// any I/O and returns an independent, canonical state slice. Scalar behavior is
// preserved. Combined search uses the 8.10 suspension capability on all entry
// paths, including direct incident discovery which can bypass adapter searches.
func NormalizeProcessInstanceSearchFilter(filter d.ProcessInstanceFilter, validateStates func(d.States) error) (d.ProcessInstanceFilter, error) {
	if len(filter.States) == 0 {
		return filter, validateStates(d.States{filter.State})
	}
	if filter.State != "" {
		return filter, fmt.Errorf("%w: singular and plural process-instance states are mutually exclusive", d.ErrValidation)
	}
	states := make(d.States, 0, len(filter.States))
	for _, raw := range filter.States {
		state := d.State(strings.ToUpper(strings.TrimSpace(string(raw))))
		if state == d.StateCanceled {
			state = d.StateTerminated
		}
		switch state {
		case d.StateActive, d.StateSuspended, d.StateCompleted, d.StateTerminated:
		default:
			return filter, fmt.Errorf("%w: invalid process-instance search state %q", d.ErrValidation, raw)
		}
		if !states.Contains(state) {
			states = append(states, state)
		}
	}
	if err := validateStates(states); err != nil {
		return filter, err
	}
	// Even a one-value plural selector requires the combined-filter capability;
	// older adapters must never silently ignore it or send a broader scalar query.
	if err := validateStates(d.States{d.StateSuspended}); err != nil {
		return filter, fmt.Errorf("plural process-instance search requires Camunda 8.10: %w", err)
	}
	filter.States = states
	return filter, nil
}
