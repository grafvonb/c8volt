// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

func validateKeys(keys []string) (ok bool, firstBadKey string, firstBadIndex int) {
	keyRE := regexp.MustCompile(`^\d{16}$`)

	for i, k := range keys {
		if !keyRE.MatchString(strings.TrimSpace(k)) {
			ok, firstBadKey, firstBadIndex = false, k, i
			return
		}
	}
	ok, firstBadKey, firstBadIndex = true, "", -1
	return
}

func validateOptionalDashArg(args []string) error {
	if len(args) == 0 {
		return nil
	}
	if len(args) == 1 && args[0] == "-" {
		return nil
	}
	return invalidFlagValuef("unexpected args: %v (use '-' to read keys from stdin)", args)
}

func readKeysIfDash(args []string) ([]string, error) {
	if len(args) != 1 || args[0] != "-" {
		return nil, nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, invalidFlagValuef("'-' requires piped/redirected stdin (example: printf 'k1\\nk2\\n' | c8volt <cmd> -)")
	}

	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var out []string
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s != "" {
			out = append(out, s)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, invalidFlagValuef("stdin contained no keys")
	}
	return out, nil
}
