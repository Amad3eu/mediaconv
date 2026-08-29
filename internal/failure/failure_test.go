package failure

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorMessageAndUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("low-level failure")
	tests := []struct {
		name    string
		err     *Error
		message string
		unwrap  error
	}{
		{
			name:    "public message takes precedence",
			err:     &Error{Kind: Input, Message: "input is invalid", Err: cause},
			message: "input is invalid",
			unwrap:  cause,
		},
		{
			name:    "cause is fallback",
			err:     &Error{Kind: Conversion, Err: cause},
			message: "low-level failure",
			unwrap:  cause,
		},
		{
			name:    "fully empty error",
			err:     &Error{},
			message: "unexpected error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.err.Error(); got != test.message {
				t.Errorf("Error() = %q, want %q", got, test.message)
			}
			if got := test.err.Unwrap(); got != test.unwrap {
				t.Errorf("Unwrap() = %v, want %v", got, test.unwrap)
			}
		})
	}
}

func TestNewAndWrapPreserveFailureMetadata(t *testing.T) {
	t.Parallel()

	cause := errors.New("disk full")
	err := New(OutputConflict, "cannot publish", "free disk space", cause)
	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("New() result type = %T, want *Error", err)
	}
	if typed.Kind != OutputConflict || typed.Message != "cannot publish" || typed.Hint != "free disk space" {
		t.Errorf("New() metadata = %#v", typed)
	}
	if !errors.Is(err, cause) {
		t.Error("New() did not preserve cause")
	}

	wrapped := Wrap(Conversion, "conversion failed", cause)
	if !errors.As(wrapped, &typed) {
		t.Fatalf("Wrap() result type = %T, want *Error", wrapped)
	}
	if typed.Kind != Conversion || typed.Message != "conversion failed" || typed.Hint != "" {
		t.Errorf("Wrap() metadata = %#v", typed)
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "success", err: nil, want: 0},
		{name: "usage", err: New(Usage, "bad arguments", "", nil), want: ExitUsage},
		{name: "dependency", err: New(Dependency, "missing ffmpeg", "", nil), want: ExitDependency},
		{name: "input", err: New(Input, "bad input", "", nil), want: ExitInput},
		{name: "output conflict", err: New(OutputConflict, "exists", "", nil), want: ExitOutputConflict},
		{name: "conversion", err: New(Conversion, "failed", "", nil), want: ExitConversion},
		{name: "interrupted", err: New(Interrupted, "cancelled", "", nil), want: ExitInterrupted},
		{name: "typed unexpected", err: New(Unexpected, "unexpected", "", nil), want: ExitUnexpected},
		{name: "wrapped typed error", err: fmt.Errorf("command failed: %w", New(Input, "bad input", "", nil)), want: ExitInput},
		// Cobra returns unclassified errors for invalid commands and arguments;
		// those are treated as usage errors unless the application assigns a kind.
		{name: "unclassified error", err: errors.New("unknown command"), want: ExitUsage},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ExitCode(test.err); got != test.want {
				t.Errorf("ExitCode(%v) = %d, want %d", test.err, got, test.want)
			}
		})
	}
}

func TestDetailsAndFormat(t *testing.T) {
	t.Parallel()

	cause := errors.New("permission denied by filesystem")
	err := New(OutputConflict, "The output could not be written.", "Check directory permissions.", cause)

	message, hint := Details(err)
	if message != "The output could not be written." || hint != "Check directory permissions." {
		t.Errorf("Details() = (%q, %q)", message, hint)
	}
	if got, want := Format(err, false), "The output could not be written.\nHint: Check directory permissions."; got != want {
		t.Errorf("Format(verbose=false) = %q, want %q", got, want)
	}
	if got, want := Format(err, true), "The output could not be written.\nHint: Check directory permissions.\nDetails: permission denied by filesystem"; got != want {
		t.Errorf("Format(verbose=true) = %q, want %q", got, want)
	}

	plain := errors.New("plain failure")
	message, hint = Details(plain)
	if message != "plain failure" || hint != "" {
		t.Errorf("Details(plain) = (%q, %q)", message, hint)
	}
	if message, hint = Details(nil); message != "" || hint != "" {
		t.Errorf("Details(nil) = (%q, %q)", message, hint)
	}
}

func TestFormatDoesNotDuplicateIdenticalUnderlyingMessage(t *testing.T) {
	t.Parallel()

	err := New(Input, "invalid media", "", errors.New("invalid media"))
	if got := Format(err, true); got != "invalid media" {
		t.Errorf("Format() = %q, want no duplicated details", got)
	}
}

func TestReportedMarksAnErrorWithoutChangingItsIdentity(t *testing.T) {
	t.Parallel()

	cause := errors.New("already rendered")
	reported := Reported(cause)
	if reported == nil {
		t.Fatal("Reported(error) = nil")
	}
	if !IsReported(reported) {
		t.Error("IsReported(Reported(error)) = false")
	}
	if !errors.Is(reported, cause) {
		t.Error("Reported(error) did not preserve error identity")
	}
	if got := reported.Error(); got != cause.Error() {
		t.Errorf("reported Error() = %q, want %q", got, cause)
	}
	if !IsReported(fmt.Errorf("outer: %w", reported)) {
		t.Error("IsReported() did not inspect wrapped errors")
	}
	if IsReported(cause) {
		t.Error("IsReported(plain error) = true")
	}
	if Reported(nil) != nil {
		t.Error("Reported(nil) != nil")
	}
}
