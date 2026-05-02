// Package git is a thin wrapper around the system git CLI. Every exported
// function shells out to git via the internal run helper. err is reserved for
// spawn failures (e.g. git binary missing); a nonzero git exit appears as
// (zero-value, nil) for read functions and as a wrapped error for write
// functions. Tests require git on PATH (TestMain hard-fails otherwise).
package git

// CheckRefFormat reports whether name is a valid git branch name, per
// `git check-ref-format --branch`. No working directory is required.
func CheckRefFormat(name string) (bool, error) {
	_, _, code, err := run("", "check-ref-format", "--branch", name)
	if err != nil {
		return false, err
	}
	return code == 0, nil
}

// GetToplevel returns the top-level path of the git working tree containing
// cwd. ok is false (with err == nil) when cwd is not inside any git repo.
// Non-nil err only on spawn failure.
func GetToplevel(cwd string) (path string, ok bool, err error) {
	stdout, _, code, err := run(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", false, err
	}
	if code != 0 {
		return "", false, nil
	}
	return stdout, true, nil
}

// IsWorktreeRoot reports whether repo is the main worktree of its repository
// (git-dir == git-common-dir). Linked worktrees have a different git-dir
// under .git/worktrees/<name>.
func IsWorktreeRoot(repo string) (bool, error) {
	commonOut, _, commonCode, err := run(repo, "rev-parse", "--git-common-dir")
	if err != nil {
		return false, err
	}
	if commonCode != 0 {
		return false, nil
	}
	dirOut, _, dirCode, err := run(repo, "rev-parse", "--git-dir")
	if err != nil {
		return false, err
	}
	if dirCode != 0 {
		return false, nil
	}
	return commonOut == dirOut, nil
}

// GetCurrentBranch returns the current branch name. ok is false (with err
// == nil) on detached HEAD. Non-nil err only on spawn failure.
func GetCurrentBranch(repo string) (branch string, ok bool, err error) {
	stdout, _, code, err := run(repo, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return "", false, err
	}
	if code != 0 {
		return "", false, nil
	}
	if stdout == "" {
		return "", false, nil
	}
	return stdout, true, nil
}
