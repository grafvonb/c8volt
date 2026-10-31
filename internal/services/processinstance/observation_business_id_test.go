// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later
package processinstance

import (
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestObservedPlansBusinessID freezes resource-specific identity, including nil and backend-empty values.
func TestObservedPlansBusinessID(t *testing.T) {
	t.Parallel()
	for _, value := range []*string{nil, new(string), func() *string { s := "old"; return &s }()} {
		instance := d.ProcessInstance{Key: "101", State: d.StateActive, BusinessId: value}
		suspend := classifySuspensionPlanItem(instance)
		resume := classifyResumptionPlanItem(instance)
		planner := processInstanceVariablePlanner{requested: map[string]any{"x": true}}
		require.NoError(t, planner.addInstance(instance, nil))
		plan := d.ProcessInstanceVariableUpdatePlan{Instances: planner.instances}
		frozen := copyProcessInstanceVariableUpdatePlan(plan)
		for _, got := range []*string{suspend.BusinessId, resume.BusinessId, planner.instances[0].BusinessId, frozen.Instances[0].BusinessId} {
			require.Equal(t, value, got)
			if value != nil {
				require.NotSame(t, value, got)
			}
		}
		if value != nil {
			*plan.Instances[0].BusinessId = "new"
			require.Equal(t, *value, *frozen.Instances[0].BusinessId)
		}
	}
}
