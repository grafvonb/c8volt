// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"time"
)

type PlainHandler struct {
	w             io.Writer
	level         slog.Leveler
	withTimestamp bool
	withSource    bool
}

func NewPlainHandler(w io.Writer, level slog.Leveler) *PlainHandler {
	return &PlainHandler{w: w, level: level}
}

func (h *PlainHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level.Level()
}

func (h *PlainHandler) Handle(_ context.Context, r slog.Record) error {
	level := r.Level.String()
	line := fmt.Sprintf("%s %s", level, r.Message)
	if h.withTimestamp {
		ts := r.Time.Format(time.RFC3339)
		line = fmt.Sprintf("%s %s", ts, line)
	}
	if h.withSource && r.PC != 0 {
		fs := r.Source()
		line = fmt.Sprintf("%s (%s:%d)", line, filepath.Base(fs.File), fs.Line)
	}
	_, err := fmt.Fprintln(h.w, line)
	return err
}

func (h *PlainHandler) WithTimestamp(b bool) *PlainHandler {
	h.withTimestamp = b
	return h
}

func (h *PlainHandler) WithSource(b bool) *PlainHandler {
	h.withSource = b
	return h
}

func (h *PlainHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *PlainHandler) WithGroup(name string) slog.Handler       { return h }
