package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var (
	flagShowConfigValidate bool
	flagShowConfigTemplate bool
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show effective configuration",
	Long:  `Show the effective configuration with sensitive values sanitized.`,
	Example: `./c8volt config show --validate
active_profile: local
apis:
    camunda_api:
        base_url: http://localhost:8080/v2
        key: camunda_api
        require_scope: false
        version: v2
    operate_api:
        base_url: http://localhost:8080
        key: operate_api
        require_scope: false
        version: ""
    tasklist_api:
        base_url: http://localhost:8080
        key: tasklist_api
        require_scope: false
        version: ""
    versioning_disable: false
app:
    backoff:
        initial_delay: 1s
        max_delay: 0s
        max_retries: 0
        multiplier: 2
        strategy: exponential
        timeout: 30m0s
    camunda_version: "8.8"
    tenant: ""
auth:
    cookie:
        base_url: ""
        password: '*****'
        username: ""
    mode: oauth2
    oauth2:
        client_id: c8volt
        client_secret: '*****'
        scopes:
            camunda_api: profile
            operate_api: profile
            tasklist_api: profile
        token_url: http://localhost:18080/auth/realms/camunda-platform/protocol/openid-connect
http:
    timeout: 30s
log:
    format: plain
    level: info
    with_request_body: false
    with_source: false

INFO configuration is valid`,
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("loading configuration: %w", err))
		}
		if !flagShowConfigTemplate {
			yCfg, err := cfg.ToSanitizedYAML()
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("marshaling configuration to YAML: %w", err))
			}
			cmd.Println(yCfg)
			if flagShowConfigValidate {
				err = cfg.Validate()
				if err != nil {
					ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("configuration is invalid:\n%w", err))
				}
				ferrors.HandleAndExitOK(log, "configuration is valid")
			}
		} else {
			cfg := config.New()
			_ = cfg.Normalize()
			yCfg, err := cfg.ToTemplateYAML()
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("marshaling configuration to YAML template: %w", err))
			}
			cmd.Println(yCfg)
		}
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)

	configShowCmd.Flags().BoolVar(&flagShowConfigValidate, "validate", false, "validate the effective configuration and exit with an error code if invalid")
	configShowCmd.Flags().BoolVar(&flagShowConfigTemplate, "template", false, "template configuration with values blanked out (copy-paste ready)")
	configShowCmd.MarkFlagsMutuallyExclusive("validate", "template")
}
