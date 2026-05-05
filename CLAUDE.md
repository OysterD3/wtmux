# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`wtmux` is a Go CLI that creates matching git worktrees across configured sibling repos, replicates symlinks (e.g. `node_modules`, `.env`) into each new worktree, and execs the configured AI agent (`claude`, `codex`, `cursor`, etc.) with sibling repos attached via the agent's flag (`--add-dir`, `--add`, ...). It's a port from the previous TypeScript implementation; the legacy TS sources were removed in commit `d529c6c` — branches before that point still contain `src/` and `tests/`.

## Common commands

| Task | Command |
|---|---|
| Build local binary (`bin/wtmux`) | `make build` |
| Build with explicit version stamped via ldflags | `make build VERSION=v0.1.0` |
| Install to `~/go/bin/<name>` (default name `wtmux-go`) | `make install` |
| Side-install for dev so it doesn't clobber another `wtmux` | `make install INSTALL_NAME=wtmux-dev` |
| Run all tests | `make test` (or `go test ./...`) |
| Run with race detector | `make test-race` |
| Run a single package | `go test ./internal/git/...` |
| Run a single test | `go test ./internal/cli/... -run TestRmRefusesOnDirty` |
| Lint (matches CI config) | `make lint` (requires `golangci-lint` on PATH) |
| Vet | `make vet` |
| Cross-compile release artifacts | `make release VERSION=vX.Y.Z` (writes `dist/wtmux-vX.Y.Z-{darwin,linux}-{amd64,arm64}.tar.gz` + `checksums.txt`) |

The release pipeline (`.github/workflows/release.yml`) fires on `v*.*.*` tags; it runs `make release`, attaches artifacts to the GitHub release, and bumps the formula in `OysterD3/homebrew-wtmux` if the `HOMEBREW_TAP_TOKEN` repo secret is set.

CI (`.github/workflows/ci.yml`) runs `go vet`, `go test -race`, and `golangci-lint` against the version pinned in `go.mod` (currently 1.26).

## High-level architecture

Entry point is `cmd/wtmux/main.go` → `internal/cli.Execute()`. Cobra dispatches to four subcommands:

```
wtmux new <name> ──┐
wtmux ls ──────────┤
wtmux rm <name> ───┼──► loadConfigOrDefault → group.Resolve → ...
wtmux config ──────┘
```

The shared resolution layer:

1. **`internal/config`** — load + validate user config from `--config <path>` / `$WTMUX_CONFIG` / nearest `.wtmux.json` / `~/.config/wtmux/config.json`. `config.Default()` is the no-config fallback.
2. **`internal/group`** — `Resolve` returns one of `KindOutside` / `KindSingle` / `KindGroup` based on cwd's git toplevel and configured groups. Always returns a non-nil `*Resolution` on `err == nil`.
3. **`internal/preflight`** — `BuildPlan` merges per-group overrides onto top-level config, computes per-repo target worktree paths, and resolves the base branch. `Plan.Validate` runs the live filesystem + git-state checks. The split exists so `--dry-run` can inspect a Plan without filesystem mutation.
4. **`internal/glob`** — expands `symlinkDirectories` items per repo. Items without glob metacharacters pass through; items containing `*?[]{}` are matched against the repo tree (supports `**`).
5. **`internal/symlinks`** — `Replicate` creates per-item symlinks; `Remove` is the rollback primitive (anti-foot-gun: only deletes if target is itself a symlink, so a stale config entry can't destroy real user content). On error, `Replicate` returns `(nil, err)` — callers pass the **full** items slice to `Remove` for rollback (see `internal/cli/create.go`).
6. **`internal/git`** — thin wrapper around the system `git` CLI. **Read functions** that return data: non-zero git exit is now surfaced as an error (this is a load-bearing invariant — the previous "swallow non-zero exits" behavior caused silent rm/ls bugs and is being actively unwound). **Write functions** wrap stderr into the error message. Tests require `git` on PATH (`TestMain` hard-fails otherwise — no mocks).
7. **`internal/launch`** — `BuildArgv` renders the agent's add-dir template (e.g. Claude's `--add-dir <path>` per sibling); `Exec` calls `os/exec` with stdio inherit and `os.Exit(code)` on completion. `ExecFunc` is swappable for tests.
8. **`internal/agents`** — registry of known agents (`claude`, `codex`, `cursor`, `code`, `opencode`, `qoder`) and `ResolveStrategy` decides between flag-template injection (`StrategyFlagAddDir`), positional siblings, or `StrategyNone` for agents without multi-root support.
9. **`internal/tui`** — `huh`-based interactive editor used by `wtmux config`. Replaced TS's `@clack/prompts`.

`internal/paths` is the small pure-helper package (tilde expansion, `RealpathSafe`, worktree-name flattening, `{name}` pattern interpolation). `internal/log` is a stderr-only logger with `[wtmux] ` / `[wtmux:debug] ` prefixes.

## Conventions worth knowing

**Exit codes are a contract.** `internal/errors` defines `KindUser` (1), `KindPrecondition` (2), `KindInternal` (3). Any `RunE` return that should map to one of those *must* go through `wtmuxerrors.New` or `wtmuxerrors.Wrapf` — a plain `error` falls through to `ExitCodeFor`'s default of 3. Cobra's positional-arg validators are wrapped via `cli.userArgs(...)`; flag-parsing failures are caught by `SetFlagErrorFunc` on the root. When you wrap a sentinel that callers may want to inspect, use `Wrapf` (preserves `errors.Is` traversal); use `New` only when there's no underlying cause.

**Path comparisons need realpath on one side.** `git worktree list` always emits symlink-resolved paths; config-supplied repo paths may not be. Use `paths.RealpathSafe(repo)` before comparing against `WorktreeEntry.Worktree`. The same trap exists in any new code that joins config paths with git output.

**rm is all-or-nothing across a group.** `runRm` runs in three passes (discover → precheck → mutate). Without `--force`, a single dirty/unpushed worktree aborts the entire operation with `KindPrecondition`. Don't refactor it back into a single-pass loop — the partial-removal failure mode is what the README's "refuses to remove" promise rules out.

**Create rollback is order-sensitive.** Append the `placed{}` record to `placedOK` *immediately* after `WorktreeAdd*` succeeds, before invoking `symlinks.Replicate`. A Replicate failure must roll back the freshly-created worktree, which means the entry has to be in the rollback list at that point.

**Version is one constant.** `internal/cli/version.go` defines `defaultVersion = "0.0.1"` and `var version = defaultVersion`. Release builds override `version` via `-ldflags=-X .../cli.version=...`; `Version()` detects the override by comparing against `defaultVersion`. Bumping the in-source default is a one-line change — the comparison reads the same constant so the VCS-suffix fallback path stays correct.

**TUI strings face users.** Validation errors from `internal/tui/validate.go` render directly in `huh.Input` form fields. Lowercased to satisfy `staticcheck ST1005`; if you need user-friendly capitalization in a *display* string (not an `error` return), it's fine.

**Integration tests build the binary.** `internal/cli/integration_test.go`'s `TestMain` runs `go build` into a temp dir, then each test invokes the binary via `os/exec` against a `t.TempDir()` git repo. Keep this pattern for tests that exercise exit codes, stdout/stderr formatting, or end-to-end command flow — the cobra layer doesn't compose well in pure unit tests.

## Useful files when starting

- `internal/cli/root.go` — cobra wiring, where new top-level flags or subcommands hang
- `internal/cli/create.go` / `rm.go` / `ls.go` — the three command flows; mirror each other in shape (resolve → discover → precheck → mutate)
- `internal/preflight/plan.go` — the `Plan` struct is what flows from config to mutation; understand it before changing anything in `create.go`
- `internal/git/git.go` — every git operation in the codebase routes through here; new git operations should follow the existing `run()` + exit-code-check pattern
- `Makefile` — single source of truth for build flags; CI calls `make` targets verbatim, not raw `go build`
