// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// readUserTaskKeys reads explicit dash input or, for this command only, an
// available nonterminal stdin stream. Terminal stdin is never consumed to infer
// keyed mode.
func readUserTaskKeys(args []string) ([]string, error) {
	explicit := len(args) == 1 && args[0] == "-"
	if !explicit && term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, nil
	}
	if explicit && term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, invalidFlagValuef("'-' requires piped/redirected stdin (example: printf 'k1\\nk2\\n' | c8volt <cmd> -)")
	}
	return scanUserTaskKeys(os.Stdin, explicit)
}

// scanUserTaskKeys preserves the shared scanner ceiling and blank-line rules,
// while allowing an empty implicit stream to select ordinary search mode.
func scanUserTaskKeys(reader io.Reader, explicit bool) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var keys []string
	for scanner.Scan() {
		if key := strings.TrimSpace(scanner.Text()); key != "" {
			keys = append(keys, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if explicit && len(keys) == 0 {
		return nil, invalidFlagValuef("stdin contained no keys")
	}
	return keys, nil
}
