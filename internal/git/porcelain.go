package git

import "strings"

// WorktreeEntry is one block in `git worktree list --porcelain`.
type WorktreeEntry struct {
	Worktree string
	Head     string
	Branch   string // "" if detached or branchless
	Detached bool
}

// parseWorktreePorcelain parses the output of `git worktree list --porcelain`.
// Entries are delimited by a blank line; recognized prefixes are
// "worktree ", "HEAD ", "branch refs/heads/", and the bare token "detached".
// Unknown lines are ignored. The parser is total — input that doesn't match
// the format yields entries with default zero values.
func parseWorktreePorcelain(s string) []WorktreeEntry {
	var entries []WorktreeEntry
	var cur *WorktreeEntry
	for _, line := range strings.Split(s, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if cur != nil {
				entries = append(entries, *cur)
			}
			cur = &WorktreeEntry{Worktree: line[len("worktree "):]}
		case strings.HasPrefix(line, "HEAD ") && cur != nil:
			cur.Head = line[len("HEAD "):]
		case strings.HasPrefix(line, "branch refs/heads/") && cur != nil:
			cur.Branch = line[len("branch refs/heads/"):]
		case line == "detached" && cur != nil:
			cur.Detached = true
		case line == "" && cur != nil:
			entries = append(entries, *cur)
			cur = nil
		}
	}
	if cur != nil {
		entries = append(entries, *cur)
	}
	return entries
}
