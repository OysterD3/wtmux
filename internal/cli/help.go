package cli

import (
	"fmt"
	"io"
	"os"
)

// rootHelp prints the top-level help. Subcommands set their own help
// functions so they don't inherit this verbose root view.
func rootHelp(w io.Writer) {
	fmt.Fprintf(w, "wtmux v%s — coordinated git worktrees across sibling repos\n", Version())
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  wtmux new <name> [flags] [-- agent-args...]   Create worktrees and launch the agent")
	fmt.Fprintln(w, "  wtmux ls                                      List worktrees in the current group")
	fmt.Fprintln(w, "  wtmux rm <name> [flags]                       Remove worktrees created with wtmux")
	fmt.Fprintln(w, "  wtmux config                                  Edit the wtmux config interactively")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GLOBAL FLAGS")
	fmt.Fprintln(w, "  -c, --config <path>    Override config discovery")
	fmt.Fprintln(w, "  -g, --group <name>     Override the auto-detected group")
	fmt.Fprintln(w, "  -v, --verbose          Extra logging")
	fmt.Fprintln(w, "      --version          Print version")
	fmt.Fprintln(w, "  -h, --help             Show this help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "EXAMPLES")
	fmt.Fprintln(w, "  wtmux new feat/auth                Create worktrees on branch feat/auth")
	fmt.Fprintln(w, "  wtmux new feat/auth -b main        ... based on main")
	fmt.Fprintln(w, "  wtmux new feat/auth -- --print     Pass extra args through to the launched agent")
	fmt.Fprintln(w, "  wtmux ls                           Status of every coordinated worktree")
	fmt.Fprintln(w, "  wtmux rm feat/auth                 Remove the worktrees and delete the branch")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run `wtmux <command> --help` for command-specific flags.")
	fmt.Fprintln(w)
}

func newHelp(w io.Writer) {
	fmt.Fprintln(w, "wtmux new — Create coordinated worktrees and launch the agent")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  wtmux new <name> [flags] [-- agent-args...]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "FLAGS")
	fmt.Fprintln(w, "  -b, --base <branch>    Base branch for the new worktrees (default: HEAD)")
	fmt.Fprintln(w, "  -n, --dry-run          Print the plan without changing anything")
	fmt.Fprintln(w, "      --no-launch        Don't launch the agent after creating worktrees")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GLOBAL FLAGS")
	fmt.Fprintln(w, "  -c, --config <path>    Override config discovery")
	fmt.Fprintln(w, "  -g, --group <name>     Override the auto-detected group")
	fmt.Fprintln(w, "  -v, --verbose          Extra logging")
	fmt.Fprintln(w, "  -h, --help             Show this help")
	fmt.Fprintln(w)
}

func lsHelp(w io.Writer) {
	fmt.Fprintln(w, "wtmux ls — List coordinated worktrees in the current group")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  wtmux ls [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GLOBAL FLAGS")
	fmt.Fprintln(w, "  -c, --config <path>    Override config discovery")
	fmt.Fprintln(w, "  -g, --group <name>     Override the auto-detected group")
	fmt.Fprintln(w, "  -v, --verbose          Extra logging")
	fmt.Fprintln(w, "  -h, --help             Show this help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "OUTPUT")
	fmt.Fprintln(w, "  ●  tracked changes     ↳N  untracked files")
	fmt.Fprintln(w, "  ◌  untracked only      ↑N  commits ahead of upstream")
	fmt.Fprintln(w, "  ○  clean               ↓N  commits behind upstream")
	fmt.Fprintln(w, "  +N -N                  insertions / deletions vs HEAD")
	fmt.Fprintln(w)
}

func rmHelp(w io.Writer) {
	fmt.Fprintln(w, "wtmux rm — Remove coordinated worktrees")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  wtmux rm <name> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "FLAGS")
	fmt.Fprintln(w, "  -n, --dry-run          Print what would be removed")
	fmt.Fprintln(w, "  -f, --force            Skip dirty / unpushed guards")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GLOBAL FLAGS")
	fmt.Fprintln(w, "  -c, --config <path>    Override config discovery")
	fmt.Fprintln(w, "  -g, --group <name>     Override the auto-detected group")
	fmt.Fprintln(w, "  -v, --verbose          Extra logging")
	fmt.Fprintln(w, "  -h, --help             Show this help")
	fmt.Fprintln(w)
}

func configHelp(w io.Writer) {
	fmt.Fprintln(w, "wtmux config — Interactively edit wtmux config")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  wtmux config [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GLOBAL FLAGS")
	fmt.Fprintln(w, "  -c, --config <path>    Path to a specific config file")
	fmt.Fprintln(w, "  -v, --verbose          Extra logging")
	fmt.Fprintln(w, "  -h, --help             Show this help")
	fmt.Fprintln(w)
}

// helpToStdout is a tiny convenience used by SetHelpFunc, since cobra
// passes the command but not a writer to that callback.
func helpToStdout(fn func(io.Writer)) {
	fn(os.Stdout)
}
