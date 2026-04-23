package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/c8volt"
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
	flagGetPDAsXML             bool
)

var getProcessDefinitionCmd = &cobra.Command{
	Use:   "process-definition",
	Short: "List or fetch deployed process definitions",
	Long: `List or fetch deployed process definitions.

Use this read-only command to inspect deployed BPMN models by key, BPMN process
ID, version selectors, or the latest deployed version. Default output is aimed
at human review; prefer ` + "`--json`" + ` when chaining the result into scripts or
AI-assisted workflows. Use ` + "`--xml`" + ` only when you need the raw BPMN XML for a
single definition selected by ` + "`--key`" + `. When ` + "`--stat`" + ` is enabled,
Camunda ` + "`8.8`" + `/` + "`8.9`" + ` report ` + "`ac`" + ` from native active
process-instance statistics for the exact process definition version and add
` + "`in:<count>`" + ` for active process instances with incidents; ` + "`cp`" + `
and ` + "`cx`" + ` keep their existing process-definition statistics meaning.
Camunda ` + "`8.7`" + ` rejects statistics because the generated client surface does
not provide the same native statistics endpoints.`,
	Example: `  ./c8volt get pd --latest
  ./c8volt get pd --bpmn-process-id C88_SimpleUserTask_Process --latest
  ./c8volt get pd --key 2251799813686017 --json
  ./c8volt get pd --key 2251799813686017 --xml`,
	Aliases: []string{"pd", "pds"},
	Run:     runGetProcessDefinition,
}

func runGetProcessDefinition(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}

	log.Debug("fetching process definitions")
	filter := populatePDSearchFilterOpts()
	if flagGetPDAsXML {
		runGetProcessDefinitionXML(cmd, cli, log, cfg.App.NoErrCodes, filter)
		return
	}
	if filter.Key != "" {
		runGetProcessDefinitionByKey(cmd, cli, log, cfg.App.NoErrCodes, filter.Key)
		return
	}
	runSearchProcessDefinitions(cmd, cli, log, cfg.App.NoErrCodes, filter)
}

func runGetProcessDefinitionXML(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, filter process.ProcessDefinitionFilter) {
	if err := validateProcessDefinitionXMLFlags(filter); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, err)
	}

	log.Debug(fmt.Sprintf("fetching process definition xml by key: %s", filter.Key))
	xml, err := cli.GetProcessDefinitionXML(cmd.Context(), filter.Key, collectOptions()...)
	if err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("get process definition xml: %w", err))
	}
	if _, err := io.WriteString(cmd.OutOrStdout(), xml); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error writing process definition xml: %w", err))
	}
}

func runGetProcessDefinitionByKey(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, key string) {
	log.Debug(fmt.Sprintf("searching by key: %s", key))
	pd, err := cli.GetProcessDefinition(cmd.Context(), key, collectOptions()...)
	if err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("get process definition: %w", err))
	}
	if err := processDefinitionView(cmd, pd); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error rendering key-only view: %w", err))
	}
}

func runSearchProcessDefinitions(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, filter process.ProcessDefinitionFilter) {
	log.Debug(fmt.Sprintf("searching process definitions for filter %+v", filter))

	var (
		pds process.ProcessDefinitions
		err error
	)
	if !flagGetPDLatest {
		pds, err = cli.SearchProcessDefinitions(cmd.Context(), filter, collectOptions()...)
	} else {
		pds, err = cli.SearchProcessDefinitionsLatest(cmd.Context(), filter, collectOptions()...)
	}
	if err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("search process definitions: %w", err))
	}
	if err := listProcessDefinitionsView(cmd, pds); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error rendering items view: %w", err))
	}
	log.Debug(fmt.Sprintf("fetched process definitions by filter, found: %d items", pds.Total))
}

func init() {
	getCmd.AddCommand(getProcessDefinitionCmd)

	fs := getProcessDefinitionCmd.Flags()
	fs.StringVarP(&flagGetPDKey, "key", "k", "", "process definition key to fetch")
	fs.StringVarP(&flagGetPDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.BoolVar(&flagGetPDLatest, "latest", false, "fetch the latest version(s) of the given BPMN process(s)")
	fs.Int32Var(&flagGetPDProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPDProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.BoolVar(&flagGetPDWithStat, "stat", false, "include process definition statistics; 8.8/8.9 use native active/incident instance stats, 8.7 unsupported")
	fs.BoolVar(&flagGetPDAsXML, "xml", false, "output the selected process definition as raw XML (requires --key and no other filters)")

	setCommandMutation(getProcessDefinitionCmd, CommandMutationReadOnly)
	setContractSupport(getProcessDefinitionCmd, ContractSupportLimited)
	setOutputModes(getProcessDefinitionCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
			Notes:            "preferred for automation when not using --xml",
		},
	)
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

func validateProcessDefinitionXMLFlags(filter process.ProcessDefinitionFilter) error {
	if filter.Key == "" {
		return missingDependentFlagsf("xml output requires --key to select a single process definition")
	}

	var incompatible []string
	for _, check := range []struct {
		enabled bool
		flag    string
	}{
		{enabled: filter.BpmnProcessId != "", flag: "--bpmn-process-id"},
		{enabled: flagGetPDProcessVersion != 0, flag: "--pd-version"},
		{enabled: filter.ProcessVersionTag != "", flag: "--pd-version-tag"},
		{enabled: flagGetPDLatest, flag: "--latest"},
		{enabled: flagGetPDWithStat, flag: "--stat"},
		{enabled: flagViewAsJson, flag: "--json"},
		{enabled: flagViewKeysOnly, flag: "--keys-only"},
	} {
		if check.enabled {
			incompatible = append(incompatible, check.flag)
		}
	}
	if len(incompatible) > 0 {
		return forbiddenFlagCombinationf("xml output only supports --key; incompatible with %s", strings.Join(incompatible, ", "))
	}

	return nil
}
