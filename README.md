# wtmux — coordinated git worktrees for Claude Code and multi-repo monorepos

**One command, one branch name, every sibling repo in sync — and Claude Code launches with all of them wired up.**

`wtmux` creates matching git worktrees across every repo you care about, symlinks `node_modules` / `.env` / whatever else you need, then launches [Claude Code](https://claude.com/claude-code) (or your editor of choice) with the siblings already mounted as `--add-dir` paths.

If you've ever run Claude Code on an API repo with a frontend `--add-dir`'d in, checked out a feature branch, and realized the frontend is still on `main` — this is the fix.

---

## Why wtmux

Claude Code's native `/worktree` command creates a worktree for the primary working directory **only**. Any repo you added with `--add-dir` stays on whatever branch it was on. You end up working on an isolated feature branch in one repo while the other is stale.

Claude Code has no hook or config to rewrite `--add-dir` paths mid-session, so this can't be fixed with a hook. `wtmux` replaces the worktree-creation step with a wrapper that:

1. **Creates matching worktrees** in every configured sibling repo — same branch name, same base branch.
2. **Replicates symlinks** (`node_modules`, `.env`, etc.) from each origin into its new worktree, so dev servers and environment config just work.
3. **Launches `claude`** with every non-primary worktree wired up as an `--add-dir` path — so your Claude Code session sees a coherent multi-repo workspace from the start.

## Features

- 🌳 **Coordinated worktrees** across a configured group of sibling repos
- 🔗 **Symlink replication** that mirrors Claude Code's `worktree.symlinkDirectories`
- 🚀 **One-shot Claude Code launch** with siblings auto-registered as `--add-dir`
- 🛡️ **Safe teardown** — `wtmux rm` refuses to remove worktrees with uncommitted changes, stashes, or unpushed commits (override with `--force`)
- 🔍 **Preflight validation** — checks branch names via `git check-ref-format`, verifies worktree roots, detects branch conflicts before mutating anything
- ♻️ **Automatic rollback** when a worktree creation fails partway through the group
- 📦 **Single-repo fallback** — works outside configured groups too
- 🖇️ **Editor-agnostic** — swap `claude` for `code`, `cursor`, or any editor via `launchCommand`
- 🔒 **Zero network, zero telemetry** — local-only tool

## Install

```bash
pnpm add -g wtmux
# or
npm install -g wtmux
# or
yarn global add wtmux
```

From source:

```bash
git clone https://github.com/OysterD3/wtmux.git
cd wtmux
pnpm install
pnpm build
pnpm link --global
```

Requirements: **Node.js 20+** and **git**.

## Quick start

Add a config at `~/.config/wtmux/config.json` (or `.wtmux.json` in your monorepo root):

```json
{
  "groups": [
    {
      "name": "myapp",
      "repos": [
        "~/code/myapp-api",
        "~/code/myapp-web"
      ]
    }
  ]
}
```

Then, from inside either repo:

```bash
wtmux feat/login
```

That creates `~/code/myapp-api/.worktrees/feat/login` and `~/code/myapp-web/.worktrees/feat/login`, both checked out on `feat/login` branched from the primary's current branch, both with `node_modules` and `.env` symlinked from their origins — and launches `claude` with the sibling wired up as `--add-dir`.

When you're done:

```bash
wtmux rm feat/login
```

`rm` refuses if any side is dirty, stashed, or unpushed. Pass `--force` to override.

## Commands

| Command | Purpose |
|---|---|
| `wtmux <name>` | Create coordinated worktrees on branch `<name>` and launch Claude Code |
| `wtmux rm <name>` | Remove coordinated worktrees (refuses on dirty / stashed / unpushed; `--force` overrides) |
| `wtmux ls` | List coordinated worktrees across the current group, with per-repo state |
| `wtmux config` | Interactively edit the wtmux config (create/edit/delete groups) |

## Flags

| Flag | Purpose |
|---|---|
| `--config <path>` | Override config discovery |
| `--group <name>` | Override auto-detected group (useful from arbitrary cwd) |
| `--dry-run` | Print the plan without mutating |
| `--no-launch` | Skip launching Claude Code at the end of `create` |
| `--force` | `rm` only: skip dirty/stash/unpushed guards |
| `-v`, `--verbose` | Extra logging |
| `--version` | Print version |
| `--help` | Print help |

## Configuration

Config lookup order (first match wins):

1. Path passed via `--config <path>`
2. `$WTMUX_CONFIG` environment variable
3. `.wtmux.json` found by walking upward from cwd
4. `~/.config/wtmux/config.json`

### Full schema

```json
{
  "symlinkDirectories": ["node_modules", ".env"],
  "worktreePathPattern": ".worktrees/{name}",
  "launchCommand": ["claude"],
  "groups": [
    {
      "name": "myapp",
      "repos": [
        "~/code/myapp-api",
        "~/code/myapp-web"
      ],
      "symlinkDirectories": ["node_modules", ".env", ".env.local"]
    }
  ]
}
```

| Field | Purpose |
|---|---|
| `symlinkDirectories` | Default paths to symlink from each repo root into each new worktree. Defaults to `["node_modules", ".env"]`. |
| `worktreePathPattern` | Where worktrees land inside each repo. `{name}` interpolates the worktree name. Defaults to `.worktrees/{name}`. |
| `launchCommand` | Argv to exec after worktrees are created. Defaults to `["claude"]`. Override for other editors. |
| `groups[].name` | Unique group identifier. |
| `groups[].repos` | Absolute or tilde-prefixed paths (min 2 repos — coordination needs siblings). |
| `groups[].symlinkDirectories` | Per-group override (replaces the top-level list). |
| `groups[].worktreePathPattern` | Per-group override. |
| `groups[].launchCommand` | Per-group override. |

`wtmux` appends `--add-dir <sibling-wt>` flags automatically **only when `launchCommand[0] === "claude"`**. For other editors, sibling worktree paths are appended as positional arguments.

## FAQ

**How do I create a config without hand-writing JSON?**
Run `wtmux config` from anywhere. It auto-detects sibling git repos in the parent directory and walks you through creating a group, with options to edit or delete groups later.

**Does `wtmux` work without Claude Code?**
Yes. Set `"launchCommand": ["code", "."]` (VS Code) or `["cursor", "."]` (Cursor) or any other editor. Claude Code's `--add-dir` flag injection only fires when the launch command is literally `claude`.

**Can I use it with only one repo?**
Yes. Run `wtmux <name>` from inside any git repo that isn't in a configured group — you get single-repo worktree creation + symlinks + Claude launch, no siblings.

**What happens if the worktrees already exist?**
`wtmux` refuses to create over existing worktrees. Use `wtmux rm <name>` first, or pick a different name.

**How do I remove a worktree with uncommitted work?**
Commit, stash, or push first — or pass `--force`. `wtmux rm` intentionally refuses to silently discard work.

**What's the difference from `git worktree add`?**
`wtmux` coordinates the same worktree across multiple repos, replicates symlinks into each, and launches Claude Code with them wired up — in one command. Plain `git worktree add` handles one repo at a time and doesn't touch symlinks or your editor.

**Does it support Windows?**
Not yet. macOS and Linux only. PRs welcome.

**Does it send anything over the network?**
No. `wtmux` is a local tool — no telemetry, no updates checks, no API calls.

## Status

v0.2.0 — stable for personal use.

Exit codes follow Unix conventions: `0` success, `1` user error, `2` precondition failure, `3` internal error.

## Contributing

Bug reports and PRs welcome at [github.com/OysterD3/wtmux](https://github.com/OysterD3/wtmux). The test suite uses real git repos in tmpdirs — `pnpm test` runs ~80 tests in a few seconds.

## License

MIT © oysterlee
