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

type ActivityWriter interface {
	io.Writer
	ActivitySink
	ActivityUpdater
}

type activityCtxKey struct{}

type concurrentWriter interface {
	io.Writer
	isConcurrentWriter()
}

type activityWriter struct {
	w          io.Writer
	mu         sync.Mutex
	enabled    bool
	maxWidth   int
	refs       int
	active     bool
	drawn      bool
	drawnWidth int
	frame      int
	stopCh     chan struct{}
	doneCh     chan struct{}
	message    string
	delay      time.Duration
	interval   time.Duration
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

func UpdateActivity(ctx context.Context, msg string) {
	activity, _ := ActivityFromContext(ctx).(ActivityUpdater)
	if activity == nil {
		return
	}
	activity.UpdateActivity(msg)
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

	w.mu.Lock()
	w.refs++
	if w.active {
		w.mu.Unlock()
		return
	}
	w.active = true
	w.drawn = false
	w.drawnWidth = 0
	w.frame = 0
	w.message = strings.TrimSpace(msg)
	w.stopCh = make(chan struct{})
	w.doneCh = make(chan struct{})
	stopCh := w.stopCh
	doneCh := w.doneCh
	delay := w.delay
	interval := w.interval
	w.mu.Unlock()

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
}

func (w *activityWriter) StopActivity() {
	if w == nil || !w.enabled {
		return
	}

	w.mu.Lock()
	if w.refs == 0 {
		w.mu.Unlock()
		return
	}
	w.refs--
	if w.refs > 0 || !w.active {
		w.mu.Unlock()
		return
	}
	stopCh := w.stopCh
	doneCh := w.doneCh
	w.active = false
	w.stopCh = nil
	w.doneCh = nil
	w.mu.Unlock()

	close(stopCh)
	<-doneCh

	w.mu.Lock()
	defer w.mu.Unlock()
	w.clearLocked()
}

func (w *activityWriter) UpdateActivity(msg string) {
	if w == nil || !w.enabled {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active {
		return
	}
	w.message = strings.TrimSpace(msg)
	if !w.drawn {
		return
	}
	w.drawLocked()
}

func (w *activityWriter) tick() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active {
		return
	}

	w.drawLocked()
}

func (w *activityWriter) drawLocked() {
	frame := activityFrames[w.frame%len(activityFrames)]
	w.frame++
	line := frame
	if w.message != "" {
		line += " " + w.message
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
