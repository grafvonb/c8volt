package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/toolx"
)

const (
	CamundaApiKeyConst  = "camunda_api"
	OperateApiKeyConst  = "operate_api"
	TasklistApiKeyConst = "tasklist_api"
)

type APIs struct {
	Version  toolx.CamundaVersion `mapstructure:"version" json:"version" yaml:"version"`
	Camunda  API                  `mapstructure:"camunda_api" json:"camunda_api" yaml:"camunda_api"`
	Operate  API                  `mapstructure:"operate_api" json:"operate_api" yaml:"operate_api"`
	Tasklist API                  `mapstructure:"tasklist_api" json:"tasklist_api" yaml:"tasklist_api"`
}

type API struct {
	Key          string `mapstructure:"key" json:"key" yaml:"key"`
	BaseURL      string `mapstructure:"base_url" json:"base_url" yaml:"base_url"`
	RequireScope bool   `mapstructure:"require_scope" json:"require_scope" yaml:"require_scope"`
}

func (a *APIs) Normalize() error {
	var errs []error
	switch a.Version {
	case "":
		a.Version = toolx.CurrentCamundaVersion
	default:
		v, err := toolx.NormalizeCamundaVersion(string(a.Version))
		if err != nil {
			errs = append(errs, fmt.Errorf("version: %w", err))
		} else {
			a.Version = v
		}
	}
	if a.Camunda.Key == "" {
		a.Camunda.Key = CamundaApiKeyConst
	}
	if a.Operate.Key == "" {
		a.Operate.Key = OperateApiKeyConst
	}
	if a.Tasklist.Key == "" {
		a.Tasklist.Key = TasklistApiKeyConst
	}
	if a.Operate.BaseURL == "" {
		a.Operate.BaseURL = a.Camunda.BaseURL
	}
	if a.Tasklist.BaseURL == "" {
		a.Tasklist.BaseURL = a.Camunda.BaseURL
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
