// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIsNormalize_DerivesOperateAndTasklistDefaultsFromCamundaRoot(t *testing.T) {
	apis := APIs{
		Camunda: API{
			BaseURL: "https://camunda.example.test/v2",
		},
	}

	err := apis.Normalize()
	require.NoError(t, err)

	require.Equal(t, "https://camunda.example.test/v2", apis.Camunda.BaseURL)
	require.Equal(t, "https://camunda.example.test/v1", apis.Operate.BaseURL)
	require.Equal(t, "https://camunda.example.test/v1", apis.Tasklist.BaseURL)
	require.Equal(t, "v2", apis.Camunda.Version)
	require.Equal(t, "v1", apis.Operate.Version)
	require.Equal(t, "v1", apis.Tasklist.Version)
}

func TestAPIsNormalize_AppendsDefaultVersionsFromHostRoot(t *testing.T) {
	apis := APIs{
		Camunda: API{
			BaseURL: "https://camunda.example.test",
		},
	}

	err := apis.Normalize()
	require.NoError(t, err)

	require.Equal(t, "https://camunda.example.test/v2", apis.Camunda.BaseURL)
	require.Equal(t, "https://camunda.example.test/v1", apis.Operate.BaseURL)
	require.Equal(t, "https://camunda.example.test/v1", apis.Tasklist.BaseURL)
}

func TestAPIsNormalize_PreservesExplicitOperateAndTasklistOverrides(t *testing.T) {
	apis := APIs{
		Camunda: API{
			BaseURL: "https://camunda.example.test/v2",
		},
		Operate: API{
			BaseURL: "https://operate.example.test",
		},
		Tasklist: API{
			BaseURL: "https://tasklist.example.test/v1",
		},
	}

	err := apis.Normalize()
	require.NoError(t, err)

	require.Equal(t, "https://camunda.example.test/v2", apis.Camunda.BaseURL)
	require.Equal(t, "https://operate.example.test/v1", apis.Operate.BaseURL)
	require.Equal(t, "https://tasklist.example.test/v1", apis.Tasklist.BaseURL)
}

func TestAPIsNormalizeWithConfiguredKeys_LeavesExplicitEmptyVersionUnsetForValidation(t *testing.T) {
	apis := APIs{
		Camunda: API{
			BaseURL: "https://camunda.example.test",
		},
	}

	err := apis.normalizeWithConfiguredKeys(func(key string) bool {
		return key == "apis.operate_api.version"
	})
	require.NoError(t, err)

	require.Equal(t, "", apis.Operate.Version)
	require.Equal(t, "https://camunda.example.test", apis.Operate.BaseURL)
}

func TestAPIsNormalize_VersioningDisableSkipsSuffixingButStillDerivesSiblingRoots(t *testing.T) {
	apis := APIs{
		VersioningDisable: true,
		Camunda: API{
			BaseURL: "https://camunda.example.test/v2",
		},
	}

	err := apis.Normalize()
	require.NoError(t, err)

	require.Equal(t, "https://camunda.example.test/v2", apis.Camunda.BaseURL)
	require.Equal(t, "https://camunda.example.test", apis.Operate.BaseURL)
	require.Equal(t, "https://camunda.example.test", apis.Tasklist.BaseURL)
}

func TestAPIsNormalize_WarnsAndCorrectsExplicitMixedOperateVersion(t *testing.T) {
	apis := APIs{
		Operate: API{
			BaseURL: "https://operate.example.test/v2",
		},
	}

	err := apis.Normalize()
	require.NoError(t, err)

	require.Equal(t, "https://operate.example.test/v1", apis.Operate.BaseURL)
	require.Len(t, apis.Warnings(), 1)
	require.Contains(t, apis.Warnings()[0], `apis.operate_api.base_url`)
	require.Contains(t, apis.Warnings()[0], `"https://operate.example.test/v2"`)
	require.Contains(t, apis.Warnings()[0], `"https://operate.example.test/v1"`)
}

func TestAPIsNormalize_WarnsAndCorrectsDuplicatedTasklistVersion(t *testing.T) {
	apis := APIs{
		Tasklist: API{
			BaseURL: "https://tasklist.example.test/v1/v1",
		},
	}

	err := apis.Normalize()
	require.NoError(t, err)

	require.Equal(t, "https://tasklist.example.test/v1", apis.Tasklist.BaseURL)
	require.Len(t, apis.Warnings(), 1)
	require.Contains(t, apis.Warnings()[0], "duplicated or mixed")
}

func TestAPIsNormalize_WarnsAndCorrectsMixedTrailingVersionChain(t *testing.T) {
	apis := APIs{
		Operate: API{
			BaseURL: "https://operate.example.test/v2/v1",
		},
	}

	err := apis.Normalize()
	require.NoError(t, err)

	require.Equal(t, "https://operate.example.test/v1", apis.Operate.BaseURL)
	require.Len(t, apis.Warnings(), 1)
	require.Contains(t, apis.Warnings()[0], "duplicated or mixed")
}

func TestAPIsNormalize_WarnsAndCorrectsCamundaWrongDefaultVersion(t *testing.T) {
	apis := APIs{
		Camunda: API{
			BaseURL: "https://camunda.example.test/v1",
		},
	}

	err := apis.Normalize()
	require.NoError(t, err)

	require.Equal(t, "https://camunda.example.test/v2", apis.Camunda.BaseURL)
	require.Len(t, apis.Warnings(), 1)
	require.Contains(t, apis.Warnings()[0], `apis.camunda_api.base_url`)
	require.Contains(t, apis.Warnings()[0], `"https://camunda.example.test/v1"`)
	require.Contains(t, apis.Warnings()[0], `"https://camunda.example.test/v2"`)
}
