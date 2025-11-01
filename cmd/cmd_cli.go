package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

func NewCli(cmd *cobra.Command) (c8volt.API, *slog.Logger, *config.Config, error) {
	log, _ := logging.FromContext(cmd.Context())
	svcs, err := NewFromContext(cmd.Context())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting services from context: %w", err)
	}
	cli, err := c8volt.New(
		c8volt.WithConfig(svcs.Config),
		c8volt.WithHTTPClient(svcs.HTTP.Client()),
		c8volt.WithLogger(log),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating c8volt client: %w", err)
	}
	return cli, log, svcs.Config, nil
}
