// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	CamundaApiKeyConst      = "camunda_api"
	CamundaApiVersionConst  = "v2"
	OperateApiKeyConst      = "operate_api"
	OperateApiVersionConst  = "v1"
	TasklistApiKeyConst     = "tasklist_api"
	TasklistApiVersionConst = "v1"
)

type APIs struct {
	Camunda           API  `mapstructure:"camunda_api" json:"camunda_api" yaml:"camunda_api"`
	Operate           API  `mapstructure:"operate_api" json:"operate_api" yaml:"operate_api"`
	Tasklist          API  `mapstructure:"tasklist_api" json:"tasklist_api" yaml:"tasklist_api"`
	VersioningDisable bool `mapstructure:"versioning_disable" json:"versioning_disable" yaml:"versioning_disable"`
	warnings          []string
}

type API struct {
	Key          string `mapstructure:"key" json:"key" yaml:"key"`
	BaseURL      string `mapstructure:"base_url" json:"base_url" yaml:"base_url"`
	RequireScope bool   `mapstructure:"require_scope" json:"require_scope" yaml:"require_scope"`
	Version      string `mapstructure:"version" json:"version" yaml:"version"`
}

func (a *APIs) Normalize() error {
	var errs []error
	normalizeAPIs(a, func(string) bool { return false })
	return errors.Join(errs...)
}

func (a *APIs) normalizeWithConfiguredKeys(isConfigured func(string) bool) error {
	var errs []error
	normalizeAPIs(a, isConfigured)
	return errors.Join(errs...)
}

func normalizeAPIs(a *APIs, isConfigured func(string) bool) {
	if isConfigured == nil {
		isConfigured = func(string) bool { return false }
	}
	a.warnings = nil
	if a.Camunda.Key == "" && !isConfigured("apis.camunda_api.key") {
		a.Camunda.Key = CamundaApiKeyConst
	}
	camundaBaseConfigured := a.Camunda.BaseURL != "" || isConfigured("apis.camunda_api.base_url")
	if a.Camunda.Version == "" && !isConfigured("apis.camunda_api.version") {
		a.Camunda.Version = CamundaApiVersionConst
	}
	if a.Operate.Key == "" && !isConfigured("apis.operate_api.key") {
		a.Operate.Key = OperateApiKeyConst
	}
	operateBaseConfigured := a.Operate.BaseURL != "" || isConfigured("apis.operate_api.base_url")
	if a.Operate.Version == "" && !isConfigured("apis.operate_api.version") {
		a.Operate.Version = OperateApiVersionConst
	}
	if a.Tasklist.Key == "" && !isConfigured("apis.tasklist_api.key") {
		a.Tasklist.Key = TasklistApiKeyConst
	}
	tasklistBaseConfigured := a.Tasklist.BaseURL != "" || isConfigured("apis.tasklist_api.base_url")
	if a.Tasklist.Version == "" && !isConfigured("apis.tasklist_api.version") {
		a.Tasklist.Version = TasklistApiVersionConst
	}
	camundaRoot := withoutAPIVersion(a.Camunda.BaseURL)
	if a.Operate.BaseURL == "" && !isConfigured("apis.operate_api.base_url") {
		a.Operate.BaseURL = camundaRoot
	}
	if a.Tasklist.BaseURL == "" && !isConfigured("apis.tasklist_api.base_url") {
		a.Tasklist.BaseURL = camundaRoot
	}
	if !a.VersioningDisable {
		a.Camunda.BaseURL = a.normalizeBaseURL("apis.camunda_api.base_url", a.Camunda.BaseURL, a.Camunda.Version, camundaBaseConfigured)
		a.Operate.BaseURL = a.normalizeBaseURL("apis.operate_api.base_url", a.Operate.BaseURL, a.Operate.Version, operateBaseConfigured)
		a.Tasklist.BaseURL = a.normalizeBaseURL("apis.tasklist_api.base_url", a.Tasklist.BaseURL, a.Tasklist.Version, tasklistBaseConfigured)
	}
}

func (a *APIs) Validate(scopes Scopes) error {
	var errs []error
	if a.Camunda.BaseURL == "" {
		errs = append(errs, fmt.Errorf("apis.camunda_api.base_url: base_url is required"))
	}
	apis := []API{a.Camunda, a.Operate, a.Tasklist}
	for _, api := range apis {
		if api.RequireScope && strings.TrimSpace(scopes[api.Key]) == "" {
			errs = append(errs, fmt.Errorf("api %s requires an auth scope but none was provided as auth.oauth2.scopes.%s", api.Key, api.Key))
		}
	}
	return errors.Join(errs...)
}

var verRx = regexp.MustCompile(`^v\d+(?:\.\d+)*$`)

func withAPIVersion(base, want string) string {
	base = strings.TrimRight(base, "/")
	want = strings.ToLower(strings.TrimSpace(want))

	// last path segment
	i := strings.LastIndex(base, "/")
	last := base
	prefix := ""
	if i >= 0 {
		prefix = base[:i]
		last = base[i+1:]
	}

	if want != "" {
		want = "/" + want
	}
	if verRx.MatchString(last) {
		return prefix + want
	}
	return base + want
}

func withoutAPIVersion(base string) string {
	base = strings.TrimRight(base, "/")
	i := strings.LastIndex(base, "/")
	last := base
	prefix := ""
	if i >= 0 {
		prefix = base[:i]
		last = base[i+1:]
	}
	if verRx.MatchString(last) {
		return prefix
	}
	return base
}

func (a *APIs) Warnings() []string {
	if len(a.warnings) == 0 {
		return nil
	}
	out := make([]string, len(a.warnings))
	copy(out, a.warnings)
	return out
}

func (a *APIs) normalizeBaseURL(key, base, want string, explicit bool) string {
	want = strings.ToLower(strings.TrimSpace(want))
	if strings.TrimSpace(base) == "" || want == "" {
		return strings.TrimRight(base, "/")
	}

	base = strings.TrimRight(base, "/")
	root, chain := trailingVersionChain(base)
	normalized := withAPIVersion(root, want)

	if explicit && shouldWarnForVersionCorrection(chain, want, base, normalized) {
		a.warnings = append(a.warnings, formatVersionCorrectionWarning(key, base, normalized, chain, want))
	}
	return normalized
}

func trailingVersionChain(base string) (string, []string) {
	base = strings.TrimRight(base, "/")
	if base == "" {
		return "", nil
	}
	parts := strings.Split(base, "/")
	var chain []string
	i := len(parts) - 1
	for i >= 0 && verRx.MatchString(parts[i]) {
		chain = append([]string{strings.ToLower(parts[i])}, chain...)
		i--
	}
	if len(chain) == 0 {
		return base, nil
	}
	root := strings.Join(parts[:i+1], "/")
	return root, chain
}

func shouldWarnForVersionCorrection(chain []string, want, original, normalized string) bool {
	if len(chain) == 0 {
		return false
	}
	if len(chain) == 1 && chain[0] == want && original == normalized {
		return false
	}
	return true
}

func formatVersionCorrectionWarning(key, original, normalized string, chain []string, want string) string {
	reason := "replaced trailing API version suffix"
	switch {
	case len(chain) > 1:
		reason = "collapsed duplicated or mixed trailing API version suffixes"
	case len(chain) == 1 && chain[0] != want:
		reason = "replaced trailing API version suffix"
	}
	return fmt.Sprintf("%s: %s; corrected %q to %q (expected final API version %q)", key, reason, original, normalized, want)
}
