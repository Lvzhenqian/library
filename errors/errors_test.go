package errors

import (
	"errors"
	"testing"
)

var ErrNotFound = New("xxx not found")

func a() error {
	return Wrap(ErrNotFound)
}

func b() error {
	return Wrapf(a(), "call %s function error", "a")
}

func TestErrorStack(t *testing.T) {
	err := b()
	t.Log(errors.Is(err, ErrNotFound))
	t.Logf("\n%+v", err)
}
