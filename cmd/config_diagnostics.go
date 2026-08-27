// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/config"
)

type camundaReleaseLineComparison int

const (
	camundaReleaseLineMatch camundaReleaseLineComparison = iota
	camundaReleaseLineMismatch
	camundaReleaseLineUnrecognizable
)

func logConfigProfile(log interface{ Info(string, ...any) }, cfg *config.Config) {
	if cfg != nil && cfg.ActiveProfile != "" {
		log.Info("config profile " + cfg.ActiveProfile)
		return
	}
	log.Info("config profile default")
}

// camundaReleaseLineCompatibilityWarnings reports gateway release-line mismatches that can affect API compatibility.
func camundaReleaseLineCompatibilityWarnings(configuredVersion string, gatewayVersion string) []string {
	switch compareCamundaReleaseLines(configuredVersion, gatewayVersion) {
	case camundaReleaseLineMatch:
		return nil
	case camundaReleaseLineMismatch:
		return []string{fmt.Sprintf("configured Camunda version %s differs from gateway version %s by major/minor version; this can cause unexpected errors because Camunda APIs can differ between versions; correct the configured version unless there is a very good reason to keep this mismatch", configuredVersion, gatewayVersion)}
	default:
		if strings.TrimSpace(gatewayVersion) == "" {
			return []string{fmt.Sprintf("gateway version is empty; cannot verify compatibility with configured Camunda version %s", configuredVersion)}
		}
		return []string{fmt.Sprintf("gateway version %q is not recognizable as a Camunda major/minor version; cannot verify compatibility with configured Camunda version %s", gatewayVersion, configuredVersion)}
	}
}

// compareCamundaReleaseLines compares only the configured and observed major/minor release lines.
func compareCamundaReleaseLines(configuredVersion string, gatewayVersion string) camundaReleaseLineComparison {
	configuredMajorMinor, configuredOK := parseMajorMinorVersion(configuredVersion)
	gatewayMajorMinor, gatewayOK := parseMajorMinorVersion(gatewayVersion)
	if !configuredOK || !gatewayOK {
		return camundaReleaseLineUnrecognizable
	}
	if configuredMajorMinor != gatewayMajorMinor {
		return camundaReleaseLineMismatch
	}
	return camundaReleaseLineMatch
}

// parseMajorMinorVersion extracts the comparable Camunda release line from normalized and gateway version strings.
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
