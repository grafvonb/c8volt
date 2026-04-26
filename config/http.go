// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type HTTP struct {
	Timeout string `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

func (h *HTTP) Validate() error {
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

func (h *HTTP) Normalize() error {
	var errs []error
	if h.Timeout == "" {
		h.Timeout = "30s"
	}
	return errors.Join(errs...)
}
