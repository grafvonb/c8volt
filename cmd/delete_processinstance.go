package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

var (
	flagDeletePIKeys      []string
	flagDeletePIWithForce bool

	flagDeletePIWorkers  int
	flagDeletePIFailFast bool
)

var deleteProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Short:   "Delete a process instance by its key",
	Aliases: []string{"pi"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("initializing client: %w", err))
		}

		keys := append([]string{}, flagDeletePIKeys...)
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

		switch {
		case len(keys) > 0:
		default:
			searchFilterOpts, ok := populatePISearchFilterOpts()
			if !ok {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("either at least one --key is required, or sufficient filtering options to search for process instances to delete"))
			}
			pisr, err := cli.SearchProcessInstances(cmd.Context(), searchFilterOpts, consts.MaxPISearchSize, collectOptions()...)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error fetching process instances: %w", err))
			}
			keys = make([]string, 0, len(pisr.Items))
			for _, pi := range pisr.Items {
				keys = append(keys, pi.Key)
			}
		}
		if len(keys) == 0 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("no process instance keys provided or found to delete"))
		}
		prompt := fmt.Sprintf("You are about to delete %d process instance(s)?", len(keys))
		if err := confirmCmdOrAbort(flagCmdAutoConfirm, prompt); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		_, err = cli.DeleteProcessInstances(cmd.Context(), keys, flagDeletePIWorkers, flagDeletePIFailFast, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deleting process instance(s): %w", err))
		}
	},
}

func init() {
	deleteCmd.AddCommand(deleteProcessInstanceCmd)

	fs := deleteProcessInstanceCmd.Flags()
	fs.BoolVar(&flagDeleteNoWait, "no-wait", false, "skip waiting for the deletion to be fully processed (no status checks)")
	fs.BoolVar(&flagDeleteNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before deleting it")
	fs.StringSliceVarP(&flagDeletePIKeys, "key", "k", nil, "process instance key(s) to delete")
	fs.BoolVar(&flagDeletePIWithForce, "force", false, "force cancellation of the process instance(s), prior to deletion")

	fs.IntVarP(&flagDeletePIWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagDeletePIFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	fs.StringVarP(&flagGetPIBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.Int32Var(&flagGetPIProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPIProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")
}
