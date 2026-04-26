// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

func DefaultVal[T comparable](val, def T) T {
	var zero T
	if val == zero {
		return def
	}
	return val
}
