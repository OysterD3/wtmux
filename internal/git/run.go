package git

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// run executes `git <args...>` in cwd. err is non-nil only on spawn failures
// (e.g. git binary missing); a nonzero git exit appears as exitCode != 0 with
// nil err. stdout and stderr are captured and trimmed of trailing newlines.
// An empty cwd means "inherit caller's working directory" (used by callers
// that don't need a repo, e.g. CheckRefFormat).
func run(cwd string, args ...string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.Command("git", args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()

	stdout = strings.TrimRight(outBuf.String(), "\n")
	stderr = strings.TrimRight(errBuf.String(), "\n")

	if runErr == nil {
		return stdout, stderr, 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		return stdout, stderr, exitErr.ExitCode(), nil
	}
	return stdout, stderr, -1, runErr
}
