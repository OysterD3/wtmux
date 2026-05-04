// Package launch builds the final argv for the configured launch command
// and execs it. Mirrors src/launch.ts.
package launch

import (
	"errors"
	"os"
	"os/exec"

	"github.com/OysterD3/wtmux/internal/agents"
	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
)

// BuildArgv composes [...launchCommand, ...siblingArgs, ...extraArgs].
//
// siblingArgs depends on strategy:
//   - StrategyFlag: agents.ExpandAddDirArgs(strategy.FlagArgs, siblings)
//   - StrategyPositional: siblings appended directly
//   - StrategyNone: rejected — caller must short-circuit and not call
//     BuildArgv (matches TS).
func BuildArgv(
	launchCommand []string,
	siblings []string,
	strategy agents.ResolvedStrategy,
	extra []string,
) ([]string, error) {
	if strategy.Kind == agents.StrategyNone {
		return nil, wtmuxerrors.New(
			wtmuxerrors.KindInternal,
			`BuildArgv called with "none" strategy (agent: %s); callers must short-circuit the launch`,
			strategy.AgentID,
		)
	}

	var siblingArgs []string
	switch strategy.Kind {
	case agents.StrategyFlag:
		siblingArgs = agents.ExpandAddDirArgs(strategy.FlagArgs, siblings)
	case agents.StrategyPositional:
		siblingArgs = append(siblingArgs, siblings...)
	}

	out := make([]string, 0, len(launchCommand)+len(siblingArgs)+len(extra))
	out = append(out, launchCommand...)
	out = append(out, siblingArgs...)
	out = append(out, extra...)
	return out, nil
}

// ExecFunc is the function used to run the final argv. Replaceable in
// tests; defaults to runWithExitCode.
var ExecFunc = runWithExitCode

// Exec runs argv in cwd with stdio inherited, then calls os.Exit with the
// child's exit code (matching TS spawnSync + process.exit(status)).
//
// Empty argv writes a [wtmux] message to stderr and exits 2 — same as TS.
func Exec(argv []string, cwd string) {
	if len(argv) == 0 {
		os.Stderr.WriteString("[wtmux] launchCommand is empty — nothing to exec\n")
		os.Exit(2)
	}
	code := ExecFunc(argv, cwd)
	os.Exit(code)
}

func runWithExitCode(argv []string, cwd string) int {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	// spawn failure (binary not found, permission, etc.)
	return 1
}
