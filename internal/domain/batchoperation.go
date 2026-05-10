// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type BatchOperation struct {
	Key                      string
	Type                     string
	State                    string
	OperationsTotalCount     int32
	OperationsCompletedCount int32
	OperationsFailedCount    int32
	Errors                   []string
	StatusCode               int
	Status                   string
}
