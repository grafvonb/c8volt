package ferrors

import (
	"context"
	"errors"
	"testing"

	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantIs    error
		wantClass Class
		wantCode  int
	}{
		{
			name:      "bad request becomes invalid input",
			err:       domain.ErrBadRequest,
			wantIs:    ErrInvalidInput,
			wantClass: ClassInvalidInput,
			wantCode:  exitcode.InvalidArgs,
		},
		{
			name:      "malformed response stays internal-class exit",
			err:       domain.ErrMalformedResponse,
			wantIs:    ErrMalformedResponse,
			wantClass: ClassMalformedResponse,
			wantCode:  exitcode.Error,
		},
		{
			name:      "gateway timeout becomes timeout",
			err:       domain.ErrGatewayTimeout,
			wantIs:    ErrTimeout,
			wantClass: ClassTimeout,
			wantCode:  exitcode.Timeout,
		},
		{
			name:      "upstream conflict preserves dedicated exit code",
			err:       domain.ErrConflict,
			wantIs:    ErrConflict,
			wantClass: ClassConflict,
			wantCode:  exitcode.Conflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NormalizeDomain(tt.err)

			require.ErrorIs(t, got, tt.wantIs)
			require.Equal(t, tt.wantClass, Classify(got))
			require.Equal(t, tt.wantCode, ExitCode(got))
		})
	}
}

func TestNormalizeLocal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantIs    error
		wantClass Class
		wantCode  int
	}{
		{
			name:      "unknown version becomes unsupported",
			err:       services.ErrUnknownAPIVersion,
			wantIs:    ErrUnsupported,
			wantClass: ClassUnsupported,
			wantCode:  exitcode.Error,
		},
		{
			name:      "unknown configured camunda version becomes unsupported",
			err:       toolx.ErrUnknownCamundaVersion,
			wantIs:    ErrUnsupported,
			wantClass: ClassUnsupported,
			wantCode:  exitcode.Error,
		},
		{
			name:      "missing logger becomes local precondition",
			err:       services.ErrNoLogger,
			wantIs:    ErrLocalPrecondition,
			wantClass: ClassLocalPrecondition,
			wantCode:  exitcode.Error,
		},
		{
			name:      "context cancellation becomes local precondition",
			err:       context.Canceled,
			wantIs:    ErrLocalPrecondition,
			wantClass: ClassLocalPrecondition,
			wantCode:  exitcode.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NormalizeLocal(tt.err)

			require.ErrorIs(t, got, tt.wantIs)
			require.Equal(t, tt.wantClass, Classify(got))
			require.Equal(t, tt.wantCode, ExitCode(got))
		})
	}
}

func TestNormalizeFallsBackToInternal(t *testing.T) {
	t.Parallel()

	err := errors.New("boom")
	got := Normalize(err)

	require.ErrorIs(t, got, ErrInternal)
	require.Equal(t, ClassInternal, Classify(got))
	require.Equal(t, exitcode.Error, ExitCode(got))
}

func TestResolveExitCodePreservesNoErrCodesOverride(t *testing.T) {
	t.Parallel()

	err := NormalizeDomain(domain.ErrNotFound)

	require.Equal(t, ClassNotFound, Classify(err))
	require.Equal(t, exitcode.NotFound, ExitCode(err))
	require.Equal(t, exitcode.OK, ResolveExitCode(true, err))
}

func TestOutcomeAlignsWithSharedFailureVocabulary(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", Outcome(nil))
	require.Equal(t, "invalid", Outcome(NormalizeDomain(domain.ErrBadRequest)))
	require.Equal(t, "failed", Outcome(NormalizeDomain(domain.ErrUnavailable)))
	require.Equal(t, "failed", Outcome(NormalizeDomain(domain.ErrNotFound)))
}

func TestWrapClassPreservesExistingClassification(t *testing.T) {
	t.Parallel()

	err := WrapClass(ErrInvalidInput, WrapClass(ErrInvalidInput, errors.New("boom")))

	require.ErrorIs(t, err, ErrInvalidInput)
	require.Equal(t, ClassInvalidInput, Classify(err))
}

func TestNormalizeKeepsAlreadyNormalizedDetailStable(t *testing.T) {
	t.Parallel()

	err := WrapClass(ErrNotFound, errors.New("get 123: process instance 123 not found"))
	got := Normalize(err)

	require.Equal(t, "resource not found: get 123: process instance 123 not found", got.Error())
	require.ErrorIs(t, got, ErrNotFound)
	require.Equal(t, ClassNotFound, Classify(got))
	require.Equal(t, exitcode.NotFound, ExitCode(got))
}

func TestWrapClassPreservesWrappedDetailText(t *testing.T) {
	t.Parallel()

	err := WrapClass(ErrUnsupported, errors.New("render topology: feature disabled by server"))

	require.Equal(t, "unsupported capability: render topology: feature disabled by server", err.Error())
	require.ErrorIs(t, err, ErrUnsupported)
	require.Equal(t, ClassUnsupported, Classify(err))
}

func TestWrapClassPreservesUnavailablePrefixAndDetailText(t *testing.T) {
	t.Parallel()

	err := WrapClass(ErrUnavailable, errors.New("get cluster topology: upstream returned 503"))

	require.Equal(t, "service unavailable: get cluster topology: upstream returned 503", err.Error())
	require.ErrorIs(t, err, ErrUnavailable)
	require.Equal(t, ClassUnavailable, Classify(err))
	require.Equal(t, exitcode.Unavailable, ExitCode(err))
}
