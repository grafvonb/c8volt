// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package batchoperation

type BatchOperation struct {
	Key                      string   `json:"key,omitempty"`
	Type                     string   `json:"type,omitempty"`
	State                    string   `json:"state,omitempty"`
	OperationsTotalCount     int32    `json:"operationsTotalCount,omitempty"`
	OperationsCompletedCount int32    `json:"operationsCompletedCount,omitempty"`
	OperationsFailedCount    int32    `json:"operationsFailedCount,omitempty"`
	Errors                   []string `json:"errors,omitempty"`
	StatusCode               int      `json:"statusCode,omitempty"`
	Status                   string   `json:"status,omitempty"`
}
