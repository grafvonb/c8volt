// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGetProcessDefinitionWatchFlagRegistered(t *testing.T) {
	flag := getProcessDefinitionCmd.Flags().Lookup("watch")

	require.NotNil(t, flag)
	require.Equal(t, "bool", flag.Value.Type())
	require.Contains(t, flag.Usage, "repeat the process-definition lookup")
}

func TestGetProcessDefinitionWatchIntervalFlagRegistered(t *testing.T) {
	flag := getProcessDefinitionCmd.Flags().Lookup("watch-interval")

	require.NotNil(t, flag)
	require.Equal(t, "duration", flag.Value.Type())
	require.Equal(t, "1s", flag.DefValue)
	require.Contains(t, flag.Usage, "after the immediate first snapshot")
}

func TestGetProcessDefinitionSelectionFlagsRemainSearchFilters(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	flagGetPDKey = "2251799813685255"
	flagGetPDBpmnProcessId = "invoice"
	flagGetPDProcessVersion = 3
	flagGetPDProcessVersionTag = "stable"
	flagGetPDLatest = true

	filter := populatePDSearchFilterOpts()

	require.Equal(t, process.ProcessDefinitionFilter{
		Key:               "2251799813685255",
		BpmnProcessId:     "invoice",
		ProcessVersion:    3,
		ProcessVersionTag: "stable",
	}, filter)
}

func TestGetProcessDefinitionWatchImmediateRepeatedBroadSnapshots(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	var (
		events   []string
		requests []process.ProcessDefinitionWatchSnapshotRequest
	)
	cli := processDefinitionWatchTestAPI{
		collect: func(_ context.Context, request process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
			events = append(events, "collect")
			requests = append(requests, request)
			item := process.ProcessDefinition{
				Key:            "2251799813685255",
				TenantId:       "tenant",
				BpmnProcessId:  "invoice",
				ProcessVersion: int32(len(requests)),
			}
			return process.ProcessDefinitionWatchSnapshot{
				Items: []process.ProcessDefinition{item},
				Total: 1,
			}, nil
		},
	}
	sleepCalls := 0
	output, err := executeGetProcessDefinitionWatchForTest(t, cli, process.ProcessDefinitionFilter{}, 0, func(_ context.Context, interval time.Duration) error {
		events = append(events, "sleep")
		require.Equal(t, time.Second, interval)
		sleepCalls++
		if sleepCalls == 1 {
			return nil
		}
		return context.Canceled
	})

	require.NoError(t, err)
	require.Equal(t, []string{"collect", "sleep", "collect", "sleep"}, events)
	require.Len(t, requests, 2)
	for _, request := range requests {
		require.True(t, request.WatchAllWhenUnselected)
		require.Equal(t, int32(1000), request.Page.Size)
		require.False(t, request.Latest)
	}
	require.Contains(t, output, "snapshot 1:")
	require.Contains(t, output, "2251799813685255 tenant invoice v1")
	require.Contains(t, output, "snapshot 2:")
	require.Contains(t, output, "2251799813685255 tenant invoice v2")
	require.Contains(t, output, "found: 1")
}

func TestGetProcessDefinitionWatchIntervalCadence(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected time.Duration
	}{
		{name: "default", expected: time.Second},
		{name: "explicit", raw: "2s", expected: 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)
			if tt.raw != "" {
				flagGetPDWatchInterval = tt.raw
			}

			var intervals []time.Duration
			cli := processDefinitionWatchTestAPI{
				collect: func(_ context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
					return process.ProcessDefinitionWatchSnapshot{
						Items: []process.ProcessDefinition{{
							Key:            "2251799813685255",
							TenantId:       "tenant",
							BpmnProcessId:  "invoice",
							ProcessVersion: 1,
						}},
						Total: 1,
					}, nil
				},
			}

			stdout, stderr, err := executeGetProcessDefinitionWatchWithBackoffForTest(t, cli, process.ProcessDefinitionFilter{}, 0, defaultBackoffMaxRetries, func(_ context.Context, interval time.Duration) error {
				intervals = append(intervals, interval)
				return context.Canceled
			})

			require.NoError(t, err)
			require.Equal(t, []time.Duration{tt.expected}, intervals)
			require.Contains(t, stdout, "snapshot 1:")
			require.Empty(t, stderr)
		})
	}
}

func TestValidateGetProcessDefinitionWatchIntervalRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "invalid", raw: "soon"},
		{name: "zero", raw: "0s"},
		{name: "negative", raw: "-1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)
			flagGetPDWatch = true
			flagGetPDWatchInterval = tt.raw

			err := validateGetProcessDefinitionFlags(getProcessDefinitionCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid value for --watch-interval")
			require.Contains(t, err.Error(), tt.raw)
		})
	}
}

// TestValidateGetProcessDefinitionWatchRejectsMachineOutputModes verifies watch
// validation blocks finite output contracts before snapshot collection can run.
func TestValidateGetProcessDefinitionWatchRejectsMachineOutputModes(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cobra.Command)
		want  string
	}{
		{
			name: "json",
			setup: func(*cobra.Command) {
				flagViewAsJson = true
			},
			want: "--json",
		},
		{
			name: "keys only",
			setup: func(*cobra.Command) {
				flagViewKeysOnly = true
			},
			want: "--keys-only",
		},
		{
			name: "xml",
			setup: func(*cobra.Command) {
				flagGetPDAsXML = true
			},
			want: "--xml",
		},
		{
			name: "quiet",
			setup: func(*cobra.Command) {
				flagQuiet = true
			},
			want: "--quiet",
		},
		{
			name: "automation",
			setup: func(*cobra.Command) {
				flagCmdAutomation = true
			},
			want: "--automation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)
			cmd := &cobra.Command{Use: "process-definition"}
			cmd.SetContext(context.Background())
			flagGetPDWatch = true
			tt.setup(cmd)

			err := validateGetProcessDefinitionFlags(cmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), "--watch cannot be combined")
			require.Contains(t, err.Error(), tt.want)
			require.Contains(t, err.Error(), "terminal snapshots")
		})
	}
}

// TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookup proves the
// command exits during local validation without contacting Camunda.
func TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookup(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "json", args: []string{"--json", "get", "process-definition", "--watch"}, want: "--json"},
		{name: "keys only", args: []string{"--keys-only", "get", "process-definition", "--watch"}, want: "--keys-only"},
		{name: "xml", args: []string{"get", "process-definition", "--watch", "--xml"}, want: "--xml"},
		{name: "quiet", args: []string{"--quiet", "get", "process-definition", "--watch"}, want: "--quiet"},
		{name: "automation", args: []string{"--automation", "get", "process-definition", "--watch"}, want: "--automation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookupHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":  cfgPath,
				"C8VOLT_TEST_PD_ARGS": marshalStringSliceForEnv(t, tt.args),
			})

			require.Error(t, err)
			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "--watch cannot be combined")
			require.Contains(t, string(output), tt.want)
			require.Empty(t, requests)
		})
	}
}

// TestGetProcessDefinitionNonWatchMachineModesStayCompatible keeps finite
// process-definition output modes unchanged while watch validation is added.
func TestGetProcessDefinitionNonWatchMachineModesStayCompatible(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantStdout   string
		wantNoStdout string
	}{
		{
			name:       "json",
			args:       []string{"--json", "get", "process-definition"},
			wantStdout: `"outcome": "succeeded"`,
		},
		{
			name:       "keys only",
			args:       []string{"--keys-only", "get", "process-definition"},
			wantStdout: "2251799813685255\n",
		},
		{
			name:       "quiet",
			args:       []string{"--quiet", "get", "process-definition"},
			wantStdout: "2251799813685255",
		},
		{
			name:       "automation",
			args:       []string{"--automation", "get", "process-definition"},
			wantStdout: "found: 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []map[string]any
			srv := newProcessDefinitionSearchServerResponses(t, &requests,
				`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
			)
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
			args := append([]string{"--config", cfgPath}, tt.args...)

			stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, args...)

			require.Len(t, requests, 1)
			require.Empty(t, stderr)
			require.Contains(t, stdout, tt.wantStdout)
			if tt.wantNoStdout != "" {
				require.NotContains(t, stdout, tt.wantNoStdout)
			}
		})
	}
}

func TestGetProcessDefinitionWatchUsesDefaultRetryBudget(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	transientErr := ferrors.WrapClass(ferrors.ErrUnavailable, errors.New("temporary Camunda outage"))
	calls := 0
	cli := processDefinitionWatchTestAPI{
		collect: func(_ context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
			calls++
			return process.ProcessDefinitionWatchSnapshot{}, transientErr
		},
	}

	stdout, stderr, err := executeGetProcessDefinitionWatchWithBackoffForTest(t, cli, process.ProcessDefinitionFilter{}, 0, defaultBackoffMaxRetries, func(context.Context, time.Duration) error {
		if calls >= 3 {
			return context.Canceled
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 3, calls)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "retrying process-definition watch after snapshot 1 failed")
	require.Contains(t, stderr, "consecutive failures: 3")
}

func TestGetProcessDefinitionWatchRetryBudgetResetsAfterSuccess(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	transientErr := ferrors.WrapClass(ferrors.ErrUnavailable, errors.New("temporary Camunda outage"))
	calls := 0
	cli := processDefinitionWatchTestAPI{
		collect: func(_ context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
			calls++
			if calls == 1 || calls == 3 {
				return process.ProcessDefinitionWatchSnapshot{}, transientErr
			}
			return process.ProcessDefinitionWatchSnapshot{
				Items: []process.ProcessDefinition{{
					Key:            "2251799813685255",
					TenantId:       "tenant",
					BpmnProcessId:  "invoice",
					ProcessVersion: 2,
				}},
				Total: 1,
			}, nil
		},
	}

	stdout, stderr, err := executeGetProcessDefinitionWatchWithBackoffForTest(t, cli, process.ProcessDefinitionFilter{}, 0, 1, func(context.Context, time.Duration) error {
		if calls >= 3 {
			return context.Canceled
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 3, calls)
	require.Contains(t, stdout, "snapshot 2:")
	require.NotContains(t, stdout, "retrying")
	require.NotContains(t, stdout, "temporary Camunda outage")
	require.Equal(t, 2, strings.Count(stderr, "retrying process-definition watch"))
	require.Contains(t, stderr, "snapshot 1 failed (1/1 consecutive failures)")
	require.Contains(t, stderr, "snapshot 3 failed (1/1 consecutive failures)")
}

func TestGetProcessDefinitionWatchRetryExhaustionReturnsErrorAndStderrStatus(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	transientErr := ferrors.WrapClass(ferrors.ErrUnavailable, errors.New("temporary Camunda outage"))
	calls := 0
	cli := processDefinitionWatchTestAPI{
		collect: func(_ context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
			calls++
			return process.ProcessDefinitionWatchSnapshot{}, transientErr
		},
	}

	stdout, stderr, err := executeGetProcessDefinitionWatchWithBackoffForTest(t, cli, process.ProcessDefinitionFilter{}, 0, 1, func(context.Context, time.Duration) error {
		return nil
	})

	require.Error(t, err)
	require.Equal(t, 2, calls)
	require.Empty(t, stdout)
	require.Contains(t, err.Error(), "watch retry exhausted")
	require.Contains(t, stderr, "retrying process-definition watch after snapshot 1 failed")
	require.Contains(t, stderr, "watch stopped: retry budget exhausted after 2 consecutive failure(s)")
}

// TestGetProcessDefinitionWatchHumanModesUseSnapshotRows verifies default and
// verbose watch output keep the same compact result rows on stdout.
func TestGetProcessDefinitionWatchHumanModesUseSnapshotRows(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{name: "default"},
		{name: "verbose", verbose: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)
			flagVerbose = tt.verbose
			cli := processDefinitionWatchTestAPI{
				collect: func(_ context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
					return process.ProcessDefinitionWatchSnapshot{
						Items: []process.ProcessDefinition{{
							Key:            "2251799813685255",
							TenantId:       "tenant",
							BpmnProcessId:  "invoice",
							ProcessVersion: 3,
						}},
						Total: 1,
					}, nil
				},
			}

			stdout, stderr, err := executeGetProcessDefinitionWatchWithBackoffForTest(t, cli, process.ProcessDefinitionFilter{}, 0, defaultBackoffMaxRetries, func(context.Context, time.Duration) error {
				return context.Canceled
			})

			require.NoError(t, err)
			require.Empty(t, stderr)
			require.Equal(t, "snapshot 1:\n2251799813685255 tenant invoice v3\nfound: 1\n", stdout)
		})
	}
}

// TestGetProcessDefinitionWatchKeyAndStatRemainHumanCompatible proves direct
// key observations still carry admin input and stat options in watch mode.
func TestGetProcessDefinitionWatchKeyAndStatRemainHumanCompatible(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	flagGetPDKey = "2251799813685255"
	flagGetPDWithStat = true

	var gotOptions *options.FacadeCfg
	cli := processDefinitionWatchTestAPI{
		collect: func(_ context.Context, request process.ProcessDefinitionWatchSnapshotRequest, opts ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
			require.Equal(t, "2251799813685255", request.Key)
			gotOptions = options.ApplyFacadeOptions(opts)
			return process.ProcessDefinitionWatchSnapshot{
				Items: []process.ProcessDefinition{{
					Key:            "2251799813685255",
					TenantId:       "tenant",
					BpmnProcessId:  "invoice",
					ProcessVersion: 3,
					Statistics: &process.ProcessDefinitionStatistics{
						Active:                 2,
						Completed:              5,
						Canceled:               1,
						Incidents:              1,
						IncidentCountSupported: true,
					},
				}},
				Total: 1,
			}, nil
		},
	}

	output, err := executeGetProcessDefinitionWatchForTest(t, cli, populatePDSearchFilterOpts(), 0, func(context.Context, time.Duration) error {
		return context.Canceled
	})

	require.NoError(t, err)
	require.True(t, gotOptions.IgnoreTenant)
	require.True(t, gotOptions.Stat)
	require.Contains(t, output, "2251799813685255 tenant invoice v3 [ac:2 cp:5 cx:1 inc:1]")
}

func TestGetProcessDefinitionWatchExplicitLatestEmptyThenChangedSnapshot(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	flagGetPDBpmnProcessId = "invoice"
	flagGetPDLatest = true

	var requests []process.ProcessDefinitionWatchSnapshotRequest
	cli := processDefinitionWatchTestAPI{
		collect: func(_ context.Context, request process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
			requests = append(requests, request)
			if len(requests) == 1 {
				return process.ProcessDefinitionWatchSnapshot{Empty: true}, nil
			}
			item := process.ProcessDefinition{
				Key:            "2251799813685256",
				TenantId:       "tenant",
				BpmnProcessId:  "invoice",
				ProcessVersion: 7,
			}
			return process.ProcessDefinitionWatchSnapshot{
				Items: []process.ProcessDefinition{item},
				Total: 1,
			}, nil
		},
	}
	sleepCalls := 0
	output, err := executeGetProcessDefinitionWatchForTest(t, cli, populatePDSearchFilterOpts(), 0, func(context.Context, time.Duration) error {
		sleepCalls++
		if sleepCalls == 1 {
			return nil
		}
		return context.Canceled
	})

	require.NoError(t, err)
	require.Len(t, requests, 2)
	for _, request := range requests {
		require.False(t, request.WatchAllWhenUnselected)
		require.True(t, request.Latest)
		require.Equal(t, "invoice", request.Filter.BpmnProcessId)
	}
	require.Contains(t, output, "snapshot 1:\nfound: 0")
	require.Contains(t, output, "snapshot 2:")
	require.Contains(t, output, "2251799813685256 tenant invoice v7")
	require.Contains(t, output, "found: 1")
}

func TestGetProcessDefinitionWatchKeySnapshotUsesExplicitAdminOptions(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	flagGetPDKey = "2251799813685255"

	var gotOptions *options.FacadeCfg
	cli := processDefinitionWatchTestAPI{
		collect: func(_ context.Context, request process.ProcessDefinitionWatchSnapshotRequest, opts ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
			require.Equal(t, "2251799813685255", request.Key)
			gotOptions = options.ApplyFacadeOptions(opts)
			return process.ProcessDefinitionWatchSnapshot{
				Items: []process.ProcessDefinition{{
					Key:            "2251799813685255",
					TenantId:       "tenant",
					BpmnProcessId:  "invoice",
					ProcessVersion: 3,
				}},
				Total: 1,
			}, nil
		},
	}

	output, err := executeGetProcessDefinitionWatchForTest(t, cli, populatePDSearchFilterOpts(), 0, func(context.Context, time.Duration) error {
		return context.Canceled
	})

	require.NoError(t, err)
	require.True(t, gotOptions.IgnoreTenant)
	require.Contains(t, output, "2251799813685255 tenant invoice v3")
}

func TestGetProcessDefinitionWatchInterruptAndTimeoutStopCleanly(t *testing.T) {
	tests := []struct {
		name     string
		sleepErr error
	}{
		{name: "interrupt", sleepErr: context.Canceled},
		{name: "timeout", sleepErr: context.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)

			cli := processDefinitionWatchTestAPI{
				collect: func(_ context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
					return process.ProcessDefinitionWatchSnapshot{
						Items: []process.ProcessDefinition{{
							Key:            "2251799813685255",
							TenantId:       "tenant",
							BpmnProcessId:  "invoice",
							ProcessVersion: 1,
						}},
						Total: 1,
					}, nil
				},
			}

			output, err := executeGetProcessDefinitionWatchForTest(t, cli, process.ProcessDefinitionFilter{}, 0, func(context.Context, time.Duration) error {
				return tt.sleepErr
			})

			require.NoError(t, err)
			require.Contains(t, output, "snapshot 1:")
			require.NotContains(t, strings.ToLower(output), "failed")
			require.NotContains(t, strings.ToLower(output), "error")
			require.NotContains(t, strings.ToLower(output), "lookup")
		})
	}
}

func TestGetProcessDefinitionWatchTimeoutStatusUsesStderr(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	cli := processDefinitionWatchTestAPI{
		collect: func(_ context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
			return process.ProcessDefinitionWatchSnapshot{
				Items: []process.ProcessDefinition{{
					Key:            "2251799813685255",
					TenantId:       "tenant",
					BpmnProcessId:  "invoice",
					ProcessVersion: 1,
				}},
				Total: 1,
			}, nil
		},
	}

	stdout, stderr, err := executeGetProcessDefinitionWatchWithBackoffForTest(t, cli, process.ProcessDefinitionFilter{}, 0, defaultBackoffMaxRetries, func(context.Context, time.Duration) error {
		return context.DeadlineExceeded
	})

	require.NoError(t, err)
	require.Contains(t, stdout, "snapshot 1:")
	require.NotContains(t, stdout, "timeout")
	require.Contains(t, stderr, "watch stopped: timeout reached")
}

func TestGetProcessDefinitionLatestSearchPreservesSelectionRequest(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionLatestSearchPreservesSelectionRequestHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.NoError(t, err, string(output))
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "invoice", filter["processDefinitionId"])
	require.Equal(t, float64(3), filter["version"])
	require.Equal(t, "stable", filter["versionTag"])
	require.Equal(t, true, filter["isLatestVersion"])
}

func TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnostic(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		writeEmptyProcessDefinitionSearchResponse(w)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnosticHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Len(t, requests, 1)
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "missing-process", filter["processDefinitionId"])
	require.Contains(t, string(output), "no visible process definition matches the provided selector")
	require.Contains(t, string(output), "[missing-process]")
}

func TestGetProcessDefinitionBpmnSelectorVisiblePreservesListing(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"order-process","name":"Order Process","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output := executeRootForTest(t,
		"--config", cfgPath,
		"get", "process-definition",
		"--bpmn-process-id", "order-process",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
	)

	require.Len(t, requests, 1)
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, float64(3), filter["version"])
	require.Equal(t, "stable", filter["versionTag"])
	require.Contains(t, output, "2251799813685255")
	require.Contains(t, output, "tenant order-process v3/stable")
}

// TestGetProcessDefinitionSearchVerboseProgress defines the process-definition progress contract for broad listing.
func TestGetProcessDefinitionSearchVerboseProgress(t *testing.T) {
	var requests []map[string]any
	srv := newProcessDefinitionSearchServerResponses(t, &requests,
		`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"},{"processDefinitionKey":"2251799813685256","processDefinitionId":"payment","name":"payment","version":2,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"pd-page-2"}}`,
		`{"items":[{"processDefinitionKey":"2251799813685257","processDefinitionId":"shipping","name":"shipping","version":1,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
		"--config", cfgPath,
		"--verbose",
		"--auto-confirm",
		"get", "process-definition",
		"--batch-size", "2",
	)

	require.Len(t, requests, 2)
	firstPage := requireJSONObject(t, requests[0]["page"])
	require.Equal(t, float64(2), firstPage["limit"])
	require.Equal(t, float64(0), firstPage["from"])
	secondPage := requireJSONObject(t, requests[1]["page"])
	require.Equal(t, "pd-page-2", secondPage["after"])
	require.Contains(t, stderr, "process-definition search scope: matched at least 3 process definitions; page size: 2; discovery pages: at least 2")
	require.Contains(t, stderr, "discovering process definitions, page 1/~2, 2 seen")
	require.Contains(t, stderr, "discovering process definitions, page 2/2, 3 seen")
	require.Contains(t, stdout, "2251799813685255")
	require.Contains(t, stdout, "2251799813685256")
	require.Contains(t, stdout, "2251799813685257")
	require.Contains(t, stdout, "found: 3")
}

// TestGetProcessDefinitionSearchMachineOutputStaysProgressFree protects process-definition JSON/key streams during progress rollout.
func TestGetProcessDefinitionSearchMachineOutputStaysProgressFree(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var requests []map[string]any
		srv := newProcessDefinitionSearchServerResponses(t, &requests,
			`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
			"--config", cfgPath,
			"--json",
			"get", "process-definition",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering process definitions")
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
		payload := requireJSONObject(t, envelope["payload"])
		items := payload["items"].([]any)
		require.Len(t, items, 1)
	})

	t.Run("keys only", func(t *testing.T) {
		var requests []map[string]any
		srv := newProcessDefinitionSearchServerResponses(t, &requests,
			`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"},{"processDefinitionKey":"2251799813685256","processDefinitionId":"payment","name":"payment","version":2,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
			"--config", cfgPath,
			"--keys-only",
			"get", "process-definition",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.Equal(t, "2251799813685255\n2251799813685256\n", stdout)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering process definitions")
	})
}

func TestGetProcessDefinitionXMLOutputRemainsKeyOnlyDisplayMode(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*process.ProcessDefinitionFilter)
		assert func(*testing.T, error)
	}{
		{
			name: "key only accepted",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "missing key",
			setup: func(*process.ProcessDefinitionFilter) {},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "xml output requires --key")
			},
		},
		{
			name: "bpmn process id",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				filter.BpmnProcessId = "invoice"
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--bpmn-process-id"),
		},
		{
			name: "process version",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDProcessVersion = 3
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--pd-version"),
		},
		{
			name: "process version tag",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				filter.ProcessVersionTag = "stable"
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--pd-version-tag"),
		},
		{
			name: "latest",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDLatest = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--latest"),
		},
		{
			name: "stat",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDWithStat = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--stat"),
		},
		{
			name: "json",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagViewAsJson = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--json"),
		},
		{
			name: "keys only",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagViewKeysOnly = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--keys-only"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)

			var filter process.ProcessDefinitionFilter
			tt.setup(&filter)
			tt.assert(t, validateProcessDefinitionXMLFlags(filter))
		})
	}
}

func requireXMLDisplayModeIncompatibleFlag(flag string) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		require.Contains(t, err.Error(), "xml output only supports --key")
		require.Contains(t, err.Error(), flag)
	}
}

func TestSearchProcessDefinitionsWithPagingStatUsesCommandActivity(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	flagGetPDWithStat = true

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	cli := processDefinitionPagingActivityAPI{
		searchProcessDefinitionsPages: func(ctx context.Context, request process.ProcessDefinitionSearchRequest, visitor process.ProcessDefinitionSearchPageVisitor, opts ...options.FacadeOption) (process.ProcessDefinitionSearchPagesResult, error) {
			require.Equal(t, int32(1000), request.Page.Size)
			require.True(t, options.ApplyFacadeOptions(opts).Stat)
			return process.ProcessDefinitionSearchPagesResult{
				Items: []process.ProcessDefinition{{
					Key:           "2251799813685255",
					BpmnProcessId: "order-process",
					TenantId:      "<default>",
				}},
				Pages: 1,
			}, nil
		},
	}

	result, err := searchProcessDefinitionsWithPaging(cmd, cli, process.ProcessDefinitionFilter{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, []activitysink.Start{{
		Message:    "loading process-definition statistics",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.Starts())
	require.Equal(t, 1, sink.Stopped())
}

type processDefinitionPagingActivityAPI struct {
	c8volt.API
	searchProcessDefinitionsPages func(context.Context, process.ProcessDefinitionSearchRequest, process.ProcessDefinitionSearchPageVisitor, ...options.FacadeOption) (process.ProcessDefinitionSearchPagesResult, error)
}

func (a processDefinitionPagingActivityAPI) SearchProcessDefinitionsPages(ctx context.Context, request process.ProcessDefinitionSearchRequest, visitor process.ProcessDefinitionSearchPageVisitor, opts ...options.FacadeOption) (process.ProcessDefinitionSearchPagesResult, error) {
	return a.searchProcessDefinitionsPages(ctx, request, visitor, opts...)
}

type processDefinitionWatchTestAPI struct {
	c8volt.API
	collect func(context.Context, process.ProcessDefinitionWatchSnapshotRequest, ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error)
}

func (a processDefinitionWatchTestAPI) CollectProcessDefinitionWatchSnapshot(ctx context.Context, request process.ProcessDefinitionWatchSnapshotRequest, opts ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
	return a.collect(ctx, request, opts...)
}

func executeGetProcessDefinitionWatchForTest(t *testing.T, cli c8volt.API, filter process.ProcessDefinitionFilter, timeout time.Duration, sleep func(context.Context, time.Duration) error) (string, error) {
	t.Helper()

	stdout, _, err := executeGetProcessDefinitionWatchWithBackoffForTest(t, cli, filter, timeout, defaultBackoffMaxRetries, sleep)
	return stdout, err
}

func executeGetProcessDefinitionWatchWithBackoffForTest(t *testing.T, cli c8volt.API, filter process.ProcessDefinitionFilter, timeout time.Duration, maxRetries int, sleep func(context.Context, time.Duration) error) (string, string, error) {
	t.Helper()

	cmd := &cobra.Command{Use: "process-definition"}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(context.Background())

	previousSleep := processDefinitionWatchSleep
	processDefinitionWatchSleep = sleep
	t.Cleanup(func() {
		processDefinitionWatchSleep = previousSleep
	})

	err := executeGetProcessDefinitionWatch(cmd, cli, filter, timeout, maxRetries)
	return stdout.String(), stderr.String(), err
}

func resetGetProcessDefinitionCommandGlobals() {
	flagGetPDKey = ""
	flagGetPDBpmnProcessId = ""
	flagGetPDProcessVersion = 0
	flagGetPDProcessVersionTag = ""
	flagGetPDLatest = false
	flagGetPDWithStat = false
	flagGetPDAsXML = false
	flagGetPDBatchSize = 0
	flagGetPDWatch = false
	flagGetPDWatchInterval = defaultGetPDWatchInterval.String()
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagQuiet = false
	flagVerbose = false
	flagDebug = false
	flagCmdAutomation = false
}

// marshalStringSliceForEnv keeps subprocess argument fixtures shell-safe.
func marshalStringSliceForEnv(t *testing.T, items []string) string {
	t.Helper()

	data, err := json.Marshal(items)
	require.NoError(t, err)
	return string(data)
}

func TestGetProcessDefinitionLatestSearchPreservesSelectionRequestHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--json",
		"get", "process-definition",
		"--bpmn-process-id", "invoice",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
		"--latest",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnosticHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"get", "process-definition",
		"--bpmn-process-id", "missing-process",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookupHelper runs the
// real command path so validation exits the subprocess exactly as users see it.
func TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookupHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_PD_ARGS")), &args); err != nil {
		t.Fatalf("decode process-definition args: %v", err)
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = append([]string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...)

	Execute()
}

func executeRootForProcessDefinitionTestWithSeparateOutputs(t *testing.T, args ...string) (string, string) {
	t.Helper()

	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return stdout.String(), stderr.String()
}

func newProcessDefinitionSearchServerResponses(t *testing.T, requests *[]map[string]any, responses ...string) *httptest.Server {
	t.Helper()

	served := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var request map[string]any
		require.NoError(t, json.Unmarshal(body, &request))
		*requests = append(*requests, request)
		require.Less(t, served, len(responses), "unexpected extra process-definition search request")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[served]))
		served++
	}))
}

func countProcessDefinitionSearchRequests(items []string) int {
	count := 0
	for _, item := range items {
		if strings.HasPrefix(item, "POST /v2/process-definitions/search ") {
			count++
		}
	}
	return count
}
