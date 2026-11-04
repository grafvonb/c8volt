// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
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
	flagAllTenants     bool
	flagHTTPTimeout    = "30s"
)

func Root() *cobra.Command { return rootCmd }

var rootCmd = &cobra.Command{
	Use:   "c8volt",
	Short: "Operate Camunda 8 workflows from the command line",
	Long: `Operate Camunda 8 with c8volt v5: inspect, repair, deploy, and clean up.

Documentation and examples target Camunda 8.10; 8.9 and 8.8 remain supported.
Set app.camunda_version to match your cluster. Start with config test-connection,
then use ops playbooks for repair, retention, purge, and runtime analysis.

Preview changes with --dry-run where supported. Use --auto-confirm for unattended
execution and --json for scripts. See capabilities --json for command support.`,
	Example: `  c8volt config test-connection
  c8volt get process-instance --state active --limit 10
  c8volt ops repair incident --key <incident-key> --dry-run`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), migrationPreRunKey{}, true))
		if hasHelpFlag(cmd) {
			return nil
		}
		if err := validateAllTenantsSelection(cmd); err != nil {
			return silenceUsageForError(cmd, err)
		}

		v, bindings, cfg, err := resolveLocalCommandConfig(cmd, flagQuiet, flagDebug)
		if err != nil {
			return silenceUsageForError(cmd, err)
		}
		tenantProvenance := tenantOverrideProvenanceFromConfig(v, bindings, cfg)
		ctx := localCommandLogContext(cmd, cfg, v, indicatorEnabled(cmd, cfg))
		ctx = tenantProvenance.ToContext(ctx)
		log, err := logging.FromContext(ctx)
		if err != nil {
			return silenceUsageForError(cmd, bootstrapLocalPrecondition(fmt.Errorf("retrieve logger from context: %w", err)))
		}
		configSource := configSourceDescription{loadedPath: v.ConfigFileUsed()}

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

// validateAllTenantsSelection rejects command-line all-tenants choices before
// configuration loading or command execution can perform side effects.
func validateAllTenantsSelection(cmd *cobra.Command) error {
	if cmd == nil || !flagAllTenants {
		return nil
	}
	if tenantFlag := cmd.Flags().Lookup("tenant"); tenantFlag != nil && tenantFlag.Changed {
		return mutuallyExclusiveFlagsf("--tenant cannot be combined with --all-tenants")
	}
	if allTenantsSupportForCommand(cmd) == AllTenantsSupportRejectedConcreteDestination {
		return invalidFlagValuef("--all-tenants cannot be used with %s; this command requires a concrete destination tenant", validationCommandPath(cmd))
	}
	return nil
}

// validationCommandPath formats a command path without depending on the root
// singleton during root command initialization.
func validationCommandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	path := cmd.CommandPath()
	root := cmd.Root()
	if root == nil || root.Name() == "" {
		return path
	}
	return strings.TrimSpace(strings.TrimPrefix(path, root.Name()))
}

// Execute preserves the selected leaf for locally prepared migration failures.
func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"--help"}
	}
	leaf, err := executeCommandWithMigrationContext(rootCmd, args)
	if err != nil {
		if _, prepared := err.(*migrationReportingError); prepared {
			handleBootstrapError(leaf, err)
		} else {
			handleBootstrapError(rootCmd, err)
		}
	}
}

// init registers root flags and command metadata once for the singleton tree.
func init() {
	useInvalidInputFlagErrors(rootCmd)

	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "suppress output except errors")
	pf.BoolVar(&flagCmdAutomation, "automation", false, "enable non-interactive mode for commands that explicitly support it")
	pf.BoolVarP(&flagCmdAutoConfirm, "auto-confirm", "y", false, "auto-confirm prompts for non-interactive use")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "show additional output")
	pf.BoolVar(&flagNoIndicator, "no-indicator", false, "disable transient terminal activity indicators")
	pf.BoolVar(&flagNoVersionCheck, "no-version-check", false, "skip the automatic advisory gateway version check")
	pf.BoolVar(&flagDebug, "debug", false, "enable debug logging")
	pf.BoolVarP(&flagViewAsJson, "json", "j", false, "output as JSON (where applicable)")
	pf.BoolVar(&flagViewKeysOnly, "keys-only", false, "output keys only (where applicable)")

	pf.String("config", "", "path to config file")
	pf.String("profile", "", "config active profile name to use (e.g. dev, prod)")
	pf.Var(toolx.NewDurationStringValue("30s", &flagHTTPTimeout), "timeout", "HTTP request timeout")
	pf.Float64("request-rate", 0, "maximum shared request starts per second (fractions allowed; 0 disables pacing)")

	pf.String("log-level", "info", "log level (debug, info, warn, error)")
	pf.String("log-format", "plain-time", "log format (plain-time, plain, json, text)")
	pf.Bool("log-with-source", false, "include source file and line number in logs")

	pf.String("tenant", "", "tenant for discovery and creation; explicit keys use backend authorization")
	pf.BoolVar(&flagAllTenants, "all-tenants", false, "search all accessible tenants; cannot combine with --tenant")
	pf.BoolVar(&flagNoErrCodes, "no-err-codes", false, "suppress error codes in error outputs")

	pf.String("camunda-version", string(toolx.CurrentCamundaVersion), fmt.Sprintf("Camunda version (%s) expected; aliases include 810, v810, and v8.10 for 8.10. Causes usage of specific API versions.", toolx.SupportedCamundaVersionsString()))
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
