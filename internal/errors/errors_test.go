package errors_test

import (
	"errors"
	"fmt"
	"testing"

	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
)

func TestExitCodeFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"user", wtmuxerrors.New(wtmuxerrors.KindUser, "bad"), 1},
		{"precondition", wtmuxerrors.New(wtmuxerrors.KindPrecondition, "bad"), 2},
		{"internal", wtmuxerrors.New(wtmuxerrors.KindInternal, "bad"), 3},
		{"plain error", errors.New("nope"), 3},
		{"wrapped user", fmt.Errorf("ctx: %w", wtmuxerrors.New(wtmuxerrors.KindUser, "bad")), 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := wtmuxerrors.ExitCodeFor(c.err); got != c.want {
				t.Fatalf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestNewFormat(t *testing.T) {
	e := wtmuxerrors.New(wtmuxerrors.KindUser, "name=%s code=%d", "foo", 7)
	if e.Error() != "name=foo code=7" {
		t.Fatalf("unexpected message: %q", e.Error())
	}
	if e.Kind != wtmuxerrors.KindUser {
		t.Fatalf("kind not preserved")
	}
}

func TestWrapfPreservesCause(t *testing.T) {
	sentinel := errors.New("group: sentinel")
	wrapped := wtmuxerrors.Wrapf(wtmuxerrors.KindUser, sentinel, "human-readable: %s", sentinel.Error())

	if !errors.Is(wrapped, sentinel) {
		t.Fatalf("errors.Is should match the wrapped sentinel through *Error.Unwrap")
	}
	if wrapped.Error() != "human-readable: group: sentinel" {
		t.Fatalf("unexpected message: %q", wrapped.Error())
	}
	if wrapped.Kind != wtmuxerrors.KindUser {
		t.Fatalf("kind not preserved")
	}
	// Wrapping nil is allowed and should not match anything via errors.Is.
	if errors.Is(wtmuxerrors.New(wtmuxerrors.KindUser, "no cause"), sentinel) {
		t.Fatalf("New() should not match unrelated sentinels")
	}
}
