// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
)

// validateDescendantPlanTraversal rejects incomplete scope and missing or mismatched
// observations, including duplicate members already observed by an earlier seed.
func validateDescendantPlanTraversal(seed string, result traversal.Result, operation string) error {
	if result.Outcome == traversal.OutcomePartial || result.Outcome == traversal.OutcomeUnresolved {
		return fmt.Errorf("%w: %s planning requires complete descendant discovery", d.ErrPrecondition, operation)
	}
	if result.Outcome != traversal.OutcomeComplete || result.Mode != traversal.ModeDescendants || result.StartKey != seed || result.RootKey != seed || len(result.MissingAncestors) != 0 {
		return fmt.Errorf("%w: invalid %s descendant scope", d.ErrMalformedResponse, operation)
	}
	foundSeed := false
	for _, key := range result.Keys {
		normalized, err := common.NormalizeProcessInstanceKeys(typex.Keys{key})
		if err != nil || normalized[0] != key {
			return fmt.Errorf("%w: invalid discovered process-instance key", d.ErrMalformedResponse)
		}
		instance, ok := result.Chain[key]
		if !ok || instance.Key != key {
			return fmt.Errorf("%w: missing or mismatched process-instance observation", d.ErrMalformedResponse)
		}
		if key == seed {
			foundSeed = true
		}
	}
	if !foundSeed {
		return fmt.Errorf("%w: descendant scope omits selected process instance", d.ErrMalformedResponse)
	}
	return nil
}
