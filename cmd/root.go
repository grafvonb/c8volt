// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagViewAsJson     bool
	flagViewKeysOnly   bool
	flagQuiet          bool
	flagVerbose        bool
	flagDebug          bool
	flagNoIndicator    bool
	flagNoErrCodes     bool
	flagCmdAutomation  bool
	flagCmdAutoConfirm bool
	flagHTTPTimeout    = "30s"
)

func Root() *cobra.Command { return rootCmd }

var rootCmd = &cobra.Command{
	Use:   "c8volt",
	Short: "Operate Camunda 8 workflows from the command line",
	Long: `c8volt: Camunda 8 Operations CLI.

Deploy BPMN models, start process instances, inspect workflow state, wait for
state changes, walk process trees, cancel, and delete.

Supports Camunda 8.7, 8.8, 8.9, and 8.10.
Camunda 8.10 baseline: 8.10.0-alpha4 (prerelease).
Use capabilities for the machine-readable command contract.`,
	Example: `  ./c8volt config show --template
  ./c8volt --config ./config.yaml config show --validate
  ./c8volt get cluster topology
  ./c8volt embed deploy --all --run
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id>
  ./c8volt capabilities --json
  ./c8volt get --help`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		v := viper.New()
		bindings, err := initViper(v, cmd)
		if err != nil {
			return silenceUsageForError(cmd, bootstrapLocalPrecondition(err))
		}
		if hasHelpFlag(cmd) {
			return nil
		}

		switch {
		case flagQuiet:
			v.Set("log.level", "error")
		case flagDebug:
			v.Set("log.level", "debug")
		}
		cfg, err := retrieveAndNormalizeConfig(v, bindings)
		if err != nil {
			if errors.Is(err, config.ErrProfileNotFound) {
				return silenceUsageForError(cmd, normalizeBootstrapError(err))
			}
			return silenceUsageForError(cmd, bootstrapLocalPrecondition(err))
		}
		root := cmd.Root()
		activityWriter := logging.NewActivityWriterEnabled(root.ErrOrStderr(), indicatorEnabled(cmd, cfg))
		root.SetErr(activityWriter)
		cmd.SetErr(activityWriter)
		ctx := cfg.ToContextWithLogWriter(cmd.Context(), activityWriter)
		ctx = logging.ToActivityContext(ctx, activityWriter)
		log, err := logging.FromContext(ctx)
		if err != nil {
			return silenceUsageForError(cmd, bootstrapLocalPrecondition(fmt.Errorf("retrieve logger from context: %w", err)))
		}
		configSource := configSourceDescription{loadedPath: v.ConfigFileUsed()}
		ctx = configSource.ToContext(ctx)

		if configSource.loadedPath != "" {
			log.Debug(configSource.InfoMessage())
		} else {
			log.Debug(configSource.InfoMessage())
			var configKeys = []string{
				"app.camunda_version",
				"app.process_instance_page_size",
				"apis.camunda_api.base_url",
				"auth.mode",
			}
			hasEnv := hasEnvConfigByKeys(configKeys)
			if !hasEnv && !bypassRootBootstrap(cmd) {
				log.Warn(missingConfigHint())
			}
		}
		if bypassRootBootstrap(cmd) {
			cmd.SetContext(ctx)
			return nil
		}
		for _, warning := range cfg.Warnings() {
			log.Warn(warning)
		}

		if err = cfg.Validate(); err != nil {
			return silenceUsageForError(cmd, bootstrapLocalPrecondition(config.FormatValidationError("configuration is invalid", err)))
		}
		if cfg.ActiveProfile != "" {
			log.Debug("config profile " + cfg.ActiveProfile)
		} else {
			log.Debug("config profile default")
		}
		log.Debug("camunda version " + string(cfg.App.CamundaVersion))
		log.Debug("tenant " + cfg.App.ViewTenant())

		ctx, err = installRemoteCommandServices(ctx, cfg, log)
		if err != nil {
			return silenceUsageForError(cmd, err)
		}
		cmd.SetContext(ctx)

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SilenceUsage:  false,
	SilenceErrors: true,
}

func silenceUsageForError(cmd *cobra.Command, err error) error {
	if cmd != nil && err != nil {
		cmd.SilenceUsage = true
	}
	return err
}

func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	if (len(os.Args)) == 1 {
		rootCmd.SetArgs([]string{"--help"})
	}
	if err := rootCmd.Execute(); err != nil {
		handleBootstrapError(rootCmd, err)
	}
}

func init() {
	useInvalidInputFlagErrors(rootCmd)

	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "suppress output except errors")
	pf.BoolVar(&flagCmdAutomation, "automation", false, "enable non-interactive mode for commands that explicitly support it")
	pf.BoolVarP(&flagCmdAutoConfirm, "auto-confirm", "y", false, "auto-confirm prompts for non-interactive use")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "show additional output")
	pf.BoolVar(&flagNoIndicator, "no-indicator", false, "disable transient terminal activity indicators")
	pf.BoolVar(&flagDebug, "debug", false, "enable debug logging")
	pf.BoolVarP(&flagViewAsJson, "json", "j", false, "output as JSON (where applicable)")
	pf.BoolVar(&flagViewKeysOnly, "keys-only", false, "output keys only (where applicable)")

	pf.String("config", "", "path to config file")
	pf.String("profile", "", "config active profile name to use (e.g. dev, prod)")
	pf.Var(toolx.NewDurationStringValue("30s", &flagHTTPTimeout), "timeout", "HTTP request timeout")

	pf.String("log-level", "info", "log level (debug, info, warn, error)")
	pf.String("log-format", "plain-time", "log format (plain-time, plain, json, text)")
	pf.Bool("log-with-source", false, "include source file and line number in logs")

	pf.String("tenant", "", "tenant ID for discovery/search, selection, create, deploy, and run flows; explicit keys/IDs remain backend-authorized")
	pf.BoolVar(&flagNoErrCodes, "no-err-codes", false, "suppress error codes in error outputs")

	pf.String("camunda-version", string(toolx.CurrentCamundaVersion), fmt.Sprintf("Camunda version (%s) expected. Causes usage of specific API versions.", toolx.SupportedCamundaVersionsString()))
	_ = rootCmd.PersistentFlags().MarkHidden("camunda-version") // not used currently
	_ = rootCmd.PersistentFlags().MarkHidden("log-format")
	_ = rootCmd.PersistentFlags().MarkHidden("log-with-source")
	_ = rootCmd.PersistentFlags().MarkHidden("no-err-codes")

	setCapabilityDocumentVersion(rootCmd, defaultContractVersion)
	setCommandMutation(rootCmd, CommandMutationReadOnly)
	setContractSupport(rootCmd, ContractSupportLimited)
}

//nolint:unused
func hasUserFlags(cmd *cobra.Command) bool {
	if cmd.Flags().NFlag() > 0 {
		return true
	}
	if cmd.InheritedFlags().NFlag() > 0 {
		return true
	}
	return false
}
