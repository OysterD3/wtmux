# wtmux

Coordinated git worktrees across sibling repos, with auto-replicated symlinks (`node_modules`, `.env`) and a one-shot launch into `claude` (or your editor of choice) with all sibling worktrees wired up as additional directories.

**Status:** v0.1 — build from source (not yet published to npm). See [`docs/design.md`](docs/design.md) for the full design.

## Install

From source:

```bash
git clone <repo>
cd wtmux
pnpm install
pnpm build
pnpm link --global
```

Once published:

```bash
pnpm add -g wtmux
```

## The problem

Claude Code's `/worktree` command creates a worktree for the primary working directory only. If you have added a sibling repo via `--add-dir` (e.g. a frontend alongside a backend), that directory keeps pointing at the original path — on whatever branch it happened to be on. You end up working on an isolated feature branch in one repo while the other is stale.

Claude Code has no mechanism to rewrite `--add-dir` paths mid-session, so a pure hook-based fix isn't possible. `wtmux` replaces the worktree-creation step with a wrapper that:

1. Creates matching worktrees in every configured sibling repo (same branch name, same base branch).
2. Symlinks `node_modules` / `.env` / etc. into each worktree — replicating Claude's `worktree.symlinkDirectories` behavior.
3. Launches `claude` with sibling worktrees passed as `--add-dir`, so the in-session view of additional directories already points at the correct worktrees.

## Usage

```bash
# Create coordinated worktrees on branch feat/foo across the group, then launch claude.
wtmux feat/foo

# Remove them (refuses if any side has uncommitted/unpushed work).
wtmux rm feat/foo

# List coordinated worktrees in the current group.
wtmux ls
```

Config lives at `~/.config/wtmux/config.json` or `.wtmux.json` walking upward from cwd:

```json
{
  "symlinkDirectories": ["node_modules", ".env"],
  "groups": [
    {
      "name": "whatsapp-manager",
      "repos": [
        "/Users/you/whatsapp-manager/whatsapp-manager-api",
        "/Users/you/whatsapp-manager/whatsapp-manager-web"
      ]
    }
  ]
}
```

## Stack

TypeScript · Node 20+ · citty · `@clack/prompts` · execa · zod · tsup · vitest

See `docs/design.md` §12 for full stack rationale.

## Configuration

Config lives at one of (in priority order):

1. Path passed via `--config <path>`
2. `$WTMUX_CONFIG` environment variable
3. `.wtmux.json` found by walking upward from cwd to the filesystem root
4. `~/.config/wtmux/config.json`

Example config:

```json
{
  "symlinkDirectories": ["node_modules", ".env"],
  "worktreePathPattern": ".worktrees/{name}",
  "launchCommand": ["claude"],
  "groups": [
    {
      "name": "whatsapp-manager",
      "repos": [
        "~/whatsapp-manager/whatsapp-manager-api",
        "~/whatsapp-manager/whatsapp-manager-web"
      ]
    }
  ]
}
```

Running `wtmux <name>` from inside any repo in a group creates matching worktrees in every sibling repo on the same branch, replicates `symlinkDirectories` from each origin into its new worktree, and launches `claude` with each non-primary worktree wired up as an `--add-dir`.

## Flags

| Flag | Purpose |
|---|---|
| `--config <path>` | Override config discovery |
| `--group <name>` | Override auto-detected group |
| `--dry-run` | Print the plan without mutating |
| `--no-launch` | Skip `exec claude` at the end of `create` |
| `-v`, `--verbose` | Extra logging |

`wtmux rm <name>` refuses to remove a coordinated worktree if any side has uncommitted changes, stashes, or unpushed commits. Pass `--force` to override.

## License

MIT.
