// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx/watch"
	"github.com/spf13/cobra"
)

const defaultGetPDWatchInterval = time.Second

var processDefinitionWatchSleep watch.SleepFunc
var processDefinitionWatchNow = time.Now

// runGetProcessDefinitionWatch executes the focused watch mode and converts
// watch lifecycle failures through the command error path.
func runGetProcessDefinitionWatch(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, filter process.ProcessDefinitionFilter, timeout time.Duration, maxRetries int) {
	log.Debug(fmt.Sprintf("watching pd; filter %s", filter.String()))
	if err := executeGetProcessDefinitionWatch(cmd, cli, filter, timeout, maxRetries); err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("watch process definitions: %w", err))
	}
}

// executeGetProcessDefinitionWatch collects serial refreshes and repaints stdout after each successful collection.
func executeGetProcessDefinitionWatch(cmd *cobra.Command, cli c8volt.API, filter process.ProcessDefinitionFilter, timeout time.Duration, maxRetries int) error {
	interval, err := resolveGetProcessDefinitionWatchInterval()
	if err != nil {
		return err
	}
	request := newGetProcessDefinitionWatchSnapshotRequest(filter)
	opts := collectOptions()
	if request.Key != "" {
		opts = collectExplicitAdminInputOptions()
	}
	var slowWarnings processDefinitionWatchSlowWarningState
	result, err := watch.Run(cmd.Context(), watch.Options{
		Interval:   interval,
		Timeout:    timeout,
		MaxRetries: maxRetries,
		Retryable:  isGetProcessDefinitionWatchRetryable,
		Sleep:      processDefinitionWatchSleep,
		OnRetry: func(event watch.RetryEvent) {
			renderGetProcessDefinitionWatchRetryStatus(cmd, event)
		},
	}, func(ctx context.Context, tick watch.Tick) error {
		startedAt := processDefinitionWatchNow()
		snapshot, err := cli.CollectProcessDefinitionWatchSnapshot(ctx, request, opts...)
		if err != nil {
			return err
		}
		if err := renderTerminalRepaint(cmd); err != nil {
			return err
		}
		if err := processDefinitionWatchView(cmd, snapshot); err != nil {
			return err
		}
		duration := processDefinitionWatchNow().Sub(startedAt)
		renderGetProcessDefinitionWatchRefreshStatus(cmd, &slowWarnings, tick.Index, duration, interval)
		return nil
	})
	renderGetProcessDefinitionWatchStopStatus(cmd, result)
	return err
}

// processDefinitionWatchSlowWarningState suppresses repeated slow-refresh
// warnings until the watch runner observes an on-time refresh.
type processDefinitionWatchSlowWarningState struct {
	active bool
}

// renderGetProcessDefinitionWatchRefreshStatus reports slow-refresh warnings
// and verbose refresh duration diagnostics on stderr.
func renderGetProcessDefinitionWatchRefreshStatus(cmd *cobra.Command, state *processDefinitionWatchSlowWarningState, refresh int64, duration, interval time.Duration) {
	if cmd == nil {
		return
	}
	slow := duration > interval
	if slow && state != nil && !state.active {
		fmt.Fprintf(cmd.ErrOrStderr(), "slow process-definition watch refresh %d: took %s, exceeding --watch-interval %s; suppressing repeated slow-refresh warnings until a refresh completes within the interval\n", refresh, duration, interval)
	}
	if state != nil {
		state.active = slow
	}
	if flagVerbose {
		status := "on-time"
		if slow {
			status = "slow"
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "process-definition watch refresh %d completed in %s (interval %s, status: %s)\n", refresh, duration, interval, status)
	}
}

// resolveGetProcessDefinitionWatchInterval parses the watch interval flag into
// the cadence used by the refresh loop.
func resolveGetProcessDefinitionWatchInterval() (time.Duration, error) {
	interval, err := time.ParseDuration(flagGetPDWatchInterval)
	if err != nil {
		return 0, invalidFlagValuef("invalid value for --watch-interval: %q, expected positive duration such as 1s, 2s, or 500ms", flagGetPDWatchInterval)
	}
	if interval <= 0 {
		return 0, invalidFlagValuef("invalid value for --watch-interval: %q, expected positive duration", flagGetPDWatchInterval)
	}
	return interval, nil
}

// isGetProcessDefinitionWatchRetryable limits watch retries to transient
// timeout and availability failures.
func isGetProcessDefinitionWatchRetryable(err error) bool {
	switch ferrors.Classify(err) {
	case ferrors.ClassTimeout, ferrors.ClassUnavailable:
		return true
	default:
		return false
	}
}

// renderGetProcessDefinitionWatchRetryStatus keeps retry diagnostics off the repainted result body.
func renderGetProcessDefinitionWatchRetryStatus(cmd *cobra.Command, event watch.RetryEvent) {
	if cmd == nil {
		return
	}
	if event.MaxRetries > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "retrying process-definition watch after refresh %d failed (%d/%d consecutive failures): %v\n", event.Tick, event.ConsecutiveFailures, event.MaxRetries, event.Err)
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "retrying process-definition watch after refresh %d failed (consecutive failures: %d): %v\n", event.Tick, event.ConsecutiveFailures, event.Err)
}

// renderGetProcessDefinitionWatchStopStatus reports terminal watch stop reasons on stderr.
func renderGetProcessDefinitionWatchStopStatus(cmd *cobra.Command, result watch.Result) {
	if cmd == nil {
		return
	}
	switch result.Reason {
	case watch.TerminationReasonTimeout:
		fmt.Fprintln(cmd.ErrOrStderr(), "watch stopped: timeout reached")
	case watch.TerminationReasonRetryExhausted:
		fmt.Fprintf(cmd.ErrOrStderr(), "watch stopped: retry budget exhausted after %d consecutive failure(s)\n", result.ConsecutiveFailures)
	}
}

// newGetProcessDefinitionWatchSnapshotRequest preserves selector state while
// enabling broad watch discovery only when no selector is active.
func newGetProcessDefinitionWatchSnapshotRequest(filter process.ProcessDefinitionFilter) process.ProcessDefinitionWatchSnapshotRequest {
	request := process.ProcessDefinitionWatchSnapshotRequest{
		Key:    filter.Key,
		Filter: filter,
		Page: process.ProcessDefinitionPageRequest{
			Size: resolveGetProcessDefinitionSearchSize(),
		},
		Latest: flagGetPDLatest,
	}
	request.WatchAllWhenUnselected = request.Key == "" &&
		filter.BpmnProcessId == "" &&
		filter.ProcessVersion == 0 &&
		filter.ProcessVersionTag == "" &&
		!request.Latest
	return request
}
