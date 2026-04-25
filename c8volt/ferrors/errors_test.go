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

// TestNormalizeDomain locks the mapping from service/domain sentinel errors to
// facade classifications and exit codes. These cases drive command behavior, so
// refactors must preserve both errors.Is compatibility and user-facing exits.
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

// TestNormalizeLocal covers local precondition and configuration failures that
// never came from Camunda but still need stable facade classes.
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

// TestNormalizeFallsBackToInternal ensures unknown errors still become a
// normalized facade error instead of leaking unclassified implementation text.
func TestNormalizeFallsBackToInternal(t *testing.T) {
	t.Parallel()

	err := errors.New("boom")
	got := Normalize(err)

	require.ErrorIs(t, got, ErrInternal)
	require.Equal(t, ClassInternal, Classify(got))
	require.Equal(t, exitcode.Error, ExitCode(got))
}

// TestResolveExitCodePreservesNoErrCodesOverride verifies the --no-err-codes
// contract: diagnostics still describe the failure, but the process exit status
// is forced to success for automation pipelines that opt into that behavior.
func TestResolveExitCodePreservesNoErrCodesOverride(t *testing.T) {
	t.Parallel()

	err := NormalizeDomain(domain.ErrNotFound)

	require.Equal(t, ClassNotFound, Classify(err))
	require.Equal(t, exitcode.NotFound, ExitCode(err))
	require.Equal(t, exitcode.OK, ResolveExitCode(true, err))
}

// TestOutcomeAlignsWithSharedFailureVocabulary keeps machine-readable outcome
// strings aligned with the shared error classes used by command envelopes.
func TestOutcomeAlignsWithSharedFailureVocabulary(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", Outcome(nil))
	require.Equal(t, "invalid", Outcome(NormalizeDomain(domain.ErrBadRequest)))
	require.Equal(t, "failed", Outcome(NormalizeDomain(domain.ErrUnavailable)))
	require.Equal(t, "failed", Outcome(NormalizeDomain(domain.ErrNotFound)))
}

// TestWrapClassPreservesExistingClassification prevents duplicate wrapping from
// changing the classification or breaking errors.Is checks.
func TestWrapClassPreservesExistingClassification(t *testing.T) {
	t.Parallel()

	err := WrapClass(ErrInvalidInput, WrapClass(ErrInvalidInput, errors.New("boom")))

	require.ErrorIs(t, err, ErrInvalidInput)
	require.Equal(t, ClassInvalidInput, Classify(err))
}

// TestNormalizeKeepsAlreadyNormalizedDetailStable verifies that re-normalizing
// a facade error keeps the original detail text and does not stack prefixes.
func TestNormalizeKeepsAlreadyNormalizedDetailStable(t *testing.T) {
	t.Parallel()

	err := WrapClass(ErrNotFound, errors.New("get 123: process instance 123 not found"))
	got := Normalize(err)

	require.Equal(t, "resource not found: get 123: process instance 123 not found", got.Error())
	require.ErrorIs(t, got, ErrNotFound)
	require.Equal(t, ClassNotFound, Classify(got))
	require.Equal(t, exitcode.NotFound, ExitCode(got))
}

// TestWrapClassPreservesWrappedDetailText documents the public error-message
// shape: class prefix first, then the lower-level operation detail.
func TestWrapClassPreservesWrappedDetailText(t *testing.T) {
	t.Parallel()

	err := WrapClass(ErrUnsupported, errors.New("render topology: feature disabled by server"))

	require.Equal(t, "unsupported capability: render topology: feature disabled by server", err.Error())
	require.ErrorIs(t, err, ErrUnsupported)
	require.Equal(t, ClassUnsupported, Classify(err))
}

// TestWrapClassPreservesUnavailablePrefixAndDetailText covers the unavailable
// class specifically because it maps to a dedicated exit code and is common for
// upstream outages.
func TestWrapClassPreservesUnavailablePrefixAndDetailText(t *testing.T) {
	t.Parallel()

	err := WrapClass(ErrUnavailable, errors.New("get cluster topology: upstream returned 503"))

	require.Equal(t, "service unavailable: get cluster topology: upstream returned 503", err.Error())
	require.ErrorIs(t, err, ErrUnavailable)
	require.Equal(t, ClassUnavailable, Classify(err))
	require.Equal(t, exitcode.Unavailable, ExitCode(err))
}
