package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth"
	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagViewAsJson     bool
	flagViewKeysOnly   bool
	flagViewAsTree     bool
	flagQuiet          bool
	flagVerbose        bool
	flagDebug          bool
	flagNoErrCodes     bool
	flagCmdAutoConfirm bool
)

var rootCmd = &cobra.Command{
	Use:   "c8volt",
	Short: "c8volt is a CLI tool to interact with Camunda 8",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		v := viper.New()
		if err := initViper(v, cmd); err != nil {
			return err
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
		cfg, err := retrieveAndNormalizeConfig(v)
		if err != nil {
			return err
		}
		ctx := cfg.ToContext(cmd.Context())
		log, err := logging.FromContext(ctx)
		if err != nil {
			return fmt.Errorf("retrieve logger from context: %w", err)
		}

		if pathcfg := v.ConfigFileUsed(); pathcfg != "" {
			log.Debug("config loaded: " + pathcfg)
		} else {
			log.Debug("no config file loaded, using defaults and environment variables")
			var configKeys = []string{
				"app.camunda_version",
				"apis.camunda_api.base_url",
				"auth.mode",
			}
			hasFlags := hasUserFlags(cmd)
			hasEnv := hasEnvConfigByKeys(configKeys)
			if !hasFlags && !hasEnv {
				log.Warn("no configuration found (no flags, environment variables, or config file); c8volt cannot run properly without configuration; run 'c8volt config show' and use the output to create a config.yaml file")
			}
		}
		if isUtilityCommand(cmd) {
			cmd.SetContext(ctx)
			return nil
		}

		if err = cfg.Validate(); err != nil {
			return fmt.Errorf("validate config:\n%w", err)
		}
		if cfg.ActiveProfile != "" {
			log.Debug("using configuration profile: " + cfg.ActiveProfile)
		} else {
			log.Debug("no active profile provided in configuration, using default settings")
		}
		log.Debug("working with Camunda version: " + string(cfg.App.CamundaVersion))
		log.Debug("using tenant ID: " + cfg.App.ViewTenant())

		httpSvc, err := httpc.New(cfg, log, httpc.WithCookieJar())
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("create http service: %w", err))
		}
		ator, err := auth.BuildAuthenticator(cfg, httpSvc.Client(), log)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("create authenticator: %w", err))
		}
		if err := ator.Init(ctx); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("initialize authenticator: %w", err))
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

func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	if (len(os.Args)) == 1 {
		rootCmd.SetArgs([]string{"--help"})
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "suppress all output, except errors, overrides --log-level")
	pf.BoolVarP(&flagCmdAutoConfirm, "auto-confirm", "y", false, "auto-confirm prompts for non-interactive use")
	pf.BoolVar(&flagVerbose, "verbose", false, "enable verbose output")
	_ = rootCmd.PersistentFlags().MarkHidden("verbose") // not used currently
	pf.BoolVar(&flagDebug, "debug", false, "enable debug logging, overwrites and is shorthand for --log-level=debug")
	pf.BoolVarP(&flagViewAsJson, "json", "j", false, "output as JSON (where applicable)")
	pf.BoolVar(&flagViewKeysOnly, "keys-only", false, "output as keys only (where applicable), can be used for piping to other commands, like cancel or delete")

	pf.String("config", "", "path to config file")
	pf.String("profile", "", "config active profile name to use (e.g. dev, prod)")

	pf.String("log-level", "info", "log level (debug, info, warn, error)")
	pf.String("log-format", "plain", "log format (json, plain, text)")
	pf.Bool("log-with-source", false, "include source file and line number in logs")

	pf.String("tenant", "", "default tenant ID")
	pf.BoolVar(&flagNoErrCodes, "no-err-codes", false, "suppress error codes in error outputs")

	pf.String("camunda-version", string(toolx.CurrentCamundaVersion), fmt.Sprintf("Camunda version (%s) expected. Causes usage of specific API versions.", toolx.SupportedCamundaVersionsString()))
	_ = rootCmd.PersistentFlags().MarkHidden("camunda-version") // not used currently
}

func initViper(v *viper.Viper, cmd *cobra.Command) error {
	fs := cmd.Flags()

	_ = v.BindPFlag("config", fs.Lookup("config"))
	_ = v.BindPFlag("active_profile", fs.Lookup("profile"))

	_ = v.BindPFlag("log.level", fs.Lookup("log-level"))
	_ = v.BindPFlag("log.format", fs.Lookup("log-format"))
	_ = v.BindPFlag("log.with_source", fs.Lookup("log-with-source"))

	_ = v.BindPFlag("app.tenant", fs.Lookup("tenant"))
	_ = v.BindPFlag("app.camunda_version", fs.Lookup("camunda-version"))
	_ = v.BindPFlag("app.no_err_codes", fs.Lookup("no-err-codes"))
	_ = v.BindPFlag("app.auto-confirm", fs.Lookup("auto-confirm"))

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "plain")
	v.SetDefault("log.with_source", false)
	v.SetDefault("log.with_request_body", false)

	v.SetEnvPrefix("c8volt")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
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
		var nf viper.ConfigFileNotFoundError
		if !errors.As(err, &nf) || v.GetString("config") != "" {
			return fmt.Errorf("read config file: %w", err)
		}
	}
	return nil
}

func retrieveAndNormalizeConfig(v *viper.Viper) (*config.Config, error) {
	var base config.Config
	if err := v.Unmarshal(&base); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg, err := base.WithProfile()
	if err != nil {
		return nil, fmt.Errorf("apply profile: %w", err)
	}
	if err := cfg.Normalize(); err != nil {
		return nil, fmt.Errorf("normalize config: %w", err)
	}
	return cfg, nil
}

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
