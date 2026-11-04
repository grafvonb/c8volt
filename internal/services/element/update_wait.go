// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// waitForElementVariables confirms every requested value using complete reads
// bound to the frozen owner, local scope, and known tenant. Polling never writes.
func (s *VariableUpdateService) waitForElementVariables(ctx context.Context, element d.ElementVariablePlan, opts ...services.CallOption) error {
	cfg := s.config
	if cfg == nil {
		cfg = config.New()
	}
	backoff := cfg.App.Backoff
	if backoff.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, backoff.Timeout)
		defer cancel()
	}
	delay := backoff.InitialDelay
	if delay <= 0 {
		delay = 500 * time.Millisecond
	}
	query := d.ScopeVariableQuery{ProcessInstanceKey: element.ProcessInstanceKey, ScopeKey: element.ScopeKey}
	if element.TenantId != "" {
		tenant := element.TenantId
		query.ExpectedTenantId = &tenant
	}
	opts = append(append([]services.CallOption(nil), opts...), services.WithIgnoreTenant())
	for attempts := 1; ; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		observed, err := s.variables.SearchScopeVariables(ctx, query, opts...)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		local, _, err := validateLocalVariableObservations(d.Element{ElementInstanceKey: element.ElementKey, ProcessInstanceKey: element.ProcessInstanceKey, TenantId: element.TenantId}, observed)
		if err != nil {
			return err
		}
		missing := make([]string, 0)
		for _, name := range sortedElementVariableNames(element.VerificationValues) {
			current, ok := local[name]
			if !ok || current.APITruncated || !normalizedElementVariableValuesEqual(element.VerificationValues[name], current.Value) {
				missing = append(missing, name)
			}
		}
		if len(missing) == 0 {
			return nil
		}
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			return fmt.Errorf("%w: variables %s not confirmed on element %s", d.ErrUpstream, strings.Join(missing, ", "), element.ElementKey)
		}
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}
