package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultBackoffStrategy   = "exponential"
	defaultBackoffMultiplier = 2.0
)

var (
	defaultBackoffInitialDelay = 500 * time.Millisecond
	defaultBackoffMaxDelay     = 8 * time.Second
	defaultBackoffMaxRetries   = 0 // 0 = unlimited
	defaultBackoffTimeout      = 2 * time.Minute
)

func addBackoffFlagsAndBindings(cmd *cobra.Command, v *viper.Viper) {
	fs := cmd.PersistentFlags()

	fs.Duration("backoff-timeout", defaultBackoffTimeout, "overall timeout for the retry loop")
	fs.Int("backoff-max-retries", defaultBackoffMaxRetries, "max retry attempts (0 = unlimited)")

	_ = v.BindPFlag("app.backoff.timeout", fs.Lookup("backoff-timeout"))
	_ = v.BindPFlag("app.backoff.max_retries", fs.Lookup("backoff-max-retries"))
	v.SetDefault("app.backoff.timeout", defaultBackoffTimeout)
	v.SetDefault("app.backoff.max_retries", defaultBackoffMaxRetries)

	// Config-only keys
	v.SetDefault("app.backoff.strategy", defaultBackoffStrategy) // fixed|exponential
	v.SetDefault("app.backoff.initial_delay", defaultBackoffInitialDelay)
	v.SetDefault("app.backoff.max_delay", defaultBackoffMaxDelay)
	v.SetDefault("app.backoff.multiplier", defaultBackoffMultiplier)
}

//nolint:unused
func requireAnyFlag(cmd *cobra.Command, flags ...string) error {
	for _, f := range flags {
		if cmd.Flags().Changed(f) {
			return nil
		}
	}
	return fmt.Errorf("one of %v must be provided", flags)
}
