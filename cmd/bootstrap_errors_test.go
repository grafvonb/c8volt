// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_V89SupportMessagingIsUpdated(t *testing.T) {
	t.Parallel()

	root := Root()

	require.Contains(t, root.Long, "Camunda 8.8, 8.9, and 8.10")
	require.Contains(t, root.Long, "Camunda 8.10 baseline: 8.10.3 (final)")
	require.Contains(t, root.Long, "Default: 8.9")
	require.NotContains(t, root.Long, "version 8.9 is recognized by config normalization")
	require.Contains(t, root.PersistentFlags().Lookup("camunda-version").Usage, toolx.SupportedCamundaVersionsString())
}

// Verifies NewCli maps missing bootstrap context to the shared local-precondition error class.
func TestNewCliNormalizesMissingBootstrapContext(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	_, _, _, err := NewCli(cmd)
	require.Error(t, err)
	require.Equal(t, ferrors.ClassLocalPrecondition, ferrors.Classify(err))
}

// Verifies NewCli now constructs a v8.9-backed client instead of rejecting the version at bootstrap.
func TestNewCliConstructsSupportedV89Client(t *testing.T) {
	cfg := &config.Config{
		App: config.App{
			CamundaVersion: toolx.V89,
		},
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "http://127.0.0.1:1",
			},
		},
		HTTP: config.HTTP{
			RequestRateReportInterval: "5s",
			Timeout:                   "30s",
		},
	}
	ctx := cfg.ToContext(context.Background())

	httpSvc, err := httpc.New(cfg, nil)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.SetContext(httpSvc.ToContext(ctx))

	_, _, cli, err := NewCli(cmd)
	require.NoError(t, err)
	require.NotNil(t, cli)
}

// Verifies NewCli constructs a v8.10-backed client after every native service
// factory is wired into the complete c8volt facade.
func TestNewCliConstructsSupportedV810Client(t *testing.T) {
	cfg := &config.Config{
		App: config.App{
			CamundaVersion: toolx.V810,
		},
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "http://127.0.0.1:1",
			},
		},
		HTTP: config.HTTP{
			RequestRateReportInterval: "5s",
			Timeout:                   "30s",
		},
	}
	ctx := cfg.ToContext(context.Background())

	httpSvc, err := httpc.New(cfg, nil)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.SetContext(httpSvc.ToContext(ctx))

	_, _, cli, err := NewCli(cmd)
	require.NoError(t, err)
	require.NotNil(t, cli)
}

// Verifies execute-time config validation failures use the shared failure model and exit behavior.
func TestExecute_ConfigValidationFailureUsesSharedFailureModel(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	content := `app:
  camunda_version: "9.9"
auth:
  mode: none
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	output, err := testx.RunCmdSubprocess(t, "TestExecute_ConfigValidationFailureUsesSharedFailureModelHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed: normalize config:")
	require.NotContains(t, string(output), "Error: local precondition failed")
	require.Equal(t, 1, strings.Count(string(output), "local precondition failed: normalize config:"))
}

// Verifies v8.9 command execution now proceeds past bootstrap and surfaces downstream runtime failures normally.
func TestExecute_V89RuntimeFailuresUseSharedFailureModel(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.9")

	output, err := testx.RunCmdSubprocess(t, "TestExecute_V89RuntimeFailuresUseSharedFailureModelHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "internal error")
	require.Contains(t, string(output), "get cluster topology")
}

// Verifies an explicit --config path wins over default search-path config discovery.
func TestExecute_ConfigFlagOverridesDefaultSearchPath(t *testing.T) {
	dir := t.TempDir()
	defaultCfgPath := filepath.Join(dir, "config.yaml")
	explicitCfgPath := filepath.Join(dir, "explicit.yaml")

	defaultCfg := `app:
  tenant: default-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://default.example.test
`
	explicitCfg := `app:
  tenant: explicit-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://explicit.example.test
`
	require.NoError(t, os.WriteFile(defaultCfgPath, []byte(defaultCfg), 0o600))
	require.NoError(t, os.WriteFile(explicitCfgPath, []byte(explicitCfg), 0o600))

	output, err := testx.RunCmdSubprocessInDir(t, "TestExecute_ConfigFlagOverridesDefaultSearchPathHelper", dir, map[string]string{
		"C8VOLT_TEST_CONFIG": explicitCfgPath,
	})
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "tenant: explicit-tenant")
	require.NotContains(t, string(output), "tenant: default-tenant")
}

// Verifies all-tenants and explicit tenant conflicts are invalid input before
// missing configuration or remote bootstrap can take over the failure.
func TestExecute_AllTenantsTenantConflictUsesInvalidInputBeforeConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "named tenant",
			args: []string{"--all-tenants", "--tenant", "tenant-a", "get", "cluster", "topology"},
		},
		{
			name: "empty tenant",
			args: []string{"--all-tenants", "--tenant", "", "get", "cluster", "topology"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsJSON, err := json.Marshal(tt.args)
			require.NoError(t, err)

			output, err := testx.RunCmdSubprocess(t, "TestExecute_AllTenantsTenantConflictUsesInvalidInputBeforeConfigHelper", map[string]string{
				"C8VOLT_TEST_ARGS": string(argsJSON),
			})
			require.Error(t, err)

			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "invalid input")
			require.Contains(t, string(output), "--tenant cannot be combined with --all-tenants")
			require.NotContains(t, string(output), "local precondition failed")
			require.NotContains(t, string(output), "Usage:")
			require.NotContains(t, string(output), "Examples:")
			require.NotContains(t, string(output), "Global Flags:")
		})
	}
}

// Verifies bootstrap normalization maps command-validation sentinels to invalid-input classification.
func TestNormalizeBootstrapErrorMapsCommandValidationToInvalidInput(t *testing.T) {
	err := normalizeBootstrapError(invalidFlagValuef("resource lookup requires a non-empty --id"))

	require.Error(t, err)
	require.Equal(t, ferrors.ClassInvalidInput, ferrors.Classify(err))
}

// Verifies bootstrap normalization keeps already-normalized shared failures stable.
func TestNormalizeBootstrapErrorPreservesNormalizedFailures(t *testing.T) {
	t.Parallel()

	err := ferrors.WrapClass(ferrors.ErrLocalPrecondition, errors.New("loading configuration: missing base url"))
	got := normalizeBootstrapError(err)

	require.Equal(t, "local precondition failed: loading configuration: missing base url", got.Error())
	require.Equal(t, ferrors.ClassLocalPrecondition, ferrors.Classify(got))
	require.Equal(t, exitcode.Error, ferrors.ExitCode(got))
}

// Verifies shared command-validation sentinels normalize to the invalid-input class.
func TestNormalizeCommandErrorMapsSharedValidationSentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "invalid flag value", err: invalidFlagValuef("boom")},
		{name: "forbidden flag combination", err: forbiddenFlagCombinationf("boom")},
		{name: "missing dependent flags", err: missingDependentFlagsf("boom")},
		{name: "mutually exclusive flags", err: mutuallyExclusiveFlagsf("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, ferrors.ClassInvalidInput, ferrors.Classify(normalizeCommandError(tt.err)))
		})
	}
}

// Verifies command normalization does not restack an already-normalized shared failure.
func TestNormalizeCommandErrorPreservesNormalizedFailures(t *testing.T) {
	t.Parallel()

	err := ferrors.WrapClass(ferrors.ErrInvalidInput, errors.New("invalid flag value: boom"))
	got := normalizeCommandError(err)

	require.Equal(t, "invalid input: invalid flag value: boom", got.Error())
	require.Equal(t, ferrors.ClassInvalidInput, ferrors.Classify(got))
	require.Equal(t, exitcode.InvalidArgs, ferrors.ExitCode(got))
}

// Verifies bootstrap normalization rule-table mappings for known and fallback error cases.
func TestNormalizeBootstrapErrorMapsSharedBootstrapRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantClass ferrors.Class
	}{
		{
			name:      "command validation keeps invalid input",
			err:       ErrInvalidFlagValue,
			wantClass: ferrors.ClassInvalidInput,
		},
		{
			name:      "missing config becomes local precondition",
			err:       config.ErrNoConfigInContext,
			wantClass: ferrors.ClassLocalPrecondition,
		},
		{
			name:      "http service bootstrap becomes local precondition",
			err:       httpc.ErrNoHttpServiceInContext,
			wantClass: ferrors.ClassLocalPrecondition,
		},
		{
			name:      "missing profile becomes invalid input",
			err:       config.ErrProfileNotFound,
			wantClass: ferrors.ClassInvalidInput,
		},
		{
			name:      "unmapped errors fall back to internal",
			err:       errors.New("boom"),
			wantClass: ferrors.ClassInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.wantClass, ferrors.Classify(normalizeBootstrapError(tt.err)))
		})
	}
}

// Helper-process entrypoint for config-validation failure normalization coverage.
func TestExecute_ConfigValidationFailureUsesSharedFailureModelHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "topology"}

	Execute()
}

// Helper-process entrypoint for v8.9 runtime-failure normalization coverage.
func TestExecute_V89RuntimeFailuresUseSharedFailureModelHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "topology"}

	Execute()
}

// Helper-process entrypoint for explicit-config precedence over default search-path config.
func TestExecute_ConfigFlagOverridesDefaultSearchPathHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "config", "show"}

	Execute()
}

// Helper-process entrypoint for all-tenants/tenant conflict bootstrap coverage.
func TestExecute_AllTenantsTenantConflictUsesInvalidInputBeforeConfigHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = append([]string{"c8volt"}, args...)

	Execute()
}

// TestBootstrapMigrationLocalContext verifies recovery uses ordinary config
// precedence and isolated values, without constructing remote service context.
func TestBootstrapMigrationLocalContext(t *testing.T) {
	for _, tt := range []struct {
		name       string
		controls   []string
		level      string
		noErrCodes bool
	}{
		{name: "environment", level: "warn", noErrCodes: true},
		{name: "explicit", controls: []string{"--log-level=error", "--no-err-codes=false"}, level: "error"},
		{name: "debug", controls: []string{"--debug"}, level: "debug", noErrCodes: true},
		{name: "quiet overrides debug", controls: []string{"--debug", "--quiet"}, level: "error", noErrCodes: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("C8VOLT_LOG_LEVEL", "warn")
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte("log:\n  level: info\n  format: json\nprofiles:\n  diagnostic:\n    app:\n      no_err_codes: true\n"), 0600))
			root := &cobra.Command{Use: "test"}
			leaf := &cobra.Command{Use: "resource"}
			root.AddCommand(leaf)
			for _, name := range []string{"config", "profile", "log-level", "log-format"} {
				root.PersistentFlags().String(name, "", "")
			}
			for _, name := range []string{"quiet", "debug", "json", "no-err-codes"} {
				root.PersistentFlags().Bool(name, false, "")
			}
			setFlagMigrations(leaf, map[string]flagMigration{"old": {Replacement: "--new"}})
			leaf.SetContext(context.Background())
			original := errors.New("private parser cause")
			args := append([]string{"--old", "--config", path, "--profile=diagnostic", "--json"}, tt.controls...)
			prepared := prepareMigrationReporting(leaf, args, original)
			var report *migrationReportingError
			require.ErrorAs(t, prepared, &report)
			require.ErrorIs(t, prepared, original)
			require.Equal(t, RenderModeJSON, report.mode)
			require.Equal(t, tt.noErrCodes, report.noErrCodes)
			cfg, err := config.FromContext(leaf.Context())
			require.NoError(t, err)
			require.Equal(t, "diagnostic", cfg.ActiveProfile)
			require.Equal(t, tt.level, cfg.Log.Level)
			require.Equal(t, "json", cfg.Log.Format)
			_, err = httpc.FromContext(leaf.Context())
			require.ErrorIs(t, err, httpc.ErrNoHttpServiceInContext)
			require.False(t, root.PersistentFlags().Changed("profile"))
			require.False(t, root.PersistentFlags().Changed("json"))
			require.NotContains(t, prepared.Error(), original.Error())
		})
	}
}

// TestBootstrapMigrationConfigurationOverrides ensures local reporting honors
// explicit configuration flags even when the base configuration cannot normalize.
func TestBootstrapMigrationConfigurationOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("app:\n  camunda_version: invalid\nhttp:\n  timeout: 45s\nlog:\n  format: json\n"), 0600))
	root := &cobra.Command{Use: "test"}
	leaf := &cobra.Command{Use: "resource"}
	root.AddCommand(leaf)
	root.PersistentFlags().String("config", "", "")
	root.PersistentFlags().String("camunda-version", "8.9", "")
	root.PersistentFlags().String("timeout", "30s", "")
	root.PersistentFlags().Float64("request-rate", 0, "")
	setFlagMigrations(leaf, map[string]flagMigration{"old": {Replacement: "--new"}})
	leaf.SetContext(context.Background())
	prepared := prepareMigrationReporting(leaf, []string{"--old", "--config", path, "--camunda-version=8.10", "--timeout=10s", "--request-rate=1.5"}, errors.New("rejected"))
	require.NotContains(t, prepared.Error(), "could not be loaded")
	cfg, err := config.FromContext(leaf.Context())
	require.NoError(t, err)
	require.Equal(t, toolx.V810, cfg.App.CamundaVersion)
	require.Equal(t, "10s", cfg.HTTP.Timeout)
	require.Equal(t, 1.5, cfg.HTTP.RequestRate)
	require.False(t, root.PersistentFlags().Changed("camunda-version"))
}
