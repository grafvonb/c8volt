// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

// SupportsFullProcessDefinitionHistoryDeletion reports whether the configured
// release line supports deleting a process definition with its full history.
func SupportsFullProcessDefinitionHistoryDeletion(version CamundaVersion) bool {
	switch version {
	case V89, V810:
		return true
	default:
		return false
	}
}
