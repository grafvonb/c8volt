package config

import (
	"errors"
	"fmt"
	"strings"
)

type HTTP struct {
	Timeout string `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

func (h *HTTP) Validate() error {
	if strings.TrimSpace(h.Timeout) == "" {
		return fmt.Errorf("timeout must not be empty")
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
