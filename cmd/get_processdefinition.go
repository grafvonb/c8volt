package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagGetPDKey               string
	flagGetPDBpmnProcessId     string
	flagGetPDProcessVersion    int32
	flagGetPDProcessVersionTag string
	flagGetPDLatest            bool
	flagGetPDWithStat          bool
)

var getProcessDefinitionCmd = &cobra.Command{
	Use:     "process-definition",
	Short:   "Get deployed process definitions",
	Aliases: []string{"pd", "pds"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}

		log.Debug("fetching process definitions")
		filter := populatePDSearchFilterOpts()
		if filter.Key != "" {
			log.Debug(fmt.Sprintf("searching by key: %s", filter.Key))
			pd, err := cli.GetProcessDefinition(cmd.Context(), filter.Key, collectOptions()...)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error fetching process definition by key %s: %w", filter.Key, err))
			}
			err = processDefinitionView(cmd, pd)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error rendering key-only view: %w", err))
			}
		} else {
			log.Debug(fmt.Sprintf("searching process definitions for filter %+v", filter))
			var pds process.ProcessDefinitions
			if !flagGetPDLatest {
				pds, err = cli.SearchProcessDefinitions(cmd.Context(), filter, collectOptions()...)
			} else {
				pds, err = cli.SearchProcessDefinitionsLatest(cmd.Context(), filter, collectOptions()...)
			}
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error fetching process definition by BPMN process ID %s and version %d: %w", flagGetPDBpmnProcessId, flagGetPDProcessVersion, err))
			}
			err = listProcessDefinitionsView(cmd, pds)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error rendering items view: %w", err))
			}
			log.Debug(fmt.Sprintf("fetched process definitions by filter, found: %d items", pds.Total))
		}
	},
}

func init() {
	getCmd.AddCommand(getProcessDefinitionCmd)

	fs := getProcessDefinitionCmd.Flags()
	fs.StringVarP(&flagGetPDKey, "key", "k", "", "process definition key to fetch")
	fs.StringVarP(&flagGetPDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.BoolVar(&flagGetPDLatest, "latest", false, "fetch the latest version(s) of the given BPMN process(s)")
	fs.Int32VarP(&flagGetPDProcessVersion, "pd-version", "v", 0, "process definition version")
	fs.StringVar(&flagGetPDProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.BoolVar(&flagGetPDWithStat, "stat", false, "include process definition statistics")
}

func populatePDSearchFilterOpts() process.ProcessDefinitionFilter {
	var filter process.ProcessDefinitionFilter
	if flagGetPDKey != "" {
		filter.Key = flagGetPDKey
	}
	if flagGetPDBpmnProcessId != "" {
		filter.BpmnProcessId = flagGetPDBpmnProcessId
	}
	if flagGetPDProcessVersion != 0 {
		filter.ProcessVersion = flagGetPDProcessVersion
	}
	if flagGetPDProcessVersionTag != "" {
		filter.ProcessVersionTag = flagGetPDProcessVersionTag
	}
	return filter
}
