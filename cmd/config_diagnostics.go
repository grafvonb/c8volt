// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/config"
)

func logConfigProfile(log interface{ Info(string, ...any) }, cfg *config.Config) {
	if cfg != nil && cfg.ActiveProfile != "" {
		log.Info("using configuration profile: " + cfg.ActiveProfile)
		return
	}
	log.Info("no active profile provided in configuration, using default settings")
}

func camundaMajorMinorMismatchWarnings(configuredVersion string, gatewayVersion string) []string {
	configuredMajorMinor, configuredOK := parseMajorMinorVersion(configuredVersion)
	gatewayMajorMinor, gatewayOK := parseMajorMinorVersion(gatewayVersion)
	if !configuredOK || !gatewayOK || configuredMajorMinor == gatewayMajorMinor {
		return nil
	}
	return []string{fmt.Sprintf("configured Camunda version %s differs from gateway version %s by major/minor version; this can cause unexpected errors because Camunda APIs can differ between versions; correct the configured version unless there is a very good reason to keep this mismatch", configuredVersion, gatewayVersion)}
}

func parseMajorMinorVersion(version string) (string, bool) {
	version = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(version), "v"))
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return "", false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%d.%d", major, minor), true
}
