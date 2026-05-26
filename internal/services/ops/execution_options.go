// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import "github.com/grafvonb/c8volt/internal/services"

func compactOpsExecutionOptions(opts ...services.CallOption) []services.CallOption {
	cfg := services.ApplyCallOptions(opts)
	out := append([]services.CallOption{}, opts...)
	if cfg.Verbose {
		return out
	}
	return append(out, services.WithSuppressWorkflowDetailLogs(), services.WithSuppressProcessInstanceDetailLogs())
}
