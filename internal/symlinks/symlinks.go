// Package symlinks replicates per-item symlinks from a repo into a worktree
// (and removes them on teardown). Each item is processed independently;
// missing sources and pre-existing targets are reported via SymlinkResult
// rather than as errors. The package is the project's first FS-mutating
// primitive — its TempDir-based test pattern is the template for later
// FS-touching packages (git, group).
package symlinks

import (
	"os"
	"path/filepath"
)

// SymlinkAction is the per-item outcome of Replicate.
type SymlinkAction string

const (
	// ActionLinked means the symlink was created.
	ActionLinked SymlinkAction = "linked"
	// ActionSkippedNoSource means the source path under repo did not exist;
	// nothing was created and nothing is wrong — the caller decides whether
	// to surface this as an error.
	ActionSkippedNoSource SymlinkAction = "skipped-no-source"
	// ActionSkippedTargetExists means the target path already existed
	// (regular file, directory, or pre-existing symlink); we never overwrite.
	ActionSkippedTargetExists SymlinkAction = "skipped-target-exists"
)

// SymlinkResult is one entry in the action log returned by Replicate.
type SymlinkResult struct {
	Item   string
	Action SymlinkAction
}

// ReplicateInputs collects the inputs Replicate needs.
type ReplicateInputs struct {
	Repo  string
	WT    string
	Items []string
}

// Replicate creates symlinks under in.WT pointing at the corresponding paths
// under in.Repo, one per entry in in.Items. Each item produces exactly one
// SymlinkResult in the returned slice (in input order). Missing sources and
// pre-existing targets are reported via Action; non-IsNotExist errors
// propagate immediately and the caller is responsible for any rollback. On
// error the returned slice is nil; callers wanting to undo partial work
// should pass the full Items slice to Remove (the anti-foot-gun there
// skips entries that aren't symlinks).
func Replicate(in ReplicateInputs) ([]SymlinkResult, error) {
	results := make([]SymlinkResult, 0, len(in.Items))
	for _, item := range in.Items {
		source := filepath.Join(in.Repo, item)
		target := filepath.Join(in.WT, item)

		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				results = append(results, SymlinkResult{Item: item, Action: ActionSkippedNoSource})
				continue
			}
			return nil, err
		}

		if _, err := os.Lstat(target); err == nil {
			results = append(results, SymlinkResult{Item: item, Action: ActionSkippedTargetExists})
			continue
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		if err := os.Symlink(source, target); err != nil {
			return nil, err
		}
		results = append(results, SymlinkResult{Item: item, Action: ActionLinked})
	}
	return results, nil
}

// Remove deletes the symlink at filepath.Join(wt, item) for each item — but
// only if the target exists and is itself a symlink. Regular files and
// directories at the target are left untouched (anti-foot-gun: a stale
// config entry should never destroy real user content). Missing targets
// are no-ops; non-IsNotExist errors propagate.
func Remove(wt string, items []string) error {
	for _, item := range items {
		target := filepath.Join(wt, item)
		info, err := os.Lstat(target)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		if err := os.Remove(target); err != nil {
			return err
		}
	}
	return nil
}
