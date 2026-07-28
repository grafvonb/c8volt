// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	defaultActivityDelay    = 350 * time.Millisecond
	defaultActivityInterval = 120 * time.Millisecond
)

var activityFrames = []string{"|", "/", "-", `\`}

type ActivitySink interface {
	StartActivity(msg string)
	StopActivity()
}

type ActivityUpdater interface {
	UpdateActivity(msg string)
}

type ActivityImportance = int

const (
	ActivityImportanceHTTP ActivityImportance = iota
	ActivityImportanceWait
	ActivityImportanceBatch
	ActivityImportanceWorkflow
)

type PriorityActivitySink interface {
	StartActivityWithImportance(msg string, importance ActivityImportance) func()
}

type PriorityActivityUpdater interface {
	UpdateActivityWithImportance(msg string, importance ActivityImportance)
}

type ActivityWriter interface {
	io.Writer
	ActivitySink
	ActivityUpdater
	PriorityActivitySink
	PriorityActivityUpdater
}

type activityCtxKey struct{}

type concurrentWriter interface {
	io.Writer
	isConcurrentWriter()
}

type activityWriter struct {
	w           io.Writer
	mu          sync.Mutex
	enabled     bool
	maxWidth    int
	scopes      map[int64]*activityScope
	legacyStack []int64
	nextScopeID int64
	nextOrder   int64
	drawn       bool
	drawnWidth  int
	frame       int
	stopCh      chan struct{}
	doneCh      chan struct{}
	delay       time.Duration
	interval    time.Duration
}

type activityScope struct {
	id         int64
	message    string
	importance ActivityImportance
	order      int64
}

func NewActivityWriter(w io.Writer) ActivityWriter {
	return NewActivityWriterEnabled(w, true)
}

func NewActivityWriterEnabled(w io.Writer, enabled bool) ActivityWriter {
	return newActivityWriter(w, enabled && isInteractiveTerminal(w))
}

func ToActivityContext(ctx context.Context, activity ActivitySink) context.Context {
	return context.WithValue(ctx, activityCtxKey{}, activity)
}

func ActivityFromContext(ctx context.Context) ActivitySink {
	if ctx == nil {
		return nil
	}
	activity, _ := ctx.Value(activityCtxKey{}).(ActivitySink)
	return activity
}

func StartActivity(ctx context.Context, msg string) func() {
	activity := ActivityFromContext(ctx)
	if activity == nil {
		return func() {}
	}
	activity.StartActivity(msg)
	return activity.StopActivity
}

func StartActivityWithImportance(ctx context.Context, msg string, importance ActivityImportance) func() {
	activity := ActivityFromContext(ctx)
	if activity == nil {
		return func() {}
	}
	if priorityActivity, ok := activity.(PriorityActivitySink); ok {
		return priorityActivity.StartActivityWithImportance(msg, importance)
	}
	activity.StartActivity(msg)
	return activity.StopActivity
}

func UpdateActivity(ctx context.Context, msg string) {
	activity, _ := ActivityFromContext(ctx).(ActivityUpdater)
	if activity == nil {
		return
	}
	activity.UpdateActivity(msg)
}

func UpdateActivityWithImportance(ctx context.Context, msg string, importance ActivityImportance) {
	activity := ActivityFromContext(ctx)
	if activity == nil {
		return
	}
	if priorityActivity, ok := activity.(PriorityActivityUpdater); ok {
		priorityActivity.UpdateActivityWithImportance(msg, importance)
		return
	}
	updater, _ := activity.(ActivityUpdater)
	if updater == nil {
		return
	}
	updater.UpdateActivity(msg)
}

// newActivityWriter builds an activity writer with default spinner timing.
func newActivityWriter(w io.Writer, enabled bool) *activityWriter {
	return &activityWriter{
		w:        w,
		enabled:  enabled,
		maxWidth: activityWriterMaxWidth(w),
		delay:    defaultActivityDelay,
		interval: defaultActivityInterval,
	}
}

func (w *activityWriter) isConcurrentWriter() {}

func (w *activityWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.clearLocked()
	return w.w.Write(p)
}

func (w *activityWriter) StartActivity(msg string) {
	if w == nil || !w.enabled {
		return
	}

	w.startActivity(msg, ActivityImportanceBatch, true)
}

func (w *activityWriter) StartActivityWithImportance(msg string, importance ActivityImportance) func() {
	if w == nil || !w.enabled {
		return func() {}
	}

	id := w.startActivity(msg, importance, false)
	var once sync.Once
	return func() {
		once.Do(func() {
			w.stopActivityID(id)
		})
	}
}

func (w *activityWriter) startActivity(msg string, importance ActivityImportance, legacy bool) int64 {
	w.mu.Lock()
	if w.scopes == nil {
		w.scopes = make(map[int64]*activityScope)
	}
	previousSelected := w.selectedScopeLocked()
	w.nextScopeID++
	w.nextOrder++
	id := w.nextScopeID
	w.scopes[id] = &activityScope{
		id:         id,
		message:    strings.TrimSpace(msg),
		importance: importance,
		order:      w.nextOrder,
	}
	if legacy {
		w.legacyStack = append(w.legacyStack, id)
	}
	startSpinner := len(w.scopes) == 1
	if startSpinner {
		w.drawn = false
		w.drawnWidth = 0
		w.frame = 0
		w.stopCh = make(chan struct{})
		w.doneCh = make(chan struct{})
	}
	shouldRedraw := w.drawn && previousSelected != nil && previousSelected.id != w.selectedScopeLocked().id
	stopCh := w.stopCh
	doneCh := w.doneCh
	delay := w.delay
	interval := w.interval
	if shouldRedraw {
		w.drawLocked()
	}
	w.mu.Unlock()

	if !startSpinner {
		return id
	}

	go func() {
		defer close(doneCh)

		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-stopCh:
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		w.tick()
		for {
			select {
			case <-ticker.C:
				w.tick()
			case <-stopCh:
				return
			}
		}
	}()
	return id
}

func (w *activityWriter) StopActivity() {
	if w == nil || !w.enabled {
		return
	}

	id, ok := w.popLegacyActivityID()
	if !ok {
		return
	}
	w.stopActivityID(id)
}

func (w *activityWriter) popLegacyActivityID() (int64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for len(w.legacyStack) > 0 {
		last := len(w.legacyStack) - 1
		id := w.legacyStack[last]
		w.legacyStack = w.legacyStack[:last]
		if _, ok := w.scopes[id]; ok {
			return id, true
		}
	}
	return 0, false
}

func (w *activityWriter) stopActivityID(id int64) {
	if w == nil || !w.enabled {
		return
	}

	w.mu.Lock()
	if _, ok := w.scopes[id]; !ok {
		w.mu.Unlock()
		return
	}
	previousSelected := w.selectedScopeLocked()
	delete(w.scopes, id)
	if len(w.scopes) > 0 {
		if w.drawn && previousSelected != nil && previousSelected.id == id {
			w.drawLocked()
		}
		w.mu.Unlock()
		return
	}
	stopCh := w.stopCh
	doneCh := w.doneCh
	w.stopCh = nil
	w.doneCh = nil
	w.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	if doneCh != nil {
		<-doneCh
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.clearLocked()
}

func (w *activityWriter) UpdateActivity(msg string) {
	if w == nil || !w.enabled {
		return
	}

	w.updateActivity(strings.TrimSpace(msg), nil)
}

func (w *activityWriter) UpdateActivityWithImportance(msg string, importance ActivityImportance) {
	if w == nil || !w.enabled {
		return
	}

	w.updateActivity(strings.TrimSpace(msg), &importance)
}

func (w *activityWriter) updateActivity(msg string, importance *ActivityImportance) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.scopes) == 0 {
		return
	}
	scope := w.scopeForUpdateLocked(importance)
	if scope == nil {
		return
	}
	scope.message = msg
	if selected := w.selectedScopeLocked(); w.drawn && selected != nil && selected.id == scope.id {
		w.drawLocked()
	}
}

func (w *activityWriter) tick() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.scopes) == 0 {
		return
	}

	w.drawLocked()
}

func (w *activityWriter) drawLocked() {
	frame := activityFrames[w.frame%len(activityFrames)]
	w.frame++
	line := frame
	if scope := w.selectedScopeLocked(); scope != nil && scope.message != "" {
		line += " " + scope.message
	}
	line = truncateActivityLine(line, w.maxWidth)
	width := activityLineWidth(w.drawnWidth, len(line), w.maxWidth)
	_, _ = fmt.Fprintf(w.w, "\r%s", padRight(line, width))
	w.drawn = true
	w.drawnWidth = width
}

func (w *activityWriter) clearLocked() {
	if !w.drawn {
		return
	}
	_, _ = fmt.Fprintf(w.w, "\r%s\r", strings.Repeat(" ", activityClearWidth(w.drawnWidth, w.maxWidth)))
	w.drawn = false
	w.drawnWidth = 0
}

func (w *activityWriter) selectedScopeLocked() *activityScope {
	var selected *activityScope
	for _, scope := range w.scopes {
		if selected == nil || scope.importance > selected.importance ||
			(scope.importance == selected.importance && scope.order > selected.order) {
			selected = scope
		}
	}
	return selected
}

func (w *activityWriter) scopeForUpdateLocked(importance *ActivityImportance) *activityScope {
	var selected *activityScope
	for _, scope := range w.scopes {
		if importance != nil && scope.importance != *importance {
			continue
		}
		if selected == nil || scope.order > selected.order {
			selected = scope
		}
	}
	if selected != nil || importance == nil {
		return selected
	}
	return w.selectedScopeLocked()
}

func activityLineWidth(previous int, current int, maxWidth int) int {
	width := max(previous, current)
	if maxWidth > 0 {
		return min(maxWidth, max(1, width))
	}
	return max(80, width)
}

func activityClearWidth(drawnWidth int, maxWidth int) int {
	if maxWidth > 0 {
		return min(maxWidth, max(1, drawnWidth))
	}
	return max(80, drawnWidth)
}

func truncateActivityLine(line string, maxWidth int) string {
	if maxWidth <= 0 || len(line) <= maxWidth {
		return line
	}
	if maxWidth <= 3 {
		return line[:maxWidth]
	}
	return line[:maxWidth-3] + "..."
}

func padRight(s string, width int) string {
	if width <= len(s) {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func isInteractiveTerminal(w io.Writer) bool {
	type fdWriter interface {
		Fd() uintptr
	}
	f, ok := w.(fdWriter)
	if !ok {
		return false
	}
	if _, ok := w.(*os.File); !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func activityWriterMaxWidth(w io.Writer) int {
	type fdWriter interface {
		Fd() uintptr
	}
	f, ok := w.(fdWriter)
	if !ok {
		return 0
	}
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil || width <= 0 {
		return 0
	}
	if width > 1 {
		return width - 1
	}
	return width
}
