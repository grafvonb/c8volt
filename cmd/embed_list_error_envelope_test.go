// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const embedListErrorEnvelopeHelper = "TestCommandErrorEnvelopeEmbedListHelper"

// TestCommandErrorEnvelopeEmbedList verifies deterministic listing failures
// terminate with one result while real CLI success coverage retains wiring.
func TestCommandErrorEnvelopeEmbedList(t *testing.T) {
	tests := []struct {
		name    string
		failure string
		class   string
		message string
	}{
		{
			name:    "listing-error",
			failure: "error",
			class:   "internal",
			message: "internal error: list embedded files failed",
		},
		{
			name:    "empty-list",
			failure: "empty",
			class:   "local_precondition",
			message: `local precondition failed: no embedded files found for Camunda version "8.8"`,
		},
		{
			name:    "nonmatching-version",
			failure: "nonmatching",
			class:   "local_precondition",
			message: `local precondition failed: no embedded files found for Camunda version "8.8"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, outputMode := range []string{"json", "human"} {
				for _, suppressExit := range []bool{false, true} {
					name := outputMode
					if suppressExit {
						name += "/no-error-codes"
					}
					t.Run(name, func(t *testing.T) {
						wantExitCode := exitcode.Error
						if suppressExit {
							wantExitCode = 0
						}
						stdout, stderr := runCommandErrorEnvelopeSubprocess(t, embedListErrorEnvelopeHelper, "", map[string]string{
							"C8VOLT_TEST_EMBED_FAILURE": tt.failure,
							"C8VOLT_TEST_OUTPUT_MODE":   outputMode,
							"C8VOLT_TEST_NO_ERR_CODES":  boolEnv(suppressExit),
						}, "", wantExitCode)
						if outputMode == "json" {
							assertCommandErrorEnvelope(t, stdout, stderr, commandErrorEnvelopeExpectation{
								Outcome: "failed",
								Class:   tt.class,
								Command: "embed list",
								Message: tt.message,
							})
						} else {
							assertHumanCommandError(t, stdout, stderr, tt.message)
						}
					})
				}
			}
		})
	}
}

// TestCommandErrorEnvelopeEmbedListHelper invokes the production runner with
// a process-local listing function and configured command identity.
func TestCommandErrorEnvelopeEmbedListHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, embedListErrorEnvelopeHelper, os.Getenv(testx.CmdSubprocessNameEnv))

	parent := &cobra.Command{Use: "embed"}
	cmd := &cobra.Command{Use: "list"}
	parent.AddCommand(cmd)
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	setContractSupport(cmd, ContractSupportFull)

	flagViewAsJson = os.Getenv("C8VOLT_TEST_OUTPUT_MODE") == "json"
	cfg := &config.Config{App: config.App{
		CamundaVersion: toolx.V88,
		NoErrCodes:     os.Getenv("C8VOLT_TEST_NO_ERR_CODES") == "1",
	}}
	log := slog.New(logging.NewPlainHandler(os.Stderr, slog.LevelInfo))

	var list embeddedListFunc
	switch os.Getenv("C8VOLT_TEST_EMBED_FAILURE") {
	case "error":
		list = func() ([]string, error) { return nil, errors.New("list embedded files failed") }
	case "empty":
		list = func() ([]string, error) { return nil, nil }
	case "nonmatching":
		list = func() ([]string, error) { return []string{"processdefinitions/C89_Only.bpmn"}, nil }
	default:
		t.Fatalf("unknown embedded failure fixture")
	}

	runEmbedListWithListing(cmd, log, cfg, list)
}
