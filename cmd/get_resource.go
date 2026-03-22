package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var flagGetResourceID string

var getResourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "Get a resource by id",
	Long: "Get a single resource by id.\n" +
		"It requires --id to select exactly one resource and renders the standard single-resource view.",
	Aliases: []string{"r"},
	Args: func(cmd *cobra.Command, args []string) error {
		_, err := validatedResourceID()
		return err
	},
	Run: runGetResource,
}

func runGetResource(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
	}

	id, err := validatedResourceID()
	if err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
	}

	runGetResourceByID(cmd, cli, log, cfg.App.NoErrCodes, id)
}

func runGetResourceByID(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, id string) {
	log.Debug(fmt.Sprintf("fetching resource by id: %s", id))
	resource, err := cli.GetResource(cmd.Context(), id, collectOptions()...)
	if err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error fetching resource by id %s: %w", id, err))
	}
	if err := resourceView(cmd, resource); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error rendering resource view: %w", err))
	}
}

func init() {
	getCmd.AddCommand(getResourceCmd)

	fs := getResourceCmd.Flags()
	fs.StringVarP(&flagGetResourceID, "id", "i", "", "resource id to fetch")
	_ = getResourceCmd.MarkFlagRequired("id")
}

func validatedResourceID() (string, error) {
	id := strings.TrimSpace(flagGetResourceID)
	if id == "" {
		return "", fmt.Errorf("resource lookup requires a non-empty --id")
	}
	return id, nil
}
