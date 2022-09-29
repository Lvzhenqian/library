package errors

import (
	"fmt"
	"runtime"
	"strings"
)

type Error struct {
	message  string
	previous error
	cause    error
	filename string
	funcName string
	line     int
}

func (e *Error) Error() string {
	err := e.previous
	switch {
	case err == nil:
		return e.message
	case e.message == "":
		return err.Error()
	}
	return fmt.Sprintf("%s: %v", e.message, err)
}

func (e *Error) SetLocation(callDepth int) {
	e.filename, e.funcName, e.line = getLocation(callDepth + 1)
}

func (e *Error) Previous() error {
	return e.previous
}

func (e *Error) Unwrap() error {
	return e.previous
}

func (e *Error) Cause() error {
	return e.cause
}

func (e *Error) GoString() string {
	return "github.com/Lvzhenqian/library/errors.Error"
}

func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			fmt.Fprintf(s, "%s", ErrorStack(e))
			return
		case s.Flag('#'):
			fmt.Fprintf(s, "%#v", (*unformatter)(e))
			return
		}
		fallthrough
	case 's':
		fmt.Fprintf(s, "%s", e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	default:
		fmt.Fprintf(s, "%%!%c(%T=%s)", verb, e, e.Error())
	}
}

type unformatter Error

func (unformatter) Format() {}

func New(message string) error {
	err := &Error{message: message}
	err.SetLocation(1)
	return err
}

func Wrapf(other error, format string, args ...interface{}) error {
	if other == nil {
		return nil
	}
	err := &Error{
		previous: other,
		cause:    Cause(other),
		message:  fmt.Sprintf(format, args...),
	}
	err.SetLocation(1)
	return err
}

func Wrap(other error) error {
	if other == nil {
		return nil
	}
	err := &Error{
		previous: other,
		cause:    Cause(other),
	}
	err.SetLocation(1)
	return err
}

func Cause(err error) error {
	var diag error
	if err, ok := err.(*Error); ok {
		diag = err.Cause()
	}
	if diag != nil {
		return diag
	}
	return err
}

func getLocation(callDepth int) (string, string, int) {
	rpc := make([]uintptr, 1)
	n := runtime.Callers(callDepth+2, rpc[:])
	if n < 1 {
		return "", "", 0
	}
	frame, _ := runtime.CallersFrames(rpc).Next()

	return frame.File, frame.Function, frame.Line
}

func ErrorStack(err error) string {
	return strings.Join(errorStack(err), "\n")
}

func errorStack(err error) []string {
	if err == nil {
		return nil
	}

	var lines []string
	for {
		if cerr, ok := err.(*Error); ok {
			line := fmt.Sprintf("%s(): %s\n\t%s:%d", cerr.funcName, cerr.Error(), cerr.filename, cerr.line)
			lines = append(lines, line)

			err = cerr.Previous()
		} else {
			lines = append(lines, err.Error())
			err = nil
		}

		if err == nil {
			break
		}
	}
	return lines
}
