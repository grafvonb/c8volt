// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// requestRatePromptWriter observes completed summaries without relying on how
// quickly the scheduler runs the reporter after the user answers.
type requestRatePromptWriter struct {
	io.Writer
	reports chan struct{}
}

// Write forwards to the configured destination before notifying the test.
func (w *requestRatePromptWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if strings.Contains(string(p), "HTTP request rate:") {
		select {
		case w.reports <- struct{}{}:
		default:
		}
	}
	return n, err
}

// TestRequestRateManualPrompt keeps real terminal input pending across several
// report intervals and checks both prompt policies, destinations, and exit paths.
func TestRequestRateManualPrompt(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, tc := range []struct {
		name, policy, answer, format string
		configured, eof, aborted     bool
	}{
		{name: "default no accepts", answer: "y", format: "plain"},
		{name: "default no declines", answer: "n", format: "json", configured: true, aborted: true},
		{name: "default no EOF", eof: true, format: "plain", aborted: true},
		{name: "default yes accepts empty", policy: "yes", format: "json", configured: true},
		{name: "default yes declines", policy: "yes", answer: "n", format: "plain", aborted: true},
		{name: "default yes EOF", policy: "yes", eof: true, format: "json", configured: true, aborted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Auth-none setup performs no remote work; the command below only asks for input.
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("prompt-only command unexpectedly dispatched a request")
				w.WriteHeader(http.StatusInternalServerError)
			}))
			t.Cleanup(server.Close)
			cfg := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")
			appendRequestRateConfig(t, cfg, "10ms")
			if tc.format == "json" {
				contents, err := os.ReadFile(cfg)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(cfg, append(contents, []byte("log:\n  level: debug\n")...), 0o600))
			}
			capture := filepath.Join(t.TempDir(), "stderr")
			choice := "[y/N]"
			if tc.policy == "yes" {
				choice = "[Y/n]"
			}
			prompt := "Continue? " + choice + ": "
			var pendingOutput string
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestRequestRateManualPromptProcess",
				Env: map[string]string{
					"C8VOLT_RATE_PROMPT_CONFIG": cfg, "C8VOLT_RATE_PROMPT_POLICY": tc.policy,
					"C8VOLT_RATE_PROMPT_FORMAT": tc.format, "C8VOLT_RATE_PROMPT_CAPTURE": capture,
					"C8VOLT_RATE_PROMPT_CONFIGURED": fmt.Sprint(tc.configured),
				},
				Exchanges: []testx.CmdTerminalExchange{{Prompt: prompt, Response: tc.answer, EndOfInput: tc.eof, ResponseDelay: 80 * time.Millisecond,
					BeforeResponse: func(_, stderr string) { pendingOutput = stderr },
				}},
				Timeout: 5 * time.Second,
			})
			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.NoError(t, result.Err, result.Stderr)
			configured, err := os.ReadFile(capture)
			require.NoError(t, err)
			require.Equal(t, string(configured), result.Stderr, "all diagnostics must use the effective writer")
			require.Equal(t, fmt.Sprintf("aborted=%t\n", tc.aborted), result.Stdout)
			_, waiting, ok := strings.Cut(pendingOutput, prompt)
			require.True(t, ok, pendingOutput)
			require.Empty(t, waiting, "nothing may interrupt the unanswered question")
			_, pending, ok := strings.Cut(result.Stderr, prompt)
			require.True(t, ok, result.Stderr)
			_, resumed, ok := strings.Cut(pending, "manual input finished")
			require.True(t, ok, result.Stderr)
			// Discard the end of the marker record before decoding structured reports.
			_, reports, ok := strings.Cut(resumed, "\n")
			require.True(t, ok)
			requireRequestRateSummaries(t, reports, tc.format)
			require.NotContains(t, result.Stdout, "HTTP request rate:")
		})
	}
}

// TestRequestRatePromptBypass leaves reporting untouched for automatic confirmation
// and non-terminal input; only actual manual questions may acquire a pause.
func TestRequestRatePromptBypass(t *testing.T) {
	previous := currentHTTPService
	currentHTTPService = &requestRateServiceStub{pause: func() func() {
		t.Fatal("noninteractive confirmation paused reporting")
		return func() {}
	}}
	t.Cleanup(func() { currentHTTPService = previous })
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	defer reader.Close()
	defer writer.Close()
	stdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = stdin })
	for _, confirm := range []func(io.Writer, bool, string) error{confirmCmdOrAbort, confirmCmdOrAbortDefaultYes} {
		for _, automatic := range []bool{true, false} {
			var output strings.Builder
			require.NoError(t, confirm(&output, automatic, "Continue?"))
			require.Empty(t, output.String())
		}
	}
}

// TestRequestRateManualPromptProcess exercises the real bootstrap/logger and
// confirmation helpers with a test-only command, without any remote dependency.
func TestRequestRateManualPromptProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	root := Root()
	resetCommandTreeFlags(root)
	resetRequestRateCommandContexts(root)
	reports := make(chan struct{}, 1)
	command := &cobra.Command{Use: "request-rate-prompt-test", RunE: func(cmd *cobra.Command, _ []string) error {
		confirm := confirmCmdOrAbort
		if os.Getenv("C8VOLT_RATE_PROMPT_POLICY") == "yes" {
			confirm = confirmCmdOrAbortDefaultYes
		}
		err := confirm(cmd.ErrOrStderr(), false, "Continue?")
		log, logErr := logging.FromContext(cmd.Context())
		require.NoError(t, logErr)
		log.Info("manual input finished")
		// Keep the invocation alive after every return path to prove deferred resume.
		for len(reports) > 0 {
			<-reports
		}
		select {
		case <-reports:
		case <-time.After(2 * time.Second):
			t.Fatal("reporting did not resume after manual input")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "aborted=%t\n", err != nil)
		return nil
	}}
	root.AddCommand(command)
	defer root.RemoveCommand(command)
	capture, err := os.Create(os.Getenv("C8VOLT_RATE_PROMPT_CAPTURE"))
	require.NoError(t, err)
	defer capture.Close()
	writer := &requestRatePromptWriter{Writer: io.MultiWriter(os.Stderr, capture), reports: reports}
	root.SetErr(writer)
	if os.Getenv("C8VOLT_RATE_PROMPT_CONFIGURED") == "true" {
		command.SetErr(writer)
		root.SetErr(io.Discard)
	}
	root.SetOut(os.Stdout)
	args := []string{"--config", os.Getenv("C8VOLT_RATE_PROMPT_CONFIG"), "--log-format", os.Getenv("C8VOLT_RATE_PROMPT_FORMAT"), command.Name()}
	if os.Getenv("C8VOLT_RATE_PROMPT_FORMAT") != "json" {
		args = append(args, "--debug")
	}
	root.SetArgs(args)
	_, err = root.ExecuteC()
	require.NoError(t, err)
	require.NoError(t, capture.Close())
	os.Exit(0)
}
