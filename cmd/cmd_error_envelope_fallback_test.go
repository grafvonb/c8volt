// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const commandErrorEnvelopeFallbackHelper = "TestCommandErrorEnvelopeFallbackHelper"

// TestCommandErrorEnvelopeFallback verifies JSON selection does not broaden
// limited or unsupported contracts while full-contract errors remain singular.
func TestCommandErrorEnvelopeFallback(t *testing.T) {
	var requests atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		http.Error(w, "connection rejected", http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	cfgPath := testx.WriteTestConfig(t, server.URL)

	tests := []struct {
		name         string
		scenario     string
		wantExitCode int
		wantMessage  string
		wantEnvelope *commandErrorEnvelopeExpectation
		wantRequests int32
	}{
		{
			name:         "limited-fixture-shared-stdin",
			scenario:     "fixture-limited",
			wantExitCode: exitcode.InvalidArgs,
			wantMessage:  "invalid input: invalid flag value: validating keys from stdin failed: use --keys-only flag to get only keys as input",
		},
		{
			name:         "unsupported-fixture-shared-stdin",
			scenario:     "fixture-unsupported",
			wantExitCode: exitcode.InvalidArgs,
			wantMessage:  "invalid input: invalid flag value: validating keys from stdin failed: use --keys-only flag to get only keys as input",
		},
		{
			name:         "real-config-command",
			scenario:     "config-test-connection",
			wantExitCode: exitcode.InvalidArgs,
			wantMessage:  "config test-connection",
			wantRequests: 1,
		},
		{
			name:         "real-embed-command",
			scenario:     "embed-export",
			wantExitCode: exitcode.InvalidArgs,
			wantMessage:  "invalid input: missing dependent flags: either --all or at least one --file is required",
		},
		{
			name:         "existing-full-contract-error",
			scenario:     "get-incident",
			wantExitCode: exitcode.InvalidArgs,
			wantEnvelope: &commandErrorEnvelopeExpectation{
				Outcome: "invalid",
				Class:   "invalid_input",
				Command: "get incident",
				Message: "invalid input: invalid flag value: --workers must be positive integer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, suppressExit := range []bool{false, true} {
				name := "classified-exit"
				wantExitCode := tt.wantExitCode
				if suppressExit {
					name = "suppressed-exit"
					wantExitCode = 0
				}
				t.Run(name, func(t *testing.T) {
					before := requests.Load()
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, commandErrorEnvelopeFallbackHelper, "", map[string]string{
						"C8VOLT_TEST_CONFIG":       cfgPath,
						"C8VOLT_TEST_FALLBACK":     tt.scenario,
						"C8VOLT_TEST_NO_ERR_CODES": boolEnv(suppressExit),
					}, "filter: state=ACTIVE\n", wantExitCode)

					if tt.wantEnvelope != nil {
						assertCommandErrorEnvelope(t, stdout, stderr, *tt.wantEnvelope)
					} else {
						assertHumanCommandError(t, stdout, stderr, tt.wantMessage)
						require.NotContains(t, stderr, `"outcome"`, "non-full commands must retain ordinary error output")
					}
					require.Equal(t, before+tt.wantRequests, requests.Load(), "fallback selection must not add or bypass command work")
				})
			}
		})
	}
}

// TestCommandErrorEnvelopeFallbackHelper exercises contract fixtures and real
// commands in isolated processes because every failure path terminates directly.
func TestCommandErrorEnvelopeFallbackHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, commandErrorEnvelopeFallbackHelper, os.Getenv(testx.CmdSubprocessNameEnv))

	noErrCodes := os.Getenv("C8VOLT_TEST_NO_ERR_CODES") == "1"
	scenario := os.Getenv("C8VOLT_TEST_FALLBACK")
	if scenario == "fixture-limited" || scenario == "fixture-unsupported" {
		parent := &cobra.Command{Use: "fixture"}
		cmd := &cobra.Command{Use: "stdin"}
		parent.AddCommand(cmd)
		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)
		if scenario == "fixture-limited" {
			setContractSupport(cmd, ContractSupportLimited)
		} else {
			setContractSupport(cmd, ContractSupportUnsupported)
		}
		flagViewAsJson = true
		cfg := config.New()
		cfg.App.NoErrCodes = noErrCodes
		log := slog.New(logging.NewPlainHandler(os.Stderr, slog.LevelInfo))
		mergeAndValidateKeys(cmd, nil, []string{"filter: state=ACTIVE"}, log, cfg)
		return
	}

	var commandArgs []string
	switch scenario {
	case "config-test-connection":
		commandArgs = []string{"config", "test-connection"}
	case "embed-export":
		commandArgs = []string{"embed", "export"}
	case "get-incident":
		commandArgs = []string{"get", "incident", "--workers", "0"}
	default:
		t.Fatalf("unknown fallback scenario %q", scenario)
	}

	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json"}
	if noErrCodes {
		args = append(args, "--no-err-codes")
	}
	os.Args = append(args, commandArgs...)
	Execute()
}
