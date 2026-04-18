# wtmux

Coordinated git worktrees across sibling repos, with auto-replicated symlinks (`node_modules`, `.env`) and a one-shot launch into `claude` (or your editor of choice) with all sibling worktrees wired up as additional directories.

**Status:** design only — not yet implemented. See [`docs/design.md`](docs/design.md) for the full spec.

## The problem

Claude Code's `/worktree` command creates a worktree for the primary working directory only. If you have added a sibling repo via `--add-dir` (e.g. a frontend alongside a backend), that directory keeps pointing at the original path — on whatever branch it happened to be on. You end up working on an isolated feature branch in one repo while the other is stale.

Claude Code has no mechanism to rewrite `--add-dir` paths mid-session, so a pure hook-based fix isn't possible. `wtmux` replaces the worktree-creation step with a wrapper that:

1. Creates matching worktrees in every configured sibling repo (same branch name, same base branch).
2. Symlinks `node_modules` / `.env` / etc. into each worktree — replicating Claude's `worktree.symlinkDirectories` behavior.
3. Launches `claude` with sibling worktrees passed as `--add-dir`, so the in-session view of additional directories already points at the correct worktrees.

## Planned usage

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

## License

MIT.
