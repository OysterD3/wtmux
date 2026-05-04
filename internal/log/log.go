// Package log is wtmux's tiny stderr logger. Output mirrors the TS
// implementation byte-for-byte: every line goes to stderr, prefixed with
// "[wtmux] " (or "[wtmux:debug] " for Debug). Debug is gated by SetVerbose.
//
// Out is exposed (var, not const) so tests can capture stderr by swapping
// it for a *bytes.Buffer.
package log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	mu      sync.Mutex
	verbose bool
	// Out is the destination for every log line. Defaults to os.Stderr.
	Out io.Writer = os.Stderr
)

// SetVerbose toggles whether Debug emits lines.
func SetVerbose(v bool) {
	mu.Lock()
	defer mu.Unlock()
	verbose = v
}

// Verbose reports whether debug logging is enabled.
func Verbose() bool {
	mu.Lock()
	defer mu.Unlock()
	return verbose
}

// Info writes "[wtmux] msg\n" to Out.
func Info(msg string) { write("[wtmux] ", msg) }

// Warn writes "[wtmux] msg\n" to Out. (Same prefix as Info, matching TS.)
func Warn(msg string) { write("[wtmux] ", msg) }

// Error writes "[wtmux] msg\n" to Out. (Same prefix as Info, matching TS.)
func Error(msg string) { write("[wtmux] ", msg) }

// Debug writes "[wtmux:debug] msg\n" to Out, but only if SetVerbose(true)
// has been called.
func Debug(msg string) {
	mu.Lock()
	if !verbose {
		mu.Unlock()
		return
	}
	mu.Unlock()
	write("[wtmux:debug] ", msg)
}

// Infof / Warnf / Errorf / Debugf are fmt.Sprintf-style conveniences.
func Infof(format string, a ...any)  { Info(fmt.Sprintf(format, a...)) }
func Warnf(format string, a ...any)  { Warn(fmt.Sprintf(format, a...)) }
func Errorf(format string, a ...any) { Error(fmt.Sprintf(format, a...)) }
func Debugf(format string, a ...any) { Debug(fmt.Sprintf(format, a...)) }

func write(prefix, msg string) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprintf(Out, "%s%s\n", prefix, msg)
}
