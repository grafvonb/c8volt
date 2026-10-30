// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// VersionCheckStatus describes an attempted advisory; no outcome means no attempt.
type VersionCheckStatus string

const (
	// VersionCheckMatched means the gateway and configured release lines agree.
	VersionCheckMatched VersionCheckStatus = "matched"
	// VersionCheckMismatched preserves configuration despite a different gateway line.
	VersionCheckMismatched VersionCheckStatus = "mismatched"
	// VersionCheckFailed means compatibility could not be established.
	VersionCheckFailed VersionCheckStatus = "failed"
)

// VersionCheckFailure is a bounded cause, never raw backend or transport evidence.
type VersionCheckFailure string

const (
	// VersionCheckAccessDenied covers unauthorized and forbidden lookups.
	VersionCheckAccessDenied VersionCheckFailure = "access_denied"
	// VersionCheckTimeout covers deadlines and network timeouts.
	VersionCheckTimeout VersionCheckFailure = "timeout"
	// VersionCheckInvalidVersion covers absent, malformed, or unsafe observations.
	VersionCheckInvalidVersion VersionCheckFailure = "invalid_version"
	// VersionCheckUnavailable covers remaining lookup failures, including redirects.
	VersionCheckUnavailable VersionCheckFailure = "unavailable"
	// VersionCheckCanceled records cancellation of the winning caller.
	VersionCheckCanceled VersionCheckFailure = "canceled"
)

// VersionCheckOutcome is an immutable value suitable for diagnostic presentation.
// ObservedVersion is validated safe ASCII of at most 128 bytes and is absent on failure.
type VersionCheckOutcome struct {
	Status            VersionCheckStatus
	ConfiguredVersion string
	ObservedVersion   string
	FailureReason     VersionCheckFailure
}

// WarningNeeded depends on the outcome, independently of any logger filtering.
func (o VersionCheckOutcome) WarningNeeded() bool {
	return o.Status == VersionCheckMismatched || o.Status == VersionCheckFailed
}
