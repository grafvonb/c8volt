// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvidenceOutcomeFailed(t *testing.T) {
	t.Parallel()
	for _, outcome := range []string{"pass", "live-covered", "partially-live-covered", "dry-run-covered", "skipped-prerequisite", "unsupported"} {
		require.False(t, EvidenceOutcomeFailed(outcome), outcome)
	}
	for _, outcome := range []string{"fail", "failed", "nonzero", "", "unknown"} {
		require.True(t, EvidenceOutcomeFailed(outcome), outcome)
	}
}
