// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/integration/internal/validation"
	"gopkg.in/yaml.v3"
)

type tenantConfigFile struct {
	App      tenantConfigApp                `yaml:"app"`
	APIs     config.APIs                    `yaml:"apis"`
	Profiles map[string]tenantConfigProfile `yaml:"profiles"`
}

type tenantConfigProfile struct {
	App  tenantConfigApp `yaml:"app"`
	APIs config.APIs     `yaml:"apis"`
}

type tenantConfigApp struct {
	CamundaVersion string `yaml:"camunda_version"`
	Tenant         string `yaml:"tenant"`
}

// loadDefaultTenantProfileMetadata reads only version and tenant identifiers
// from the standard local config search path; credentials are never retained.
func loadDefaultTenantProfileMetadata(env map[string]string) (map[string]validation.TenantProfileMetadata, error) {
	paths, err := defaultTenantConfigPaths(env)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if errors.Is(readErr, os.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("read default local config %s: %w", path, readErr)
		}
		var raw tenantConfigFile
		if unmarshalErr := yaml.Unmarshal(data, &raw); unmarshalErr != nil {
			return nil, fmt.Errorf("parse default local config %s: %w", path, unmarshalErr)
		}
		return tenantProfileMetadata(raw)
	}
	return nil, fmt.Errorf("default local c8volt config not found in %s", strings.Join(paths, ", "))
}

// defaultTenantConfigPaths mirrors default-local profile discovery without
// accepting a generated or role-file-supplied configuration path.
func defaultTenantConfigPaths(env map[string]string) ([]string, error) {
	var paths []string
	if xdg := strings.TrimSpace(env["XDG_CONFIG_HOME"]); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "c8volt", "config.yaml"))
	}
	if home := strings.TrimSpace(env["HOME"]); home != "" {
		paths = append(paths,
			filepath.Join(home, ".config", "c8volt", "config.yaml"),
			filepath.Join(home, ".c8volt", "config.yaml"),
		)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	paths = append(paths, filepath.Join(workingDirectory, "config.yaml"), "/etc/c8volt/config.yaml")
	return paths, nil
}

// tenantProfileMetadata applies base inheritance for the two fields used by
// logical role validation and rejects unparseable configured versions.
func tenantProfileMetadata(raw tenantConfigFile) (map[string]validation.TenantProfileMetadata, error) {
	result := make(map[string]validation.TenantProfileMetadata, len(raw.Profiles))
	for name, profile := range raw.Profiles {
		version := strings.TrimSpace(profile.App.CamundaVersion)
		if version == "" {
			version = strings.TrimSpace(raw.App.CamundaVersion)
		}
		minor, err := configuredMinor(version)
		if err != nil {
			return nil, fmt.Errorf("profile %q Camunda version: %w", name, err)
		}
		tenant := profile.App.Tenant
		if tenant == "" {
			tenant = raw.App.Tenant
		}
		apis := raw.APIs
		if profile.APIs.Camunda.BaseURL != "" {
			apis.Camunda.BaseURL = profile.APIs.Camunda.BaseURL
		}
		if profile.APIs.Camunda.Version != "" {
			apis.Camunda.Version = profile.APIs.Camunda.Version
		}
		if err := apis.Normalize(); err != nil {
			return nil, fmt.Errorf("profile %q Camunda API: %w", name, err)
		}
		result[name] = validation.TenantProfileMetadata{
			Name: name, ExpectedMinor: minor, LogicalTenantID: tenant, CamundaBaseURL: apis.Camunda.BaseURL,
		}
	}
	return result, nil
}

// configuredMinor accepts canonical minor or full numeric gateway versions.
func configuredMinor(value string) (string, error) {
	if minor, err := validation.CanonicalMinor(value); err == nil {
		return minor, nil
	}
	return validation.ObservedMinor(value)
}
