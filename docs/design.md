# wtmux — Design Spec

**Status:** Draft, 2026-04-18
**Name:** `wtmux` (worktree multiplexer)
**License:** MIT
**Language:** TypeScript (Node 20+)
**Package manager:** pnpm

---

## 1. Problem

Claude Code's native `/worktree` feature creates a git worktree for the primary working directory only. When a user has added a second independent git repo via `--add-dir` (e.g. a sibling service in a monorepo), that additional directory continues to point at the original repo path — not at a matching worktree. The user ends up working on an isolated branch in the API repo while the web repo is still on whatever branch it was before.

Additionally, Claude Code does **not** expose any mechanism to rewrite additional-directory paths mid-session: the `WorktreeCreate` hook can only return `worktreePath` for the primary, no `SessionStart`/`UserPromptSubmit` hook can register new directories, and `--add-dir` paths are resolved once at startup.

**Therefore:** a purely hook-based solution cannot achieve "coordinated worktrees that Claude sees as add-dirs in the same session". A wrapper that controls the `claude` invocation is required.

## 2. Goals

1. Given a worktree name, create matching git worktrees across a configured **group** of sibling repos, on the same branch, branched from the same base.
2. Replicate Claude's `worktree.symlinkDirectories` behavior (symlink `node_modules`, `.env`, etc. from origin repo into each worktree) so dev servers/env work immediately.
3. Launch `claude` in the primary worktree with sibling worktrees passed as `--add-dir`, so the in-session view of additional directories already points at the correct worktrees.
4. Safely tear down coordinated worktrees (refuse to remove if any side has uncommitted, stashed, or unpushed work, unless forced).
5. Be reusable across any number of projects via a config file (user-level default + project-local override).
6. Work as a single distributable npm package; no Claude dependency beyond shelling out to `claude` at the end.
7. Graceful single-repo fallback when cwd isn't in any configured group (behaves like `git worktree add` + symlinks + `exec claude`).

## 3. Non-goals

- Replacing Claude's native worktree command for non-coordinated use cases. `wtmux` is opt-in; the `/worktree` flow continues to work.
- Managing non-git worktrees, submodules, or non-sibling repo layouts.
- Rewriting Claude's `--add-dir` resolution behavior. That's a platform limitation we route around by controlling the launch.
- Publishing to Homebrew tap (can be added later; npm is the initial distribution).
- Interactive branch selection / merge-conflict handling. That's git's job.

## 4. CLI surface

```
wtmux <name> [-- claude-args...]      # Create coordinated worktrees + launch claude
wtmux rm <name> [--force]              # Remove coordinated worktrees (clean-only unless --force)
wtmux ls                               # List worktrees in cwd's group
wtmux doctor                           # Diagnose config, PATH, git versions, symlink sources
wtmux init                             # Interactive first-run: write a starter config file
```

Global flags:
- `--config <path>` — override config discovery
- `--group <name>` — override auto-detected group (useful from arbitrary cwd)
- `--dry-run` — print planned actions, make no changes
- `--no-launch` — skip `exec claude` at the end of `create`
- `--verbose` / `-v`
- `--version` / `-V`, `--help` / `-h`

Argv parsing: **citty** (modern, TypeScript-first, pairs cleanly with clack). Interactive prompts: **@clack/prompts**.

## 5. Config

### Discovery order

1. `--config <path>` flag
2. `$WTMUX_CONFIG` env var
3. `.wtmux.json` walking upward from cwd until a repo boundary (first match wins — lets a monorepo commit a shared config)
4. `~/.config/wtmux/config.json` (XDG user default)

Missing config → `wtmux init` suggests running the wizard. `create` in single-repo mode can run without any config at all.

### Schema

Validate with **zod**. Example:

```json
{
  "$schema": "https://unpkg.com/wtmux/schema.json",
  "symlinkDirectories": ["node_modules", ".env"],
  "worktreePathPattern": ".worktrees/{name}",
  "launchCommand": ["claude"],
  "groups": [
    {
      "name": "whatsapp-manager",
      "repos": [
        "/Users/oysterlee/whatsapp-manager/whatsapp-manager-api",
        "/Users/oysterlee/whatsapp-manager/whatsapp-manager-web"
      ],
      "symlinkDirectories": ["node_modules", ".env", ".env.local"]
    }
  ]
}
```

- `symlinkDirectories` (top level): default list of paths to symlink from each repo root into each new worktree. Group-level override replaces (not merges) the top-level list.
- `worktreePathPattern`: controls where worktrees land inside each repo. `{name}` interpolates the worktree name. Default `.worktrees/{name}` (matches Claude's convention). Group-level override allowed.
- `launchCommand`: argv to exec after worktrees are created. Defaults to `["claude"]`. Can be overridden for non-Claude use (e.g. `["code", "."]` or `["echo"]` for testing). `wtmux` appends `--add-dir <sibling-wt>` flags automatically **only when `launchCommand[0] === "claude"`**; otherwise sibling paths are appended as bare positional args (document this).
- `groups[].repos`: absolute paths (tilde expanded). Order only matters as a stable display order; primary is detected from cwd, not config order.

### Paths

- Home dir of tildes expanded on load.
- All repo paths in config must be absolute after expansion; validator rejects relative paths (they'd be cwd-dependent, which defeats the "run from anywhere" goal).
- Each repo must resolve to a git toplevel (checked lazily on commands that need it).

## 6. Create flow — `wtmux <name>`

1. **Resolve config** and determine group:
   - `--group` flag wins.
   - Else find git toplevel of cwd. Find the group whose `repos` array contains that toplevel. Exactly one match required; zero → single-repo mode; >1 → error and ask for `--group`.
2. **Determine primary repo** = git toplevel of cwd if it's in the group; else first repo in group.
3. **Determine base branch** = `git -C $primary rev-parse --abbrev-ref HEAD`. Reject if detached HEAD — require the user to check out a branch first so the intent is explicit.
4. **Preflight checks** (all must pass before any mutation):
   a. Worktree name is a valid git ref: shell out to `git check-ref-format --branch "$name"` and reject on non-zero. Authoritative validator; zero bugs for us to maintain.
   b. Each repo exists and is a git worktree root (not a worktree itself).
   c. For each repo, either:
      - Branch `$name` already exists and is not currently checked out anywhere, OR
      - Branch `$name` does not exist AND the base branch `$base_branch` exists in this repo.
   d. For each repo, target worktree path `$repo/$worktree_path_pattern` does not exist.
5. **Create worktrees** sequentially (not parallel; errors are easier to roll back one at a time):
   - Branch exists → `git -C $repo worktree add $wt_path $name`
   - Else → `git -C $repo worktree add -b $name $wt_path $base_branch`
6. **Symlink replication**: for each repo, for each entry in effective `symlinkDirectories`:
   - Source = `$repo/$item`. Skip if source doesn't exist.
   - Target = `$wt_path/$item`. Skip if target already exists (could happen if the worktree initialization created it).
   - `fs.symlink(source, target, "dir"|"file")` — detect type from source.
7. **Launch**:
   - If `--no-launch`, print summary and exit 0.
   - Else build argv: `[...launchCommand]`. If `launchCommand[0] === "claude"`, append `--add-dir $sibling_wt_path` for each non-primary worktree. Else append sibling paths positionally.
   - `cd` to primary worktree.
   - `exec` (replace process) — not spawn. This lets Claude inherit the TTY and gives the wrapper zero ongoing overhead.

### Rollback

If step 5 or step 6 fails partway through, roll back every already-created worktree via `git worktree remove --force` and unlink any symlinks we placed, before exiting with non-zero. The goal is all-or-nothing.

### Idempotency

`wtmux <name>` is not idempotent by design — if any target already exists, preflight fails loudly. Users who want to re-enter an existing coordinated session can use `wtmux enter <name>` (future work).

## 7. Remove flow — `wtmux rm <name>`

1. Resolve config and group (same logic as create; `--group` allowed).
2. For each repo in group:
   - Resolve `$wt_path`. If it doesn't exist, skip.
   - Open the worktree's git state:
     - `git -C $wt_path status --porcelain` must be empty.
     - `git -C $wt_path stash list` must be empty (stashes are easy to lose silently when the worktree is removed).
     - If the branch has an upstream, `git -C $wt_path log @{u}..HEAD --oneline` must be empty. No upstream → treat as clean (nothing to push against).
   - Clean → `git -C $repo worktree remove $wt_path`. If branch is fully merged into `base_branch`, `git -C $repo branch -d $name`. If not merged, leave the branch (safer default).
   - Dirty → print `[wtmux] skipping $repo/$name — dirty or unpushed` to stderr, do not remove.
3. `git -C $repo worktree prune` per repo to clean stale metadata.
4. Exit 0 if every repo either removed cleanly or skipped safely. Exit non-zero only if an unexpected error occurred (filesystem failure, config issue).

### `--force`

Replaces both the dirty check and the stash check with `git worktree remove --force`. Still does not delete unmerged branches — prints a note telling the user how to delete manually. Dangerous enough to require the flag; not dangerous enough to require a second confirmation.

## 8. List flow — `wtmux ls`

Walk the group's repos, union the worktree names across them (via `git worktree list --porcelain` parsing), and print:

```
feat/foo            api: feat/foo  dirty     web: feat/foo  clean
feat/orphaned       api: —                   web: feat/orphaned  clean
```

Uses clack's `note` / formatted output for alignment. Columns dynamic to group size.

## 9. Symlink replication semantics

Claude's `worktree.symlinkDirectories` setting creates symlinks **from the new worktree pointing at the origin repo's corresponding path**. So:

```
<repo>/.worktrees/feat-foo/node_modules → <repo>/node_modules
```

This is what `wtmux` replicates. Specifically:

- Source resolution: `path.join(repoRoot, item)`. No glob expansion; plain paths only.
- Type detection: `fs.lstat` on source — dir vs file. If source is itself a symlink, we resolve through it (symlinks pointing at a shared `.env` still work).
- Atomicity: no rollback of already-placed symlinks on failure — they're cheap and harmless.
- Conflict: if target already exists as a regular file (rare; git shouldn't check one in at worktree root level), skip with a warning rather than overwrite.

## 10. `wtmux doctor`

Prints:
- `wtmux` version
- Node version
- `git --version`
- Resolved config path + parsed config (redacted for secrets — there aren't any, but be defensive)
- For each group: each repo's existence, git toplevel check, default branch detection
- Whether `claude` is on `PATH`
- Any symlink source paths that don't exist in any repo

Exit non-zero if anything is broken. Useful in CI of a monorepo that wants to assert config stays valid.

## 11. Packaging & distribution

- Output: single ESM bundle via **tsup** (`bin/wtmux.js`), shebang `#!/usr/bin/env node`.
- `package.json` `bin` field: `{ "wtmux": "./dist/wtmux.js" }`.
- Published to npm as `wtmux` (verify availability before publish).
- No install-time compilation: tsup builds in CI. Users get a pre-built tarball.
- Future: a Homebrew formula that wraps the npm install, so `brew install wtmux` works for users without Node. Out of scope for v0.1.
- Semantic versioning via **changesets**. `main` branch → release PR → npm publish.

## 12. Tech stack

| Concern | Choice | Why |
|---|---|---|
| Language | TypeScript (strict) | User's stack; ecosystem |
| Runtime | Node 20+ | ESM, native `fs.cp`, modern stdlib |
| Argv parsing | **citty** | TypeScript-first, composable subcommands |
| Interactive prompts | **@clack/prompts** | User pick; modern TUI |
| Subprocess | **execa** | Mature, good stream handling, stderr propagation |
| Config validation | **zod** | Already in the user's main project |
| Bundling | **tsup** | Fastest path to a single-file bin |
| Testing | **vitest** | ESM-native, fast, TS-first |
| Lint + format | **eslint + prettier** | Matches user's existing TypeScript projects |
| Release | **changesets** | Solo maintainer friendly |
| CI | GitHub Actions | Lint, typecheck, test, build matrix |

Stack decisions confirmed in the 2026-04-18 brainstorm: argv = citty, lint = eslint + prettier, prompts = @clack/prompts, license = MIT.

## 13. Testing strategy

- **Unit**: config parsing (zod schema), path resolution, group detection, preflight checks — all pure functions with deterministic inputs.
- **Integration**: spawn temp directories via `fs.mkdtemp`, `git init` two fake repos with matching branches, run the `wtmux create` flow end-to-end with `--no-launch`, assert filesystem state (worktrees exist, symlinks present, correct branches).
- **Snapshot**: CLI help text, `doctor` output structure.
- **--dry-run coverage**: every mutating command must pass with `--dry-run` in a test to guarantee the dry-run output stays accurate.
- Coverage target: 90% for pure logic modules, 70% overall (integration tests are slower and selectively run).

## 14. Error handling philosophy

- Every user-visible error ends in a single actionable line: what went wrong + how to fix. No stack traces in normal mode (`--verbose` enables them).
- Exit codes:
  - `0` success
  - `1` user error (bad config, missing branch, dirty worktree in `rm`)
  - `2` precondition failure (cwd not in group, no config, detached HEAD)
  - `3` unexpected internal error (bug)
- Preflight failures name the specific repo + the specific check that failed.

## 15. Open questions

All items previously listed here were resolved in the 2026-04-18 brainstorm:

1. **`init` wizard scope** — deferred to v0.2 (§16). MVP scope unchanged: detect cwd is a git repo, prompt for sibling repo paths, write a starter config.
2. **Per-repo branch mapping (`baseBranchMap`)** — deferred to v0.3+. v0.1 uses a single base branch taken from the primary's HEAD.
3. **`addDirsFlag` for non-Claude editors** — deferred to v0.3+. v0.1 keeps Claude-specific `--add-dir` injection; `launchCommand` still works for other tools but appends sibling paths positionally.
4. **Worktree name validation** — resolved. Shell out to `git check-ref-format --branch "$name"` in preflight (§6 step 4a).
5. **Concurrency file lock** — deferred to v0.2. Preflight will still catch the common case (duplicate target paths).

## 16. Milestones

**v0.1 (MVP):**
- `create`, `rm`, `ls` commands
- `--force` on `rm` (skips dirty + stash checks)
- Config schema + discovery
- Symlink replication
- Single-repo fallback
- Tests for core flows
- README + quickstart

**v0.2:**
- `doctor`, `init` wizard
- File locking
- Homebrew formula

**v0.3+:**
- Per-repo branch mapping
- Hooks for pre/post-create
- Shell completion (bash/zsh/fish)

## 17. Appendix: why not hooks?

The `WorktreeCreate` and `WorktreeRemove` hooks documented at https://code.claude.com/docs/en/hooks.md can create sibling worktrees, but they cannot rewrite the current Claude session's `--add-dir` paths. There is no hook output field, settings entry, or runtime API for this. Confirmed via docs review — see `docs/research/hook-limitations.md` (to be added) if we want to preserve the reasoning.

The upstream feature request worth filing:
- `WorktreeCreate` hook returns `hookSpecificOutput.additionalDirectories` that Claude honors for the session.
- Or: a `worktree.additionalDirectoryGroups` setting that Claude itself understands.

If/when either ships, `wtmux` can become thinner — potentially just a config + a hook script — but the current scope still makes sense as a standalone tool because the symlink replication + launch logic is useful regardless.

---

## Handoff note for the next Claude session

When you open this project in a new session:

1. Read this file (`docs/design.md`) end-to-end.
2. Invoke the `superpowers:writing-plans` skill to turn this spec into an implementation plan.
3. Bootstrap the project: `git init`, `pnpm init`, TS config, tsup, vitest, eslint + prettier, citty, @clack/prompts, execa, zod. License: MIT.
4. Build v0.1 milestone (§16) first. Commit after each subsection of the plan.

The user's environment: macOS, pnpm, Node 20+, zsh. No Node global flags needed. User will run `pnpm install` themselves; don't assume a lockfile exists.
