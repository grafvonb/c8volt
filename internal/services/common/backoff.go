package common

import (
	"errors"
	"fmt"
	"time"
)

type BackoffStrategy string

const (
	BackoffFixed       BackoffStrategy = "fixed"
	BackoffExponential BackoffStrategy = "exponential"
)

type BackoffConfig struct {
	Strategy     BackoffStrategy `mapstructure:"strategy" json:"strategy" yaml:"strategy"`
	InitialDelay time.Duration   `mapstructure:"initial_delay" json:"initial_delay" yaml:"initial_delay"`
	MaxDelay     time.Duration   `mapstructure:"max_delay" json:"max_delay" yaml:"max_delay"`
	MaxRetries   int             `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`
	Multiplier   float64         `mapstructure:"multiplier" json:"multiplier" yaml:"multiplier"` // Used for exponential backoff strategy
	Timeout      time.Duration   `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

func (c *BackoffConfig) NextDelay(prev time.Duration) time.Duration {
	if prev <= 0 {
		prev = c.InitialDelay
	}
	var delay time.Duration
	switch c.Strategy {
	case BackoffExponential:
		delay = time.Duration(float64(prev) * c.Multiplier)
	default: // fixed strategy
		delay = prev
	}
	if c.MaxDelay > 0 && delay > c.MaxDelay {
		delay = c.MaxDelay
	}
	return delay
}

func (c *BackoffConfig) Normalize() error {
	/*
	   strategy: exponential
	   initial_delay: 500ms
	   max_delay: 8s
	   max_retries: 0
	   multiplier: 2.0
	   timeout: 30s
	*/
	if c.Strategy == "" {
		c.Strategy = BackoffExponential
	}
	if c.InitialDelay <= 0 {
		c.InitialDelay = 500 * time.Millisecond
	}
	if c.MaxDelay < 0 {
		c.MaxDelay = 8 * time.Second
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0 // unlimited
	}
	if c.Strategy == BackoffExponential && c.Multiplier <= 1 {
		c.Multiplier = 2.0
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	return nil
}

func (c *BackoffConfig) Validate() error {
	var errs []error

	if c.Strategy != BackoffFixed && c.Strategy != BackoffExponential {
		errs = append(errs, fmt.Errorf("strategy must be either %q or %q", BackoffFixed, BackoffExponential))
	}
	if c.InitialDelay <= 0 {
		errs = append(errs, errors.New("initial_delay must be a positive duration"))
	}
	if c.MaxDelay < 0 {
		errs = append(errs, errors.New("max_delay must be non-negative"))
	}
	if c.MaxDelay > 0 && c.MaxDelay < c.InitialDelay {
		errs = append(errs, errors.New("max_delay must be greater than or equal to initial_delay"))
	}
	if c.MaxRetries < 0 {
		errs = append(errs, errors.New("max_retries must be non-negative"))
	}
	if c.Strategy == BackoffExponential && c.Multiplier <= 1 {
		errs = append(errs, errors.New("multiplier must be greater than 1 for exponential backoff"))
	}
	if c.Timeout <= 0 {
		errs = append(errs, errors.New("timeout must be a positive duration"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
