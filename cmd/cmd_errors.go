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
)

func invalidFlagValuef(format string, args ...any) error {
	return commandInputError(ErrInvalidFlagValue, format, args...)
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

func normalizeCommandError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrInvalidFlagValue),
		errors.Is(err, ErrForbiddenFlagCombination),
		errors.Is(err, ErrMissingDependentFlags),
		errors.Is(err, ErrMutuallyExclusiveFlags):
		return wrapBootstrapClass(ferrors.ErrInvalidInput, err)
	default:
		return ferrors.Normalize(err)
	}
}

func commandInputError(kind error, format string, args ...any) error {
	return wrapBootstrapClass(ferrors.ErrInvalidInput, fmt.Errorf("%w: %s", kind, fmt.Sprintf(format, args...)))
}
