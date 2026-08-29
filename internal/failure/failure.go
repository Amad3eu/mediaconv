package failure

import (
	"errors"
	"fmt"
)

type Kind int

const (
	Unexpected Kind = iota
	Usage
	Dependency
	Input
	OutputConflict
	Conversion
	Interrupted
)

const (
	ExitUnexpected     = 1
	ExitUsage          = 2
	ExitDependency     = 3
	ExitInput          = 4
	ExitOutputConflict = 5
	ExitConversion     = 6
	ExitInterrupted    = 130
)

type Error struct {
	Kind    Kind
	Message string
	Hint    string
	Err     error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unexpected error"
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(kind Kind, message, hint string, err error) error {
	return &Error{Kind: kind, Message: message, Hint: hint, Err: err}
}

func Wrap(kind Kind, message string, err error) error {
	return New(kind, message, "", err)
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	var target *Error
	if !errors.As(err, &target) {
		return ExitUsage
	}

	switch target.Kind {
	case Usage:
		return ExitUsage
	case Dependency:
		return ExitDependency
	case Input:
		return ExitInput
	case OutputConflict:
		return ExitOutputConflict
	case Conversion:
		return ExitConversion
	case Interrupted:
		return ExitInterrupted
	default:
		return ExitUnexpected
	}
}

func Details(err error) (message, hint string) {
	if err == nil {
		return "", ""
	}

	var target *Error
	if errors.As(err, &target) {
		return target.Error(), target.Hint
	}

	return err.Error(), ""
}

type reportedError struct {
	err error
}

func (e *reportedError) Error() string { return e.err.Error() }
func (e *reportedError) Unwrap() error { return e.err }

func Reported(err error) error {
	if err == nil {
		return nil
	}
	return &reportedError{err: err}
}

func IsReported(err error) bool {
	var target *reportedError
	return errors.As(err, &target)
}

func Format(err error, verbose bool) string {
	message, hint := Details(err)
	if hint != "" {
		message += fmt.Sprintf("\nHint: %s", hint)
	}
	if verbose {
		var target *Error
		if errors.As(err, &target) && target.Err != nil && target.Err.Error() != target.Message {
			message += fmt.Sprintf("\nDetails: %v", target.Err)
		}
	}
	return message
}
