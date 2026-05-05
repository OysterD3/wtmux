package preflight

import (
	"errors"
	"fmt"

	"github.com/OysterD3/wtmux/internal/git"
	"github.com/OysterD3/wtmux/internal/paths"
)

// Sentinel errors aggregated by (*Plan).Validate via errors.Join. Each
// per-repo error is wrapped with the offending path or detail via
// fmt.Errorf("%w: ...", ErrX).
var (
	// ErrInvalidName is returned when git check-ref-format rejects the
	// worktree/branch name. Not per-repo: short-circuits the rest of
	// Validate when reported.
	ErrInvalidName = errors.New("preflight: invalid worktree name")
	// ErrNotWorktreeRoot wraps a repo path that is not a main worktree
	// root (most often: a linked worktree mistakenly pointed at).
	ErrNotWorktreeRoot = errors.New("preflight: not a worktree root")
	// ErrWtPathExists wraps a target worktree path that already exists.
	ErrWtPathExists = errors.New("preflight: target worktree path already exists")
	// ErrBranchCheckedOut wraps a repo where the target branch is already
	// checked out in another worktree.
	ErrBranchCheckedOut = errors.New("preflight: branch already checked out in another worktree")
	// ErrBaseMissing wraps a repo whose BaseBranch doesn't resolve.
	ErrBaseMissing = errors.New("preflight: base branch does not exist")
)

// Validate runs the live filesystem and git-state checks against p.
// Aggregates errors with errors.Join so a single call surfaces every
// problem. Returns nil on a clean plan.
func (p *Plan) Validate() error {
	ok, err := git.CheckRefFormat(p.Name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %q", ErrInvalidName, p.Name)
	}

	var errs []error
	for _, rp := range p.Repos {
		isRoot, err := git.IsWorktreeRoot(rp.Repo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !isRoot {
			errs = append(errs, fmt.Errorf("%w: %s", ErrNotWorktreeRoot, rp.Repo))
			continue
		}
		if paths.Exists(rp.WtPath) {
			errs = append(errs, fmt.Errorf("%w: %s", ErrWtPathExists, rp.WtPath))
		}

		exists, err := git.BranchExists(rp.Repo, p.Name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if exists {
			wts, err := git.ListWorktrees(rp.Repo)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, w := range wts {
				if w.Branch == p.Name {
					errs = append(errs, fmt.Errorf("%w: %s: branch %q in %s", ErrBranchCheckedOut, rp.Repo, p.Name, w.Worktree))
					break
				}
			}
		} else {
			hasBase, err := git.HasRef(rp.Repo, p.BaseBranch)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if !hasBase {
				errs = append(errs, fmt.Errorf("%w: %s: %q", ErrBaseMissing, rp.Repo, p.BaseBranch))
			}
		}
	}
	return errors.Join(errs...)
}
