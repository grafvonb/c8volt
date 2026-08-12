// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

// ProductionFixturePrefix returns the embedded production fixture prefix for a Camunda compatibility line.
func ProductionFixturePrefix(version CamundaVersion) (string, bool) {
	switch version {
	case V87:
		return "C87_", true
	case V88:
		return "C88_", true
	case V89, V810:
		return "C89_", true
	default:
		return "", false
	}
}
