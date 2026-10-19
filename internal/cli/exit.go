package cli

import (
	"fmt"
)

const (
	ExitOK        = 0
	ExitError     = 1
	ExitUsage     = 2
	ExitNeedsPath = 3
	ExitNotFound  = 4
)

// exitError carries an exit code and, optionally, structured detail printed to
// stderr in the selected format.
type exitError struct {
	code   int
	err    error
	detail any
	// printed means the command already wrote its full report to stdout.
	printed bool
}

func (e *exitError) Error() string { return e.err.Error() }

func (e *exitError) Unwrap() error { return e.err }

func failf(format string, args ...any) error {
	return &exitError{code: ExitError, err: fmt.Errorf(format, args...)}
}

func usagef(format string, args ...any) error {
	return &exitError{code: ExitUsage, err: fmt.Errorf(format, args...)}
}

// fail marks an error from a command body as a plain failure, keeping any code
// it already carries.
func fail(err error) error {
	if err == nil {
		return nil
	}

	if ee, ok := err.(*exitError); ok { //nolint:errorlint // only the outermost carries the code
		return ee
	}

	return &exitError{code: ExitError, err: err}
}

func notFoundf(format string, args ...any) error {
	return &exitError{code: ExitNotFound, err: fmt.Errorf(format, args...)}
}
