// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const processInstancePagingTerminalConfigEnv = "C8VOLT_PI_PAGING_TERMINAL_CONFIG"

// TestGetProcessInstanceKeysOnlyPagingTerminal verifies real-terminal continuation keeps the key stream clean and stops without another request.
func TestGetProcessInstanceKeysOnlyPagingTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runGetProcessInstanceKeysOnlyPagingTerminalHelper(t)
		os.Exit(0)
	}

	firstPrompt := "Fetched 1 process instance(s) on this page (1/3+ loaded). More matching process instances remain.\nContinue? [y/N]: "
	secondPrompt := "Fetched 1 process instance(s) on this page (2/3+ loaded). More matching process instances remain.\nContinue? [y/N]: "
	responses := []string{
		processInstancePagingTerminalResponse("123", true),
		processInstancePagingTerminalResponse("124", true),
		processInstancePagingTerminalResponse("125", false),
	}
	tests := []struct {
		name         string
		exchanges    []testx.CmdTerminalExchange
		wantStdout   string
		wantStderr   string
		wantRequests int
	}{
		{
			name: "repeated continuation reaches completion",
			exchanges: []testx.CmdTerminalExchange{
				{Prompt: firstPrompt, Response: "y"},
				{Prompt: secondPrompt, Response: "yes"},
			},
			wantStdout:   "123\n124\n125\n",
			wantStderr:   firstPrompt + secondPrompt,
			wantRequests: 3,
		},
		{
			name: "decline after one continuation retains fetched keys",
			exchanges: []testx.CmdTerminalExchange{
				{Prompt: firstPrompt, Response: "y"},
				{Prompt: secondPrompt, Response: "n"},
			},
			wantStdout:   "123\n124\n",
			wantStderr:   firstPrompt + secondPrompt,
			wantRequests: 2,
		},
		{
			name: "EOF stops before another page",
			exchanges: []testx.CmdTerminalExchange{
				{Prompt: firstPrompt, EndOfInput: true},
			},
			wantStdout:   "123\n",
			wantStderr:   firstPrompt,
			wantRequests: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests, responses...)
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestGetProcessInstanceKeysOnlyPagingTerminal",
				Env: map[string]string{
					processInstancePagingTerminalConfigEnv: cfgPath,
				},
				Exchanges: tt.exchanges,
				Timeout:   3 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.True(t, result.Supported, result.UnsupportedReason)
			require.NoError(t, result.Err, result.Stderr)
			require.Equal(t, tt.wantStdout, result.Stdout)
			require.Equal(t, tt.wantStderr, result.Stderr)
			require.Equal(t, tt.wantRequests, len(requests))
			require.Equal(t, len(tt.exchanges), strings.Count(result.Stderr, "Continue? [y/N]: "))
			require.NotContains(t, result.Stdout, "Fetched")
			require.NotContains(t, result.Stdout, "Continue?")
		})
	}
}

// runGetProcessInstanceKeysOnlyPagingTerminalHelper executes the real keys-only command path with terminal stdin.
func runGetProcessInstanceKeysOnlyPagingTerminalHelper(t *testing.T) {
	require.Equal(t, "TestGetProcessInstanceKeysOnlyPagingTerminal", os.Getenv(testx.CmdSubprocessNameEnv))

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SetArgs([]string{
		"--config", os.Getenv(processInstancePagingTerminalConfigEnv),
		"--tenant", "tenant",
		"--keys-only",
		"get", "process-instance",
		"--batch-size", "1",
	})

	_, err := root.ExecuteC()
	require.NoError(t, err)
}

// processInstancePagingTerminalResponse returns one fake page with stable keys and continuation metadata.
func processInstancePagingTerminalResponse(key string, hasMore bool) string {
	return `{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"` + key + `","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":` + strconv.FormatBool(hasMore) + `}}`
}
