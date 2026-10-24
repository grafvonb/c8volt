// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// HTTP configures shared request deadlines, optional pacing and DEBUG reporting.
type HTTP struct {
	RequestRateReportInterval string  `mapstructure:"request_rate_report_interval" json:"request_rate_report_interval" yaml:"request_rate_report_interval"`
	RequestRate               float64 `mapstructure:"request_rate" json:"request_rate" yaml:"request_rate"`
	Timeout                   string  `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

// Validate rejects invalid deadlines, rates and report intervals before remote work.
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
	interval, err := time.ParseDuration(h.RequestRateReportInterval)
	if err != nil || interval < 0 {
		return fmt.Errorf("http.request_rate_report_interval: must be a parseable nonnegative duration")
	}
	return nil
}

// Normalize supplies omitted defaults while preserving explicit zero controls.
func (h *HTTP) Normalize() error {
	return h.normalizeWithConfiguredKeys(func(string) bool { return false })
}

// normalizeWithConfiguredKeys preserves explicit empty intervals for validation.
func (h *HTTP) normalizeWithConfiguredKeys(isConfigured func(string) bool) error {
	if h.Timeout == "" {
		h.Timeout = "30s"
	}
	if h.RequestRateReportInterval == "" && !isConfigured("http.request_rate_report_interval") {
		h.RequestRateReportInterval = "5s"
	}
	return nil
}

// validateRequestRateInput rejects malformed scalars before weak decoding can
// coerce empty strings or booleans to a valid-looking zero or one.
func validateRequestRateInput(value any) error {
	if _, err := strconv.ParseFloat(fmt.Sprint(value), 64); err != nil {
		return fmt.Errorf("http.request_rate: must be a finite nonnegative number")
	}
	return nil
}
