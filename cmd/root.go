package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/services/auth"
	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	flagViewAsJson        bool
	flagViewKeysOnly      bool
	flagViewAsTree        bool
	flagQuiet             bool
	flagVerbose           bool
	flagDebug             bool
	flagNoIndicator       bool
	flagNoErrCodes        bool
	flagCmdAutomation     bool
	flagCmdAutoConfirm    bool
	flagAllowInconsistent bool
	flagHTTPTimeout       = "30s"
)

func Root() *cobra.Command { return rootCmd }

type resolverBindings struct {
	flags map[string]*pflag.Flag
}

func newResolverBindings() *resolverBindings {
	return &resolverBindings{
		flags: make(map[string]*pflag.Flag),
	}
}

func (r *resolverBindings) bindPFlag(v *viper.Viper, key string, flag *pflag.Flag) {
	if flag == nil {
		return
	}
	_ = v.BindPFlag(key, flag)
	r.flags[key] = flag
}

func (r *resolverBindings) hasHigherPrecedenceSource(key string) bool {
	if flag, ok := r.flags[key]; ok && flag != nil && flag.Changed {
		return true
	}
	return hasEnvConfigByKeys([]string{key})
}

var rootCmd = &cobra.Command{
	Use:   "c8volt",
	Short: "Operate Camunda 8 with guided help and script-safe output modes",
	Long: `c8volt: Camunda 8 Operations CLI.

Built for Camunda 8 operators and developers who need confirmation, not guesses.
c8volt focuses on operational workflows such as deploying BPMN models, starting process instances,
waiting for state transitions, walking process trees, cancelling safely, and deleting thoroughly.

Start with "c8volt <group> --help" when choosing an operator workflow, or use
"c8volt capabilities --json" when a script, CI job, or AI caller needs the public command inventory,
flag metadata, output modes, mutation behavior, and automation support without scraping prose help.
Human-oriented command families remain the primary interactive surface; JSON and keys-only modes layer onto
the same public Cobra tree for script-safe automation on supported commands.
Prefer --json where a command exposes structured output, and use --automation only when that command's
capabilities entry reports automation:full for the canonical non-interactive contract.

Strict single-resource lookups keep their normal not-found behavior. The newer orphan-parent warning
contract is limited to traversal and dependency-expansion flows such as walk, cancel, and delete when
actionable process-instance data was still resolved.

Tenant-aware process-instance flows use one effective tenant context per command execution.
Supported wrong-tenant lookups resolve as not found. Current process-instance runtime support
is implemented for Camunda 8.7, 8.8, and 8.9 through the repository's versioned service
factories and facades, with the same repository command-family coverage on 8.9 that already
exists on 8.8.

Commands that create tenant-owned data, such as deploy and run, target <default> when the
effective tenant is empty. Read/search commands preserve an empty tenant as an unscoped
visible-tenants query unless --tenant is provided.

Refer to the documentation at https://c8volt.info for more information.`,
	Example: `  ./c8volt get --help
  ./c8volt run process-instance --help
  ./c8volt capabilities --json
  ./c8volt --config ./config.yaml config show --validate`,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		v := viper.New()
		bindings, err := initViper(v, cmd)
		if err != nil {
			return bootstrapLocalPrecondition(err)
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
				return normalizeBootstrapError(err)
			}
			return bootstrapLocalPrecondition(err)
		}
		activityWriter := logging.NewActivityWriterEnabled(cmd.ErrOrStderr(), indicatorEnabled(cmd, cfg))
		ctx := cfg.ToContextWithLogWriter(cmd.Context(), activityWriter)
		ctx = logging.ToActivityContext(ctx, activityWriter)
		log, err := logging.FromContext(ctx)
		if err != nil {
			return bootstrapLocalPrecondition(fmt.Errorf("retrieve logger from context: %w", err))
		}

		if pathcfg := v.ConfigFileUsed(); pathcfg != "" {
			log.Debug("config loaded: " + pathcfg)
		} else {
			log.Debug("no config file loaded, using defaults and environment variables")
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
			return bootstrapLocalPrecondition(config.FormatValidationError("configuration is invalid", err))
		}
		if cfg.ActiveProfile != "" {
			log.Debug("using configuration profile: " + cfg.ActiveProfile)
		} else {
			log.Debug("no active profile provided in configuration, using default settings")
		}
		log.Debug("working with Camunda version: " + string(cfg.App.CamundaVersion))
		log.Debug("using tenant ID: " + cfg.App.ViewTenant())

		httpSvc, err := httpc.New(cfg, log, httpc.WithCookieJar(), httpc.WithActivitySink(activityWriter))
		if err != nil {
			return bootstrapLocalPrecondition(fmt.Errorf("create http service: %w", err))
		}
		ator, err := auth.BuildAuthenticator(cfg, httpSvc.Client(), log)
		if err != nil {
			return bootstrapLocalPrecondition(fmt.Errorf("create authenticator: %w", err))
		}
		if err := ator.Init(ctx); err != nil {
			return normalizeBootstrapError(fmt.Errorf("initialize authenticator: %w", err))
		}
		httpSvc.InstallAuthEditor(ator.Editor())
		ctx = httpSvc.ToContext(ctx)
		ctx = authenticator.ToContext(ctx, ator)
		cmd.SetContext(ctx)

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SilenceUsage:  false,
	SilenceErrors: false,
}

func missingConfigHint() string {
	const exampleName = "config.example.yaml"
	if wd, err := os.Getwd(); err == nil {
		examplePath := filepath.Join(wd, exampleName)
		if info, statErr := os.Stat(examplePath); statErr == nil && !info.IsDir() {
			return fmt.Sprintf("no configuration found (environment variables, or config file); c8volt cannot run properly without configuration; found %q in the current directory, copy or edit it into a local config.yaml and run 'c8volt --config ./config.yaml config show --validate'", exampleName)
		}
	}
	return "no configuration found (environment variables, or config file); c8volt cannot run properly without configuration; run 'c8volt config show --template' and use the output to create a config.yaml file"
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
	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "suppress all output, except errors, overrides --log-level")
	pf.BoolVar(&flagCmdAutomation, "automation", false, "enable the canonical non-interactive contract for commands that explicitly support it")
	pf.BoolVarP(&flagCmdAutoConfirm, "auto-confirm", "y", false, "auto-confirm prompts for non-interactive use")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "adds additional verbosity to the output, e.g. for progress indication")
	pf.BoolVar(&flagNoIndicator, "no-indicator", false, "disable transient terminal activity indicators")
	pf.BoolVar(&flagDebug, "debug", false, "enable debug logging, overwrites and is shorthand for --log-level=debug")
	pf.BoolVarP(&flagViewAsJson, "json", "j", false, "output as JSON (where applicable)")
	pf.BoolVar(&flagViewKeysOnly, "keys-only", false, "output as keys only (where applicable), can be used for piping to other commands")

	pf.String("config", "", "path to config file")
	pf.String("profile", "", "config active profile name to use (e.g. dev, prod)")
	pf.Var(toolx.NewDurationStringValue("30s", &flagHTTPTimeout), "timeout", "HTTP request timeout")

	pf.String("log-level", "info", "log level (debug, info, warn, error)")
	pf.String("log-format", "plain", "log format (json, plain, text)")
	pf.Bool("log-with-source", false, "include source file and line number in logs")

	pf.String("tenant", "", "tenant ID for tenant-aware command flows (overrides env, profile, and base config)")
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

func initViper(v *viper.Viper, cmd *cobra.Command) (*resolverBindings, error) {
	fs := cmd.Flags()
	bindings := newResolverBindings()

	bindings.bindPFlag(v, "config", fs.Lookup("config"))
	bindings.bindPFlag(v, "active_profile", fs.Lookup("profile"))
	bindings.bindPFlag(v, "http.timeout", fs.Lookup("timeout"))

	bindings.bindPFlag(v, "log.level", fs.Lookup("log-level"))
	bindings.bindPFlag(v, "log.format", fs.Lookup("log-format"))
	bindings.bindPFlag(v, "log.with_source", fs.Lookup("log-with-source"))

	bindings.bindPFlag(v, "app.tenant", fs.Lookup("tenant"))
	bindings.bindPFlag(v, "app.camunda_version", fs.Lookup("camunda-version"))
	bindings.bindPFlag(v, "app.automation", fs.Lookup("automation"))
	bindings.bindPFlag(v, "app.no_err_codes", fs.Lookup("no-err-codes"))
	bindings.bindPFlag(v, "app.auto-confirm", fs.Lookup("auto-confirm"))
	bindCommandLocalConfigFlags(v, bindings, fs)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "plain")
	v.SetDefault("log.with_source", false)
	v.SetDefault("log.with_request_body", false)
	v.SetDefault("http.timeout", "30s")
	v.SetDefault("app.process_instance_page_size", consts.MaxPISearchSize)

	v.SetEnvPrefix("c8volt")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Config file resolution and read
	if p := v.GetString("config"); p != "" {
		v.SetConfigFile(p)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$XDG_CONFIG_HOME/c8volt")
		v.AddConfigPath("$HOME/.config/c8volt")
		v.AddConfigPath("$HOME/.c8volt")
		v.AddConfigPath("/etc/c8volt")
	}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok || v.GetString("config") != "" {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}
	return bindings, nil
}

func bindCommandLocalConfigFlags(v *viper.Viper, bindings *resolverBindings, fs *pflag.FlagSet) {
	bindings.bindPFlag(v, "app.backoff.timeout", fs.Lookup("backoff-timeout"))
	bindings.bindPFlag(v, "app.backoff.max_retries", fs.Lookup("backoff-max-retries"))

	if fs.Lookup("backoff-timeout") == nil {
		return
	}
	v.SetDefault("app.backoff.timeout", defaultBackoffTimeout)
	v.SetDefault("app.backoff.max_retries", defaultBackoffMaxRetries)
	v.SetDefault("app.backoff.strategy", defaultBackoffStrategy)
	v.SetDefault("app.backoff.initial_delay", defaultBackoffInitialDelay)
	v.SetDefault("app.backoff.max_delay", defaultBackoffMaxDelay)
	v.SetDefault("app.backoff.multiplier", defaultBackoffMultiplier)
}

func automationModeEnabled(cmd *cobra.Command) bool {
	if cmd != nil {
		if cfg, err := config.FromContext(cmd.Context()); err == nil && cfg != nil {
			return cfg.App.Automation
		}
		if flag := cmd.Flags().Lookup("automation"); flag != nil {
			if value, err := strconv.ParseBool(flag.Value.String()); err == nil {
				return value
			}
		}
	}
	return flagCmdAutomation
}

func indicatorEnabled(cmd *cobra.Command, cfg *config.Config) bool {
	if flagNoIndicator || flagQuiet {
		return false
	}
	if cfg != nil {
		if strings.EqualFold(cfg.Log.Format, "json") {
			return false
		}
		return !cfg.App.Automation
	}
	return !automationModeEnabled(cmd)
}

func retrieveAndNormalizeConfig(v *viper.Viper, bindings *resolverBindings) (*config.Config, error) {
	cfg, err := config.ResolveEffectiveConfig(
		v,
		bindings.hasHigherPrecedenceSource,
		func(activeProfile, key string) bool {
			if activeProfile == "" {
				return false
			}
			return v.InConfig("profiles." + activeProfile + "." + key)
		},
	)
	if err != nil {
		return nil, err
	}
	return cfg, nil
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

func envNameForKey(key string) string {
	key = strings.ReplaceAll(key, ".", "_")
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ToUpper(key)
	return "C8VOLT_" + key
}

func hasEnvConfigByKeys(configKeys []string) bool {
	for _, key := range configKeys {
		envName := envNameForKey(key)
		if _, ok := os.LookupEnv(envName); ok {
			return true
		}
	}
	return false
}
