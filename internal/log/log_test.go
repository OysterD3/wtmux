package log_test

import (
	"bytes"
	"strings"
	"testing"

	wtlog "github.com/OysterD3/wtmux/internal/log"
)

func capture(t *testing.T) *bytes.Buffer {
	t.Helper()
	prev := wtlog.Out
	buf := &bytes.Buffer{}
	wtlog.Out = buf
	t.Cleanup(func() { wtlog.Out = prev })
	return buf
}

func TestInfoWarnErrorPrefix(t *testing.T) {
	buf := capture(t)
	wtlog.Info("a")
	wtlog.Warn("b")
	wtlog.Error("c")
	got := buf.String()
	want := "[wtmux] a\n[wtmux] b\n[wtmux] c\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDebugGated(t *testing.T) {
	buf := capture(t)
	wtlog.SetVerbose(false)
	wtlog.Debug("hidden")
	if buf.Len() != 0 {
		t.Fatalf("debug leaked when verbose=false: %q", buf.String())
	}

	wtlog.SetVerbose(true)
	t.Cleanup(func() { wtlog.SetVerbose(false) })
	wtlog.Debug("shown")
	if !strings.Contains(buf.String(), "[wtmux:debug] shown\n") {
		t.Fatalf("debug missing: %q", buf.String())
	}
}

func TestFormatHelpers(t *testing.T) {
	buf := capture(t)
	wtlog.Infof("hello %s %d", "world", 7)
	if buf.String() != "[wtmux] hello world 7\n" {
		t.Fatalf("got %q", buf.String())
	}
}
