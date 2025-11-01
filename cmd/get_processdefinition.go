package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

const maxPDSearchSize int32 = 1000

var (
	flagGetPDKey                 string
	flagGetPDBpmnProcessId       string
	flagGetPDBpmnProcessIdLatest bool
	flagGetPDProcessVersion      int32
	flagGetPDProcessVersionTag   string
)

var getProcessDefinitionCmd = &cobra.Command{
	Use:     "process-definition",
	Short:   "Get deployed process definitions",
	Aliases: []string{"pd", "pds"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, _, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}

		log.Debug("fetching process definitions")
		searchFilterOpts := populatePDSearchFilterOpts()
		if searchFilterOpts.Key != "" {
			log.Debug(fmt.Sprintf("searching by key: %s", searchFilterOpts.Key))
			pd, err := cli.GetProcessDefinitionByKey(cmd.Context(), searchFilterOpts.Key)
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("error fetching process definition by key %s: %w", searchFilterOpts.Key, err))
			}
			err = processDefinitionView(cmd, pd)
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("error rendering key-only view: %w", err))
			}
		} else {
			if flagGetPDProcessVersionTag == "" {
				if flagGetPDProcessVersion != 0 && flagGetPDBpmnProcessId != "" {
					log.Debug(fmt.Sprintf("searching by BPMN process ID: %s and version: %d", flagGetPDBpmnProcessId, flagGetPDProcessVersion))
					pd, err := cli.GetProcessDefinitionByBpmnProcessIdAndVersion(cmd.Context(), flagGetPDBpmnProcessId, flagGetPDProcessVersion)
					if err != nil {
						ferrors.HandleAndExit(log, fmt.Errorf("error fetching process definition by BPMN process ID %s and version %d: %w", flagGetPDBpmnProcessId, flagGetPDProcessVersion, err))
					}
					err = processDefinitionView(cmd, pd)
					if err != nil {
						ferrors.HandleAndExit(log, fmt.Errorf("error rendering BPMN process ID and version view: %w", err))
					}
					log.Debug(fmt.Sprintf("searched by BPMN process ID and version, found process definition with key: %s", pd.Key))
					return
				} else if flagGetPDProcessVersionTag != "" {
					ferrors.HandleAndExit(log, fmt.Errorf("%w: fetching by process version tag requires BPMN process ID to be set", ferrors.ErrBadRequest))
				}
				if flagGetPDBpmnProcessId != "" && flagGetPDBpmnProcessIdLatest {
					log.Debug(fmt.Sprintf("searching by BPMN process ID: %s and latest version", flagGetPDBpmnProcessId))
					pd, err := cli.GetProcessDefinitionByBpmnProcessIdLatest(cmd.Context(), flagGetPDBpmnProcessId)
					if err != nil {
						ferrors.HandleAndExit(log, fmt.Errorf("error fetching process definition by BPMN process ID %s and latest version: %w", flagGetPDBpmnProcessId, err))
					}
					err = processDefinitionView(cmd, pd)
					if err != nil {
						ferrors.HandleAndExit(log, fmt.Errorf("error rendering BPMN process ID latest version view: %w", err))
					}
					log.Debug(fmt.Sprintf("searched by BPMN process ID and latest version, found process definition with key: %s", pd.Key))
					return
				} else if flagGetPDBpmnProcessIdLatest {
					log.Debug("searching latest versions of process definitions")
					pds, err := cli.GetProcessDefinitionsLatest(cmd.Context())
					if err != nil {
						ferrors.HandleAndExit(log, fmt.Errorf("error fetching latest process definitions: %w", err))
					}
					err = listProcessDefinitionsView(cmd, pds)
					if err != nil {
						ferrors.HandleAndExit(log, fmt.Errorf("error rendering items view: %w", err))
					}
					log.Debug(fmt.Sprintf("fetched latest process definitions, found: %d items", pds.Total))
					return
				}
			}
			log.Debug(fmt.Sprintf("searching by filter: %v", searchFilterOpts))
			pds, err := cli.SearchProcessDefinitions(cmd.Context(), searchFilterOpts, maxPDSearchSize)
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("error fetching process definitions: %w", err))
			}
			err = listProcessDefinitionsView(cmd, pds)
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("error rendering items view: %w", err))
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
	fs.BoolVar(&flagGetPDBpmnProcessIdLatest, "latest", false, "fetch the latest version of the given BPMN process(s)")
	fs.Int32VarP(&flagGetPDProcessVersion, "process-version", "v", 0, "process definition version")
	fs.StringVar(&flagGetPDProcessVersionTag, "process-version-tag", "", "process definition version tag")
}

func populatePDSearchFilterOpts() process.ProcessDefinitionSearchFilterOpts {
	var filter process.ProcessDefinitionSearchFilterOpts
	if flagGetPDKey != "" {
		filter.Key = flagGetPDKey
	}
	if flagGetPDBpmnProcessId != "" {
		filter.BpmnProcessId = flagGetPDBpmnProcessId
	}
	if flagGetPDProcessVersion != 0 {
		filter.Version = flagGetPDProcessVersion
	}
	if flagGetPDProcessVersionTag != "" {
		filter.VersionTag = flagGetPDProcessVersionTag
	}
	return filter
}
