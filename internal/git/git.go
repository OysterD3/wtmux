// Package git is a thin wrapper around the system git CLI. Every exported
// function shells out to git via the internal run helper. err is reserved for
// spawn failures (e.g. git binary missing); a nonzero git exit appears as
// (zero-value, nil) for read functions and as a wrapped error for write
// functions. Tests require git on PATH (TestMain hard-fails otherwise).
package git

import "strings"

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

// BranchExists reports whether a local branch named name exists in repo
// (i.e. refs/heads/<name>).
func BranchExists(repo, name string) (bool, error) {
	_, _, code, err := run(repo, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if err != nil {
		return false, err
	}
	return code == 0, nil
}

// HasRef reports whether ref resolves in repo via `rev-parse --verify`.
// Accepts any ref form (branch, tag, SHA, HEAD, etc.).
func HasRef(repo, ref string) (bool, error) {
	_, _, code, err := run(repo, "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		return false, err
	}
	return code == 0, nil
}

// StatusPorcelain returns the trimmed stdout of `git status --porcelain`.
// Empty string means a clean working tree.
func StatusPorcelain(repo string) (string, error) {
	stdout, _, _, err := run(repo, "status", "--porcelain")
	if err != nil {
		return "", err
	}
	return stdout, nil
}

// StashList returns the trimmed non-empty lines of `git stash list`.
// Returns empty (nil) on a repo with no stashes.
func StashList(repo string) ([]string, error) {
	stdout, _, _, err := run(repo, "stash", "list")
	if err != nil {
		return nil, err
	}
	if stdout == "" {
		return nil, nil
	}
	var out []string
	for _, line := range strings.Split(stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out, nil
}

// UnpushedCommits returns the trimmed non-empty lines of
// `git log @{u}..HEAD --oneline`. Returns empty (nil) when no upstream is
// configured for the current branch (e.g. a freshly created branch with no
// `git push -u`).
func UnpushedCommits(repo string) ([]string, error) {
	_, _, code, err := run(repo, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, nil
	}
	stdout, _, _, err := run(repo, "log", "@{u}..HEAD", "--oneline")
	if err != nil {
		return nil, err
	}
	if stdout == "" {
		return nil, nil
	}
	var out []string
	for _, line := range strings.Split(stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out, nil
}

// ListWorktrees returns every worktree (main + linked) attached to repo, by
// shelling out to `git worktree list --porcelain` and parsing the result.
// Returns empty (nil) on git failure (matches TS).
func ListWorktrees(repo string) ([]WorktreeEntry, error) {
	stdout, _, code, err := run(repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, nil
	}
	return parseWorktreePorcelain(stdout), nil
}
