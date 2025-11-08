package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
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

// reads newline-separated keys from stdin if piped; returns nil if stdin is a TTY
func readKeysFromStdin() ([]string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	isTTY := (info.Mode() & os.ModeCharDevice) != 0
	if isTTY {
		return nil, nil // not piped
	}
	sc := bufio.NewScanner(os.Stdin)
	// allow long lines if needed
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 10*1024*1024)

	var out []string
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s != "" {
			out = append(out, s)
		}
	}
	if err = sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("stdin was piped but contained no keys")
	}
	return out, nil
}
