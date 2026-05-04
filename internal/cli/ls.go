package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"

	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
	"github.com/OysterD3/wtmux/internal/git"
	"github.com/OysterD3/wtmux/internal/group"
	"github.com/OysterD3/wtmux/internal/paths"
)

// newLsCmd returns the `wtmux ls` cobra command.
func newLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ls",
		Short:         "List coordinated worktrees in the current group",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyVerbose()
			return runLs()
		},
	}
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		helpToStdout(lsHelp)
	})
	return cmd
}

type lsRepoStatus struct {
	repo        string
	present     bool
	wtPath      string
	tracked     int
	untracked   int
	files       int
	insertions  int
	deletions   int
	ahead       int
	behind      int
	hasUpstream bool
}

type lsRow struct {
	name    string
	perRepo []lsRepoStatus
}

func runLs() error {
	cwd, err := mustGetwd()
	if err != nil {
		return err
	}

	cfg, err := loadConfigOrDefault(cwd)
	if err != nil {
		return err
	}

	resolved, err := group.Resolve(group.ResolveInput{
		Cwd:       cwd,
		Config:    cfg,
		GroupFlag: pf.group,
	})
	if err != nil {
		return wtmuxerrors.Wrapf(wtmuxerrors.KindUser, err, "%s", err.Error())
	}
	if resolved.Kind == group.KindOutside {
		return wtmuxerrors.New(wtmuxerrors.KindPrecondition, "cwd is not inside any git repository")
	}

	repos := []string{}
	if resolved.Kind == group.KindGroup {
		repos = append(repos, resolved.Group.Repos...)
	} else {
		repos = append(repos, resolved.Single)
	}

	type repoWts struct {
		repo     string // original (config-form) path, used for display + git commands
		repoReal string // symlink-resolved path, used to compare against `git worktree list` output
		wts      []git.WorktreeEntry
	}
	perRepoWts := make([]repoWts, 0, len(repos))
	for _, r := range repos {
		wts, err := git.ListWorktrees(r)
		if err != nil {
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "git worktree list failed in %s: %s", r, err.Error())
		}
		perRepoWts = append(perRepoWts, repoWts{repo: r, repoReal: paths.RealpathSafe(r), wts: wts})
	}

	nameSet := map[string]struct{}{}
	for _, rw := range perRepoWts {
		for _, w := range rw.wts {
			if w.Branch == "" {
				continue
			}
			if w.Worktree == rw.repoReal {
				continue
			}
			nameSet[w.Branch] = struct{}{}
		}
	}
	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}
	sort.Strings(names)

	rows := make([]lsRow, 0, len(names))
	for _, name := range names {
		row := lsRow{name: name, perRepo: make([]lsRepoStatus, 0, len(perRepoWts))}
		for _, rw := range perRepoWts {
			var match *git.WorktreeEntry
			for i := range rw.wts {
				w := &rw.wts[i]
				if w.Branch == name && w.Worktree != rw.repoReal {
					match = w
					break
				}
			}
			if match == nil {
				row.perRepo = append(row.perRepo, lsRepoStatus{repo: rw.repo, present: false})
				continue
			}
			st, err := collectStatus(rw.repo, match.Worktree)
			if err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
			row.perRepo = append(row.perRepo, st)
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		fmt.Fprintln(os.Stdout, "(no coordinated worktrees)")
		return nil
	}

	useColor := isatty.IsTerminal(os.Stdout.Fd())
	multiRepo := len(perRepoWts) > 1
	renderTable(os.Stdout, rows, multiRepo, useColor)
	return nil
}

func collectStatus(repo, wtPath string) (lsRepoStatus, error) {
	st := lsRepoStatus{repo: repo, present: true, wtPath: wtPath}
	tracked, untracked, err := git.PorcelainCounts(wtPath)
	if err != nil {
		return st, err
	}
	st.tracked = tracked
	st.untracked = untracked

	files, ins, dels, err := git.DiffShortStat(wtPath)
	if err != nil {
		return st, err
	}
	st.files = files
	st.insertions = ins
	st.deletions = dels

	a, b, has, err := git.AheadBehind(wtPath)
	if err != nil {
		return st, err
	}
	st.ahead = a
	st.behind = b
	st.hasUpstream = has
	return st, nil
}

// ANSI color helpers. All emit empty strings when useColor is false so the
// rendering code stays linear regardless of TTY status.
type styler struct{ on bool }

func (s styler) wrap(code, text string) string {
	if !s.on || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}
func (s styler) bold(t string) string    { return s.wrap("1", t) }
func (s styler) dim(t string) string     { return s.wrap("2", t) }
func (s styler) red(t string) string     { return s.wrap("31", t) }
func (s styler) green(t string) string   { return s.wrap("32", t) }
func (s styler) yellow(t string) string  { return s.wrap("33", t) }
func (s styler) blue(t string) string    { return s.wrap("34", t) }
func (s styler) magenta(t string) string { return s.wrap("35", t) }
func (s styler) cyan(t string) string    { return s.wrap("36", t) }

// visualWidth returns the display width of s, ignoring ANSI escape sequences
// and using go-runewidth so wide / combining / zero-width characters are
// counted correctly. Used for column padding.
func visualWidth(s string) int {
	var b strings.Builder
	in := false
	for _, r := range s {
		if in {
			if r == 'm' {
				in = false
			}
			continue
		}
		if r == 0x1b {
			in = true
			continue
		}
		b.WriteRune(r)
	}
	return runewidth.StringWidth(b.String())
}

func pad(s string, width int) string {
	w := visualWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// stateGlyph returns a colored single-character marker summarizing the
// worktree's working-tree state. Upstream divergence is shown by the sync
// column (↑N / ↓N), not here — keeping the glyph dimensions disjoint
// avoids users seeing two arrows on the same row.
func stateGlyph(s lsRepoStatus, sty styler) string {
	if !s.present {
		return sty.dim("·")
	}
	if s.tracked > 0 {
		return sty.yellow("●")
	}
	if s.untracked > 0 {
		return sty.magenta("◌")
	}
	return sty.green("○")
}

// changes renders modified/untracked/insertions/deletions for one repo,
// returning "" when there's nothing to show.
func changes(s lsRepoStatus, sty styler) string {
	parts := []string{}
	if s.files > 0 {
		unit := "files"
		if s.files == 1 {
			unit = "file"
		}
		parts = append(parts, sty.bold(fmt.Sprintf("%d %s", s.files, unit)))
	}
	if s.insertions > 0 {
		parts = append(parts, sty.green(fmt.Sprintf("+%d", s.insertions)))
	}
	if s.deletions > 0 {
		parts = append(parts, sty.red(fmt.Sprintf("-%d", s.deletions)))
	}
	if s.untracked > 0 {
		parts = append(parts, sty.magenta(fmt.Sprintf("↳%d", s.untracked)))
	}
	return strings.Join(parts, " ")
}

// sync renders ahead/behind arrows, returning "" when nothing notable.
func sync(s lsRepoStatus, sty styler) string {
	if !s.present {
		return ""
	}
	if !s.hasUpstream {
		return sty.dim("no upstream")
	}
	if s.ahead == 0 && s.behind == 0 {
		return ""
	}
	out := []string{}
	if s.ahead > 0 {
		out = append(out, sty.cyan(fmt.Sprintf("↑%d", s.ahead)))
	}
	if s.behind > 0 {
		out = append(out, sty.yellow(fmt.Sprintf("↓%d", s.behind)))
	}
	return strings.Join(out, " ")
}

func renderTable(w *os.File, rows []lsRow, multiRepo, useColor bool) {
	sty := styler{on: useColor}

	// For single-repo groups we drop the repo column. For multi-repo groups
	// we render one line per repo under each branch heading.
	if multiRepo {
		for _, row := range rows {
			fmt.Fprintln(w, sty.bold(row.name))
			for _, s := range row.perRepo {
				glyph := stateGlyph(s, sty)
				label := filepath.Base(s.repo)
				if !s.present {
					fmt.Fprintf(w, "  %s %s %s\n", glyph, pad(label, 18), sty.dim("—"))
					continue
				}
				ch := changes(s, sty)
				sy := sync(s, sty)
				cols := []string{ch, sy}
				body := strings.TrimSpace(joinNonEmpty(cols, "  "))
				if body == "" {
					body = sty.dim("clean")
				}
				fmt.Fprintf(w, "  %s %s %s\n", glyph, pad(label, 18), body)
			}
		}
		return
	}

	// Single-repo group: one row per branch.
	type cell struct {
		glyph, name, ch, sy string
	}
	cells := make([]cell, 0, len(rows))
	maxName := 0
	maxCh := 0
	for _, row := range rows {
		s := row.perRepo[0]
		c := cell{
			glyph: stateGlyph(s, sty),
			name:  row.name,
		}
		if !s.present {
			c.ch = sty.dim("—")
		} else {
			c.ch = changes(s, sty)
			c.sy = sync(s, sty)
			if c.ch == "" && c.sy == "" {
				c.ch = sty.dim("clean")
			}
		}
		cells = append(cells, c)
		if vw := visualWidth(c.name); vw > maxName {
			maxName = vw
		}
		if vw := visualWidth(c.ch); vw > maxCh {
			maxCh = vw
		}
	}
	for _, c := range cells {
		fmt.Fprintf(w, "  %s  %s  %s%s\n",
			c.glyph,
			pad(sty.bold(c.name), maxName),
			pad(c.ch, maxCh),
			withLeadingSpace(c.sy),
		)
	}
}

func withLeadingSpace(s string) string {
	if s == "" {
		return ""
	}
	return "  " + s
}

func joinNonEmpty(parts []string, sep string) string {
	out := []string{}
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}
