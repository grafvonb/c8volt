// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

// EvidenceOutcomeFailed keeps successful observations and explicit coverage
// limitations out of the failure list. Unknown outcomes fail closed.
func EvidenceOutcomeFailed(outcome string) bool {
	switch outcome {
	case "pass", "live-covered", "partially-live-covered", "dry-run-covered", "skipped-prerequisite", "unsupported":
		return false
	default:
		return true
	}
}
