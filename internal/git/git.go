// Package git is a thin wrapper around the system git CLI. Every exported
// function shells out to git via the internal run helper. err is reserved for
// spawn failures (e.g. git binary missing); a nonzero git exit appears as
// (zero-value, nil) for read functions and as a wrapped error for write
// functions. Tests require git on PATH (TestMain hard-fails otherwise).
package git

import (
	"fmt"
	"strings"
)

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

// PorcelainCounts parses the output of `git status --porcelain` and returns
// (tracked, untracked) counts. tracked covers every line that does not start
// with "??" (i.e. any staged or unstaged change to a tracked path); untracked
// covers lines beginning with "??".
func PorcelainCounts(repo string) (tracked, untracked int, err error) {
	stdout, err := StatusPorcelain(repo)
	if err != nil {
		return 0, 0, err
	}
	if stdout == "" {
		return 0, 0, nil
	}
	for _, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "??") {
			untracked++
		} else {
			tracked++
		}
	}
	return tracked, untracked, nil
}

// AheadBehind reports how many commits the current branch is ahead of and
// behind its upstream. hasUpstream is false (with all counts zero, err nil)
// when no upstream is configured. Non-nil err only on spawn failure.
func AheadBehind(repo string) (ahead, behind int, hasUpstream bool, err error) {
	_, _, code, err := run(repo, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return 0, 0, false, err
	}
	if code != 0 {
		return 0, 0, false, nil
	}
	stdout, _, code, err := run(repo, "rev-list", "--left-right", "--count", "@{u}...HEAD")
	if err != nil {
		return 0, 0, true, err
	}
	if code != 0 {
		return 0, 0, true, nil
	}
	fields := strings.Fields(stdout)
	if len(fields) != 2 {
		return 0, 0, true, nil
	}
	b, err1 := parseUint(fields[0])
	a, err2 := parseUint(fields[1])
	if err1 != nil || err2 != nil {
		return 0, 0, true, nil
	}
	return a, b, true, nil
}

// DiffShortStat returns counts from `git diff HEAD --shortstat`, covering
// both staged and unstaged tracked changes. Untracked files are not included.
// Returns zeros when the working tree is clean. Non-nil err only on spawn failure.
func DiffShortStat(repo string) (files, insertions, deletions int, err error) {
	stdout, _, code, err := run(repo, "diff", "HEAD", "--shortstat")
	if err != nil {
		return 0, 0, 0, err
	}
	if code != 0 || stdout == "" {
		return 0, 0, 0, nil
	}
	for _, part := range strings.Split(stdout, ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.Contains(part, "file"):
			n, _ := parseUint(strings.Fields(part)[0])
			files = n
		case strings.Contains(part, "insertion"):
			n, _ := parseUint(strings.Fields(part)[0])
			insertions = n
		case strings.Contains(part, "deletion"):
			n, _ := parseUint(strings.Fields(part)[0])
			deletions = n
		}
	}
	return files, insertions, deletions, nil
}

func parseUint(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("non-digit")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
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

// WorktreeAddNew creates a new branch and a linked worktree at path.
// Runs `git worktree add -b <branch> <path> <base>`. An empty base
// substitutes "HEAD".
func WorktreeAddNew(repo, path, branch, base string) error {
	if base == "" {
		base = "HEAD"
	}
	_, stderr, code, err := run(repo, "worktree", "add", "-b", branch, path, base)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("git worktree add failed in %s: %s", repo, stderr)
	}
	return nil
}

// WorktreeAddExisting checks out an existing branch into a new linked
// worktree at path. Runs `git worktree add <path> <branch>`.
func WorktreeAddExisting(repo, path, branch string) error {
	_, stderr, code, err := run(repo, "worktree", "add", path, branch)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("git worktree add failed in %s: %s", repo, stderr)
	}
	return nil
}

// WorktreeRemove removes a linked worktree at wtPath without --force.
// Runs `git worktree remove <wtPath>`.
func WorktreeRemove(repo, wtPath string) error {
	_, stderr, code, err := run(repo, "worktree", "remove", wtPath)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("git worktree remove failed in %s: %s", repo, stderr)
	}
	return nil
}

// WorktreeRemoveForce removes a linked worktree at wtPath, forcing removal
// even if it has uncommitted changes. Runs `git worktree remove --force <wtPath>`.
func WorktreeRemoveForce(repo, wtPath string) error {
	_, stderr, code, err := run(repo, "worktree", "remove", "--force", wtPath)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("git worktree remove failed in %s: %s", repo, stderr)
	}
	return nil
}

// WorktreePrune removes worktree entries whose directories no longer exist.
// Best-effort: nonzero git exit codes are swallowed (matches the TS
// `await run(...)` without exit-code check). Spawn errors (e.g. git binary
// missing) are still returned, since silently dropping those would be a
// foot-gun.
func WorktreePrune(repo string) error {
	_, _, _, err := run(repo, "worktree", "prune")
	return err
}

// DeleteBranch deletes a fully-merged branch via `git branch -d <branch>`.
// Returns an error if the branch is not fully merged (use DeleteBranchForce
// to bypass).
func DeleteBranch(repo, branch string) error {
	_, stderr, code, err := run(repo, "branch", "-d", branch)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("git branch -d failed in %s: %s", repo, stderr)
	}
	return nil
}

// DeleteBranchForce deletes a branch via `git branch -D <branch>`,
// regardless of merge state.
func DeleteBranchForce(repo, branch string) error {
	_, stderr, code, err := run(repo, "branch", "-D", branch)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("git branch -D failed in %s: %s", repo, stderr)
	}
	return nil
}
