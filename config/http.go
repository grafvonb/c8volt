// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// HTTP configures shared request deadlines and optional invocation-wide pacing.
type HTTP struct {
	RequestRate float64 `mapstructure:"request_rate" json:"request_rate" yaml:"request_rate"`
	Timeout     string  `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

// Validate rejects invalid deadlines and rates before any remote work.
func (h *HTTP) Validate() error {
	if math.IsNaN(h.RequestRate) || math.IsInf(h.RequestRate, 0) || h.RequestRate < 0 {
		return fmt.Errorf("http.request_rate: must be finite and nonnegative")
	}
	if strings.TrimSpace(h.Timeout) == "" {
		return fmt.Errorf("http.timeout: timeout must not be empty")
	}
	d, err := time.ParseDuration(h.Timeout)
	if err != nil {
		return fmt.Errorf("http.timeout: invalid duration: %w", err)
	}
	if d <= 0 {
		return fmt.Errorf("http.timeout: timeout must be a positive duration")
	}
	return nil
}

// Normalize supplies the timeout default while preserving an explicit zero rate.
func (h *HTTP) Normalize() error {
	var errs []error
	if h.Timeout == "" {
		h.Timeout = "30s"
	}
	return errors.Join(errs...)
}

// validateRequestRateInput rejects malformed scalars before weak decoding can
// coerce empty strings or booleans to a valid-looking zero or one.
func validateRequestRateInput(value any) error {
	if _, err := strconv.ParseFloat(fmt.Sprint(value), 64); err != nil {
		return fmt.Errorf("http.request_rate: must be a finite nonnegative number")
	}
	return nil
}
