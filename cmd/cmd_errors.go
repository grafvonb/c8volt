// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
)

var (
	ErrInvalidFlagValue         = errors.New("invalid flag value")
	ErrForbiddenFlagCombination = errors.New("forbidden flag combination")
	ErrMissingDependentFlags    = errors.New("missing dependent flags")
	ErrMutuallyExclusiveFlags   = errors.New("mutually exclusive flags")
	ErrAmbiguousConfigValue     = errors.New("ambiguous config value")
)

type errorClassRule struct {
	match error
	class error
}

var commandErrorClassRules = []errorClassRule{
	{match: ErrInvalidFlagValue, class: ferrors.ErrInvalidInput},
	{match: ErrForbiddenFlagCombination, class: ferrors.ErrInvalidInput},
	{match: ErrMissingDependentFlags, class: ferrors.ErrInvalidInput},
	{match: ErrMutuallyExclusiveFlags, class: ferrors.ErrInvalidInput},
	{match: ErrAmbiguousConfigValue, class: ferrors.ErrInvalidInput},
}

func invalidFlagValuef(format string, args ...any) error {
	return commandInputError(ErrInvalidFlagValue, format, args...)
}

func invalidInputError(err error) error {
	return ferrors.WrapClass(ferrors.ErrInvalidInput, err)
}

func localPreconditionError(err error) error {
	return ferrors.WrapClass(ferrors.ErrLocalPrecondition, err)
}

func missingDependentFlagsf(format string, args ...any) error {
	return commandInputError(ErrMissingDependentFlags, format, args...)
}

func forbiddenFlagCombinationf(format string, args ...any) error {
	return commandInputError(ErrForbiddenFlagCombination, format, args...)
}

func mutuallyExclusiveFlagsf(format string, args ...any) error {
	return commandInputError(ErrMutuallyExclusiveFlags, format, args...)
}

func ambiguousConfigValuef(format string, args ...any) error {
	return commandInputError(ErrAmbiguousConfigValue, format, args...)
}

func normalizeCommandError(err error) error {
	if err == nil {
		return nil
	}

	// Command-local sentinels may pick the shared class, but all remaining
	// normalization still routes through ferrors to keep one rendering boundary.
	for _, rule := range commandErrorClassRules {
		if errors.Is(err, rule.match) {
			return ferrors.WrapClass(rule.class, err)
		}
	}

	return ferrors.Normalize(err)
}

func commandInputError(kind error, format string, args ...any) error {
	return invalidInputError(fmt.Errorf("%w: %s", kind, fmt.Sprintf(format, args...)))
}
