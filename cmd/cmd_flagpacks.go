package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
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

func addBackoffFlagsAndBindings(cmd *cobra.Command) {
	fs := cmd.PersistentFlags()

	fs.Duration("backoff-timeout", defaultBackoffTimeout, "overall timeout for the retry loop")
	fs.Int("backoff-max-retries", defaultBackoffMaxRetries, "max retry attempts (0 = unlimited)")
	_ = fs.MarkHidden("backoff-timeout")
	_ = fs.MarkHidden("backoff-max-retries")
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
