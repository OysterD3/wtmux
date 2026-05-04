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
