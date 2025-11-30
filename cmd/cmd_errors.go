package cmd

import "errors"

var (
	ErrForbiddenFlagCombination = errors.New("forbidden flag combination")
	ErrMissingDependentFlags    = errors.New("missing dependent flags")
	ErrMutuallyExclusiveFlags   = errors.New("mutually exclusive flags")
)
