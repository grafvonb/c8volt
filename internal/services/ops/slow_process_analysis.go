// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// AnalyseSlowProcessInstances reserves the ops service boundary for read-only slow process analysis.
func (s *Service) AnalyseSlowProcessInstances(ctx context.Context, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (d.SlowProcessAnalysisResult, error) {
	_ = ctx
	_ = opts
	capturedAt := request.CapturedNow
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
		request.CapturedNow = capturedAt
	}
	return d.SlowProcessAnalysisResult{
		Request:    request,
		CapturedAt: capturedAt,
		Items:      []d.SlowProcessAnalysisProcessInstance{},
		Count:      0,
		Empty:      true,
	}, fmt.Errorf("%w: slow process analysis implementation is pending", d.ErrUnsupported)
}
