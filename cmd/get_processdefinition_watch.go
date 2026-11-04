// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx/watch"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const defaultGetPDWatchInterval = time.Second

var processDefinitionWatchSleep watch.SleepFunc
var processDefinitionWatchNow = time.Now

// runGetProcessDefinitionWatch executes the focused watch mode and converts
// watch lifecycle failures through the command error path.
func runGetProcessDefinitionWatch(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, filter process.ProcessDefinitionFilter, timeout time.Duration, maxRetries int) {
	log.Debug(fmt.Sprintf("watching pd; filter %s", filter.String()))
	if err := executeGetProcessDefinitionWatch(cmd, cli, filter, timeout, maxRetries); err != nil {
		message := fmt.Sprintf("watch process definitions: stopped (%s)", strings.ReplaceAll(string(ferrors.Classify(err)), "_", " "))
		var failure *processDefinitionWatchFailure
		if errors.As(err, &failure) && failure.result.Reason == watch.TerminationReasonRetryExhausted {
			message = fmt.Sprintf("watch process definitions: retry budget exhausted after %d consecutive failures (%s)", failure.result.ConsecutiveFailures, ferrors.Classify(err))
		}
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("watch process definitions: %w", err),
			message)
	}
}

// processDefinitionWatchFailure carries bounded session facts for human errors
// while retaining the original error text and class for machine consumers.
type processDefinitionWatchFailure struct {
	result watch.Result
	cause  error
}

// Error preserves the runner's evidence without adding another error prefix.
func (e *processDefinitionWatchFailure) Error() string { return e.cause.Error() }

// Unwrap preserves timeout, availability and other original error classes.
func (e *processDefinitionWatchFailure) Unwrap() error { return e.cause }

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
	var baseline processDefinitionWatchBaseline
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
		if err := renderProcessDefinitionWatchRefresh(cmd, &baseline, snapshot); err != nil {
			return err
		}
		duration := processDefinitionWatchNow().Sub(startedAt)
		renderGetProcessDefinitionWatchRefreshStatus(cmd, &slowWarnings, tick.Index, duration, interval)
		return nil
	})
	renderGetProcessDefinitionWatchStopStatus(cmd, result)
	if err != nil {
		return &processDefinitionWatchFailure{result: result, cause: err}
	}
	return err
}

// renderProcessDefinitionWatchRefresh commits display history only after the
// complete candidate body, including repaint, has reached the configured writer.
func renderProcessDefinitionWatchRefresh(cmd *cobra.Command, baseline *processDefinitionWatchBaseline, snapshot process.ProcessDefinitionWatchSnapshot) error {
	candidate := baseline.candidate(snapshot.Items)
	if err := renderTerminalRepaint(cmd); err != nil {
		return err
	}
	if err := processDefinitionWatchView(cmd, candidate, processDefinitionWatchBoldEnabled(cmd)); err != nil {
		return err
	}
	baseline.commit(candidate)
	return nil
}

// processDefinitionWatchBoldEnabled applies the conservative watch styling
// policy to the command's actual configured or inherited stdout destination.
func processDefinitionWatchBoldEnabled(cmd *cobra.Command) bool {
	if cmd == nil || os.Getenv("NO_COLOR") != "" {
		return false
	}
	terminalName := strings.TrimSpace(os.Getenv("TERM"))
	if terminalName == "" || strings.EqualFold(terminalName, "dumb") {
		return false
	}
	writer, ok := cmd.OutOrStdout().(interface{ Fd() uintptr })
	return ok && term.IsTerminal(int(writer.Fd()))
}

// processDefinitionWatchBaseline distinguishes an empty successful baseline
// from a session that has not displayed any successful refresh yet.
type processDefinitionWatchBaseline struct {
	initialized bool
	byKey       map[string]processDefinitionDisplay
}

// candidate compares a complete snapshot without mutating acknowledged history.
func (b processDefinitionWatchBaseline) candidate(items []process.ProcessDefinition) processDefinitionWatchCandidate {
	candidate := processDefinitionWatchCandidate{
		rows:      make([]processDefinitionWatchRow, 0, len(items)),
		nextByKey: make(map[string]processDefinitionDisplay, len(items)),
	}
	for _, item := range items {
		display := newProcessDefinitionDisplay(item)
		candidate.nextByKey[display.key] = display
		row := processDefinitionWatchRow{display: display, marker: ' '}
		if b.initialized {
			previous, found := b.byKey[display.key]
			if !found {
				row.marker = '+'
				candidate.new++
			} else {
				row.changed = display.changedFields(previous)
				if row.changed.any() {
					row.marker = '*'
					candidate.updated++
				}
			}
		}
		candidate.rows = append(candidate.rows, row)
	}
	if b.initialized {
		for key := range b.byKey {
			if _, found := candidate.nextByKey[key]; !found {
				candidate.removed++
			}
		}
	}
	return candidate
}

// commit acknowledges a candidate only after repaint and view writes succeed.
func (b *processDefinitionWatchBaseline) commit(candidate processDefinitionWatchCandidate) {
	b.initialized = true
	b.byKey = candidate.nextByKey
}

// processDefinitionWatchSlowWarningState suppresses repeated slow-refresh
// warnings until the watch runner observes an on-time refresh.
type processDefinitionWatchSlowWarningState struct {
	active bool
}

// renderGetProcessDefinitionWatchRefreshStatus reports slow-refresh warnings
// and verbose refresh duration diagnostics through the configured logger.
func renderGetProcessDefinitionWatchRefreshStatus(cmd *cobra.Command, state *processDefinitionWatchSlowWarningState, refresh int64, duration, interval time.Duration) {
	if cmd == nil {
		return
	}
	slow := duration > interval
	if slow && state != nil && !state.active {
		renderHumanWarningLine(cmd, "slow process-definition watch refresh %d: took %s, exceeding --watch-interval %s; suppressing repeated slow-refresh warnings until a refresh completes within the interval", refresh, duration, interval)
	}
	if state != nil {
		state.active = slow
	}
	if flagVerbose {
		status := "on-time"
		if slow {
			status = "slow"
		}
		renderHumanLine(cmd, "process-definition watch refresh %d completed in %s (interval %s, status: %s)", refresh, duration, interval, status)
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
		renderHumanWarningLine(cmd, "retrying process-definition watch after refresh %d failed (%d/%d consecutive failures): %s", event.Tick, event.ConsecutiveFailures, event.MaxRetries, ferrors.Classify(event.Err))
		return
	}
	renderHumanWarningLine(cmd, "retrying process-definition watch after refresh %d failed (consecutive failures: %d): %s", event.Tick, event.ConsecutiveFailures, ferrors.Classify(event.Err))
}

// renderGetProcessDefinitionWatchStopStatus reports terminal watch stop reasons on stderr.
func renderGetProcessDefinitionWatchStopStatus(cmd *cobra.Command, result watch.Result) {
	if cmd == nil {
		return
	}
	switch result.Reason {
	case watch.TerminationReasonTimeout:
		renderHumanLine(cmd, "watch stopped: timeout reached")
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
