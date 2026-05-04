// Package errors defines wtmux's typed CLI error and the exit-code
// translation used by the top-level Execute function. Mirrors the TS
// WtmuxError ↔ exitCodeFor pairing byte-for-byte: user=1, precondition=2,
// internal=3, and any other error type defaults to 3.
package errors

import (
	"errors"
	"fmt"
)

// Kind discriminates between the three exit-code buckets the CLI uses.
type Kind int

const (
	// KindUser is a user-facing error caused by bad input or invocation.
	// Exit code 1.
	KindUser Kind = iota
	// KindPrecondition is a refusal to act because the environment isn't
	// in the expected shape (e.g. detached HEAD, dirty worktree). Exit
	// code 2.
	KindPrecondition
	// KindInternal is an unexpected internal failure. Exit code 3, also
	// the default when an error of any other type bubbles up.
	KindInternal
)

// Error is wtmux's tagged CLI error. Cause is optional and only used so
// errors.Is / errors.As checks against sentinels (e.g. ErrGroupFlagNotFound)
// continue to match after the CLI rephrases the message.
type Error struct {
	Msg   string
	Kind  Kind
	Cause error
}

// Error implements the error interface.
func (e *Error) Error() string { return e.Msg }

// Unwrap returns the wrapped cause (or nil). Lets errors.Is/As traverse
// past a *Error.
func (e *Error) Unwrap() error { return e.Cause }

// New returns a new *Error with no wrapped cause. Format is fmt.Sprintf-style.
func New(kind Kind, format string, a ...any) *Error {
	return &Error{Msg: fmt.Sprintf(format, a...), Kind: kind}
}

// Wrapf returns a new *Error that wraps cause and formats msg via fmt.Sprintf.
// Use this when the source error carries sentinels (errors.Is targets) the
// outer caller may want to inspect.
func Wrapf(kind Kind, cause error, format string, a ...any) *Error {
	return &Error{Msg: fmt.Sprintf(format, a...), Kind: kind, Cause: cause}
}

// ExitCodeFor returns the process exit code for err. Unwraps via
// errors.As; non-*Error values default to 3.
func ExitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	var e *Error
	if errors.As(err, &e) {
		switch e.Kind {
		case KindUser:
			return 1
		case KindPrecondition:
			return 2
		case KindInternal:
			return 3
		}
	}
	return 3
}
