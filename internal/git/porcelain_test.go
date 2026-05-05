package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWorktreePorcelain(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []WorktreeEntry
	}{
		{
			name: "empty input",
			in:   "",
			want: nil,
		},
		{
			name: "single entry on a branch",
			in:   "worktree /repo\nHEAD abc123\nbranch refs/heads/main\n",
			want: []WorktreeEntry{{Worktree: "/repo", Head: "abc123", Branch: "main"}},
		},
		{
			name: "single entry detached",
			in:   "worktree /repo\nHEAD abc123\ndetached\n",
			want: []WorktreeEntry{{Worktree: "/repo", Head: "abc123", Detached: true}},
		},
		{
			name: "single entry branchless (worktree + HEAD only)",
			in:   "worktree /repo\nHEAD abc123\n",
			want: []WorktreeEntry{{Worktree: "/repo", Head: "abc123"}},
		},
		{
			name: "multiple entries separated by blank lines",
			in: "worktree /repo\nHEAD aaa\nbranch refs/heads/main\n\n" +
				"worktree /repo/wt-feat\nHEAD bbb\nbranch refs/heads/feat\n",
			want: []WorktreeEntry{
				{Worktree: "/repo", Head: "aaa", Branch: "main"},
				{Worktree: "/repo/wt-feat", Head: "bbb", Branch: "feat"},
			},
		},
		{
			name: "trailing blank line same as no trailing blank line",
			in:   "worktree /repo\nHEAD aaa\nbranch refs/heads/main\n\n",
			want: []WorktreeEntry{{Worktree: "/repo", Head: "aaa", Branch: "main"}},
		},
		{
			name: "unknown lines ignored",
			in:   "worktree /repo\nlocked\nHEAD aaa\nprunable foo\nbranch refs/heads/main\n",
			want: []WorktreeEntry{{Worktree: "/repo", Head: "aaa", Branch: "main"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseWorktreePorcelain(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}
