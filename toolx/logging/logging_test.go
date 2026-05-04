// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package logging

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPlainLoggerIncludesFullLocalTimestampWithMilliseconds(t *testing.T) {
	var buf bytes.Buffer
	log := New(LoggerConfig{
		Level:  "info",
		Format: "plain",
		Writer: &buf,
	})

	log.Info("config loaded")

	require.Regexp(t, regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}(Z|[+-]\d{2}:\d{2}) INFO config loaded\n$`), buf.String())
}

func TestNewPlainTimeLoggerIncludesTimeWithMilliseconds(t *testing.T) {
	var buf bytes.Buffer
	log := New(LoggerConfig{
		Level:  "info",
		Format: "plain-time",
		Writer: &buf,
	})

	log.Info("config loaded")

	require.Regexp(t, regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\.\d{3} INFO config loaded\n$`), buf.String())
	require.Contains(t, buf.String(), " INFO config loaded\n")
}

func TestNewLoggerDefaultsToPlainTimeFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(LoggerConfig{
		Level:  "info",
		Writer: &buf,
	})

	log.Info("config loaded")

	require.Regexp(t, regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\.\d{3} INFO config loaded\n$`), buf.String())
}
