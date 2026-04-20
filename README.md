# wtmux — coordinated git worktrees for AI coding agents

**One command, one branch name, every sibling repo in sync — and your AI agent launches with all of them attached.**

`wtmux` creates matching git worktrees across every repo you care about, symlinks `node_modules` / `.env` / whatever else you need, then launches your AI agent (Claude Code, Codex, Cursor, OpenCode, Qoder, or any other CLI) with the sibling repos already attached.

If you've ever run an AI agent on an API repo with the frontend repo attached, checked out a feature branch, and realized the frontend is still on `main` — this is the fix.

---

## Why wtmux

AI coding agents can work across multiple repos in one session — Claude Code's `--add-dir`, Codex's `--add-dir`, Cursor's `--add`, and so on. But git worktrees are per-repo: when you branch off in the primary, the attached repos stay on their old branches. You end up coding against a feature branch in one repo and a stale `main` in the others.

No AI agent has a hook or config to rewrite attached directory paths mid-session, so this has to be fixed before the agent starts. `wtmux` replaces the worktree-creation step:

1. **Creates matching worktrees** in every configured sibling repo — same branch name, same base branch.
2. **Replicates symlinks** (`node_modules`, `.env`, etc.) from each origin into its new worktree, so dev servers and environment config just work.
3. **Launches your agent** with every non-primary worktree attached via whichever flag the agent expects (`--add-dir`, `--add`, etc.).

## Features

- 🌳 **Coordinated worktrees** across a configured group of sibling repos
- 🧠 **Multi-agent** — built-in support for `claude`, `codex`, `cursor`, `code`, `opencode`, `qoder`; `addDirArgs` config for any other CLI
- 🔗 **Symlink replication** for `node_modules`, `.env`, or any path — globs supported
- 🚀 **One-shot launch** with siblings auto-attached via the right flag
- 🛡️ **Safe teardown** — `wtmux rm` refuses to remove worktrees with uncommitted changes, stashes, or unpushed commits (`--force` overrides)
- 🔍 **Preflight validation** — validates branch names, worktree roots, and detects conflicts before mutating anything
- ♻️ **Automatic rollback** when a worktree creation fails partway through the group
- 📦 **Single-repo fallback** — works outside configured groups too
- 🔒 **Zero network, zero telemetry** — local-only tool

## Install

```bash
pnpm add -g wtmux
# or
npm install -g wtmux
# or
yarn global add wtmux
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

That creates `~/code/myapp-api/.worktrees/feat/login` and `~/code/myapp-web/.worktrees/feat/login`, both checked out on `feat/login` branched from the primary's current branch, both with `node_modules` and `.env` symlinked from their origins — and launches your agent (Claude Code by default) with the sibling attached.

When you're done:

```bash
wtmux rm feat/login
```

`rm` refuses if any side is dirty, stashed, or unpushed. Pass `--force` to override.

## Commands

| Command | Purpose |
|---|---|
| `wtmux <name>` | Create coordinated worktrees on branch `<name>` and launch the agent |
| `wtmux rm <name>` | Remove coordinated worktrees (refuses on dirty / stashed / unpushed; `--force` overrides) |
| `wtmux ls` | List coordinated worktrees across the current group, with per-repo state |
| `wtmux config` | Interactively edit the wtmux config (create/edit/delete groups) |

## Flags

| Flag | Short | Purpose |
|---|---|---|
| `--config <path>` | `-c` | Override config discovery |
| `--group <name>` | `-g` | Override auto-detected group |
| `--base <branch>` | `-b` | Override the base branch (create only) |
| `--dry-run` | `-n` | Print the plan without mutating |
| `--no-launch` | — | Skip launching the agent at the end of `create` |
| `--force` | `-f` | `rm` only: skip dirty/stash/unpushed guards |
| `--verbose` | `-v` | Extra logging |
| `--version` | `-V` | Print version |
| `--help` | `-h` | Print help |

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
    },
    {
      "name": "myapp-codex",
      "repos": ["~/code/myapp-api", "~/code/myapp-web"],
      "launchCommand": ["codex"]
    },
    {
      "name": "myapp-custom",
      "repos": ["~/code/myapp-api", "~/code/myapp-web"],
      "launchCommand": ["my-agent"],
      "addDirArgs": ["--include", "{path}"]
    }
  ]
}
```

| Field | Purpose |
|---|---|
| `symlinkDirectories` | Paths to symlink from each repo root into each new worktree. Defaults to `["node_modules", ".env"]`. |
| `worktreePathPattern` | Where worktrees land inside each repo. `{name}` interpolates the worktree name. Defaults to `.worktrees/{name}`. |
| `launchCommand` | Argv to exec after worktrees are created. Defaults to `["claude"]`. |
| `agent` | Force a specific agent's sibling-injection rules (`claude` \| `codex` \| `cursor` \| `code` \| `opencode` \| `qoder`). Only needed if your `launchCommand` uses a wrapper or alias that wtmux can't detect from its basename. |
| `addDirArgs` | Argv template for CLIs wtmux doesn't know about. `{path}` is interpolated with each sibling's absolute path. See *Agents & sibling injection* below. |
| `groups[].*` | Any of the above (except `groups`) can appear per-group and overrides the top-level value. |

### Agents & sibling injection

Built-in agents and the flag wtmux injects per sibling:

| Agent | Flag per sibling |
|---|---|
| `claude`, `codex` | `--add-dir <path>` |
| `cursor`, `code` | `--add <path>` |
| `opencode`, `qoder` | *no multi-root support* — see below |

**Built-in agents auto-detect.** wtmux matches the basename of `launchCommand[0]` against the list above, so `claude`, `/usr/local/bin/claude`, and `~/.local/bin/claude` all work the same. Qoder's CLI is named `qodercli` — wtmux aliases it automatically.

If you invoke an agent through a wrapper script or alias that wtmux can't recognize (e.g. `launchCommand: ["my-claude-wrapper"]`), set `agent: "claude"` to force the right injection rules.

**Custom CLIs** go through `addDirArgs` — an argv template where `{path}` is replaced with each sibling's absolute path:

```json
{
  "launchCommand": ["my-agent"],
  "addDirArgs": ["--workdir", "{path}"]
}
```

With two siblings, this expands to:

```
my-agent --workdir /abs/sibling1 --workdir /abs/sibling2
```

**Agents without multi-root support** (`opencode`, `qoder`): wtmux creates the worktrees and replicates symlinks, then prints the worktree paths and skips the launch — the same as passing `--no-launch`. Start the agent yourself in whichever worktree you want.

### Glob patterns in `symlinkDirectories`

Items in `symlinkDirectories` can be literal paths or glob patterns. Anything containing `*`, `?`, `[`, `]`, `{`, or `}` is treated as a glob.

```json
{
  "symlinkDirectories": ["node_modules", ".env*", "config/*.json"]
}
```

Patterns resolve relative to each repo's root. Dotfiles are included by default, so `.env*` matches `.env`, `.env.local`, `.env.development`, etc. `**` is supported for recursion.

## FAQ

1. **How do I create a config without hand-writing JSON?**

   Run `wtmux config` from anywhere. It auto-detects sibling git repos in the parent directory and walks you through creating a group.

2. **Which AI agents does wtmux support out of the box?**

   `claude`, `codex`, `cursor`, `code`, `opencode`, `qoder`. For anything else, set `addDirArgs` in config — see *Agents & sibling injection* above.

3. **Can I use it with only one repo?**

   Yes. Run `wtmux <name>` from inside any git repo that isn't in a configured group — you get single-repo worktree creation + symlinks + agent launch, no siblings.

4. **What happens if the worktrees already exist?**

   `wtmux` refuses to create over existing worktrees. Use `wtmux rm <name>` first, or pick a different name.

5. **How do I remove a worktree with uncommitted work?**

   Commit, stash, or push first — or pass `--force`. `wtmux rm` intentionally refuses to silently discard work.

6. **What's the difference from `git worktree add`?**

   `wtmux` coordinates the same worktree across multiple repos, replicates symlinks into each, and launches your agent with siblings attached — in one command. Plain `git worktree add` handles one repo at a time and doesn't touch symlinks or your agent.

7. **Does it support Windows?**

   Not yet. macOS and Linux only. PRs welcome.

8. **Does it send anything over the network?**

   No. `wtmux` is a local tool — no telemetry, no update checks, no API calls.

9. **What happens if I run `wtmux` with no name?**

   It generates a random, memorable name like `wt/brave-penguin` and creates coordinated worktrees on that branch. The generated name is printed to stderr so you can find it later via `wtmux rm wt/brave-penguin`.

## Status

v0.4.0 — stable for personal use.

Exit codes follow Unix conventions: `0` success, `1` user error, `2` precondition failure, `3` internal error.

## License

MIT © oysterlee
