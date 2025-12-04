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
	OperateApiVersionConst  = "" // v1 by default from client side
	TasklistApiKeyConst     = "tasklist_api"
	TasklistApiVersionConst = "" // v1 by default from client side
)

type APIs struct {
	Camunda           API  `mapstructure:"camunda_api" json:"camunda_api" yaml:"camunda_api"`
	Operate           API  `mapstructure:"operate_api" json:"operate_api" yaml:"operate_api"`
	Tasklist          API  `mapstructure:"tasklist_api" json:"tasklist_api" yaml:"tasklist_api"`
	VersioningDisable bool `mapstructure:"versioning_disable" json:"versioning_disable" yaml:"versioning_disable"`
}

type API struct {
	Key          string `mapstructure:"key" json:"key" yaml:"key"`
	BaseURL      string `mapstructure:"base_url" json:"base_url" yaml:"base_url"`
	RequireScope bool   `mapstructure:"require_scope" json:"require_scope" yaml:"require_scope"`
	Version      string `mapstructure:"version" json:"version" yaml:"version"`
}

func (a *APIs) Normalize() error {
	var errs []error
	if a.Camunda.Key == "" {
		a.Camunda.Key = CamundaApiKeyConst
	}
	if a.Camunda.Version == "" {
		a.Camunda.Version = CamundaApiVersionConst
	}
	if a.Operate.Key == "" {
		a.Operate.Key = OperateApiKeyConst
	}
	if a.Operate.Version == "" {
		a.Operate.Version = OperateApiVersionConst
	}
	if a.Tasklist.Key == "" {
		a.Tasklist.Key = TasklistApiKeyConst
	}
	if a.Tasklist.Version == "" {
		a.Tasklist.Version = TasklistApiVersionConst
	}
	if a.Operate.BaseURL == "" {
		a.Operate.BaseURL = a.Camunda.BaseURL
	}
	if a.Tasklist.BaseURL == "" {
		a.Tasklist.BaseURL = a.Camunda.BaseURL
	}
	if !a.VersioningDisable {
		a.Camunda.BaseURL = withAPIVersion(a.Camunda.BaseURL, a.Camunda.Version)
		a.Operate.BaseURL = withAPIVersion(a.Operate.BaseURL, a.Operate.Version)
		a.Tasklist.BaseURL = withAPIVersion(a.Tasklist.BaseURL, a.Tasklist.Version)
	}
	return errors.Join(errs...)
}

func (a *APIs) Validate(scopes Scopes) error {
	var errs []error
	if a.Camunda.BaseURL == "" {
		errs = append(errs, fmt.Errorf("apis.camunda_api.base_url: %w", ErrNoBaseURL))
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
