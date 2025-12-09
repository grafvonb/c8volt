package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var ErrCmdAborted = errors.New("aborted by user")

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

func confirmCmdOrAbort(autoConfirm bool, prompt string) error {
	if autoConfirm || !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}
	fmt.Printf("%s [y/N]: ", prompt)
	in := bufio.NewScanner(os.Stdin)
	if !in.Scan() {
		return ErrCmdAborted
	}
	switch strings.ToLower(strings.TrimSpace(in.Text())) {
	case "y", "yes":
		return nil
	default:
		return ErrCmdAborted
	}
}

func mergeAndValidateKeys(baseKeys []string, log *slog.Logger, cfg *config.Config) typex.Keys {
	keys := append([]string{}, baseKeys...)
	if inKeys, err := readKeysFromStdin(); err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("reading stdin: %w", err))
	} else if len(inKeys) > 0 {
		if ok, firstBadKey, firstBadIndex := validateKeys(inKeys); !ok {
			if strings.HasPrefix(firstBadKey, "filter: ") {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("validating keys from stdin failed: use --keys-only flag to get only keys as input"))
			}
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("validating keys from stdin failed: line %q at index %d is not a valid key; have you forgotten to use --keys-only flag in case of c8volt commands?", firstBadKey, firstBadIndex))
		}
		keys = append(keys, inKeys...)
	}
	return keys
}
