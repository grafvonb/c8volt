// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import "github.com/grafvonb/c8volt/typex"

// NormalizeProcessInstanceSuspensionKeys preserves the suspension normalization
// entry point while sharing exact-key validation with other instance workflows.
func NormalizeProcessInstanceSuspensionKeys(keys typex.Keys) (typex.Keys, error) {
	return NormalizeProcessInstanceKeys(keys)
}
