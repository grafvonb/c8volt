// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// flagIntentSurface inventories every rename consumer, including retained job controls.
var flagIntentSurface = []struct {
	cmd   *cobra.Command
	names map[string]string
}{
	{updateElementCmd, map[string]string{"vars": "set-vars", "vars-file": "set-vars-file"}},
	{updateProcessInstanceCmd, map[string]string{"vars": "set-vars", "vars-file": "set-vars-file"}},
	{updateUserTaskCmd, map[string]string{"vars": "set-vars", "vars-file": "set-vars-file", "assign": "set-assignee"}},
	{updateJobCmd, map[string]string{"vars": "set-vars", "retries": "set-retries", "timeout": "set-timeout", "message": "set-error-message", "retry-backoff": "set-retry-backoff"}},
	{runProcessInstanceCmd, map[string]string{"vars": "set-vars", "bid": "set-bid"}},
	{opsRepairIncidentCmd, map[string]string{"vars": "set-vars", "vars-file": "set-vars-file", "retries": "set-retries", "job-timeout": "set-job-timeout"}},
	{opsRepairProcessInstanceCmd, map[string]string{"vars": "set-vars", "vars-file": "set-vars-file", "retries": "set-retries", "job-timeout": "set-job-timeout"}},
	{getProcessDefinitionCmd, map[string]string{"stat": "with-stats"}},
	{embedListCmd, map[string]string{"details": "with-paths"}},
	{getIncidentCmd, map[string]string{"with-no-error-message": "no-error-message"}},
}

// TestFlagIntentEnrichmentReporting requires one configured diagnostic or envelope
// for removed Boolean input, even with equal canonical input and later controls.
func TestFlagIntentEnrichmentReporting(t *testing.T) {
	var requests atomic.Int64
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	for _, tc := range flagIntentSurface[7:] {
		for old, canonical := range tc.names {
			for _, format := range []string{"plain-time", "json"} {
				for _, destination := range []string{"leaf", "inherited"} {
					for _, mode := range []string{"normal", "verbose", "quiet", "json"} {
						t.Run(tc.cmd.CommandPath()+"/"+format+"/"+destination+"/"+mode, func(t *testing.T) {
							capture := filepath.Join(t.TempDir(), "diagnostic.log")
							args := strings.Fields(strings.TrimPrefix(tc.cmd.CommandPath(), "c8volt "))
							legacy := "--" + old + "=false"
							if mode == "normal" {
								legacy = "--" + old
							}
							args = append(args, "--"+canonical+"=false", legacy, "--config", completionLogConfig(t, server.URL, format, "warn"))
							if mode != "normal" {
								args = append(args, "--"+mode)
							}
							encoded, err := json.Marshal(args)
							require.NoError(t, err)
							stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentMigrationProductionHelper", t.TempDir(), map[string]string{"C8VOLT_TEST_ARGS": string(encoded), "C8VOLT_TEST_DESTINATION": destination, "C8VOLT_TEST_CAPTURE": capture}, "", exitcode.InvalidArgs)
							require.Empty(t, stderr)
							data, err := os.ReadFile(capture)
							require.NoError(t, err)
							message := "invalid input: --" + old + " was removed; use --" + canonical
							if mode == "json" {
								assertCommandErrorEnvelope(t, stdout, string(data), commandErrorEnvelopeExpectation{Outcome: "invalid", Class: "invalid_input", Command: strings.TrimPrefix(tc.cmd.CommandPath(), "c8volt "), Message: message})
							} else {
								require.Empty(t, stdout)
								require.Equal(t, [][2]string{{"ERROR", message}}, bulkCompletionDiagnosticRecords(t, string(data), format))
							}
						})
					}
				}
			}
		}
	}
	require.Zero(t, requests.Load())
}
