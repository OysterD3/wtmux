import { WtmuxError } from "../errors.js";
import { expandWorktreePath } from "../paths.js";
import { debug, info, warn } from "../log.js";
import { resolveGroup, type GroupResolution } from "../group.js";
import { preflightCreate } from "../preflight.js";
import {
  branchExists,
  getCurrentBranch,
  worktreeAdd,
  worktreeRemove,
} from "../git.js";
import { removeSymlinks, replicateSymlinks } from "../symlinks.js";
import { buildLaunchArgv, execLaunch } from "../launch.js";
import type { Config } from "../config/schema.js";

export interface CreateFlowInput {
  name: string;
  cwd: string;
  config: Config;
  groupFlag: string | undefined;
  dryRun: boolean;
  noLaunch: boolean;
  extraArgs: readonly string[];
  baseOverride: string | undefined;
}

export type CreateFlowResult =
  | { kind: "group"; worktrees: { repo: string; wtPath: string }[]; primary: string }
  | { kind: "single"; repo: string; wtPath: string };

export async function createFlow(input: CreateFlowInput): Promise<CreateFlowResult> {
  const resolved = await resolveGroup({
    cwd: input.cwd,
    config: input.config,
    groupFlag: input.groupFlag,
  });

  if (resolved.kind === "outside") {
    throw new WtmuxError(
      "cwd is not inside any git repository — run wtmux from within a repo, or pass --group",
      "precondition",
    );
  }

  if (resolved.kind === "single") {
    return createSingleRepo(resolved.repo, input);
  }

  return createGroup(resolved, input);
}

async function createSingleRepo(repo: string, input: CreateFlowInput): Promise<CreateFlowResult> {
  const base = input.baseOverride ?? (await getCurrentBranch(repo));
  if (!base) {
    throw new WtmuxError(
      `${repo}: detached HEAD — check out a branch before creating a worktree, or pass --base`,
      "precondition",
    );
  }

  const wtPath = expandWorktreePath(repo, input.config.worktreePathPattern, input.name);
  const plan = { name: input.name, baseBranch: base, repos: [{ path: repo, wtPath }] };

  if (input.dryRun) {
    info(`dry-run: would create ${wtPath} from ${base} (${input.name})`);
    return { kind: "single", repo, wtPath };
  }

  const pre = await preflightCreate(plan);
  if (!pre.ok) throw new WtmuxError(pre.errors.join("\n"), "user");

  if (input.baseOverride) {
    const exists = await branchExists(repo, input.name);
    if (exists) {
      warn(`--base "${input.baseOverride}" ignored: branch "${input.name}" already exists`);
    }
  }

  const exists = await branchExists(repo, input.name);
  await worktreeAdd(repo, {
    path: wtPath,
    branch: input.name,
    base,
    createBranch: !exists,
  });

  const items = input.config.symlinkDirectories;
  await replicateSymlinks({ repo, wt: wtPath, items });

  if (!input.noLaunch) {
    const argv = buildLaunchArgv({
      launchCommand: input.config.launchCommand,
      siblingWorktrees: [],
      extraArgs: input.extraArgs,
    });
    execLaunch({ argv, cwd: wtPath });
  }

  return { kind: "single", repo, wtPath };
}

async function createGroup(
  resolved: Extract<GroupResolution, { kind: "group" }>,
  input: CreateFlowInput,
): Promise<CreateFlowResult> {
  const primary = resolved.primary;
  const base = input.baseOverride ?? (await getCurrentBranch(primary));
  if (!base) {
    throw new WtmuxError(
      `${primary}: detached HEAD — check out a branch before creating a worktree, or pass --base`,
      "precondition",
    );
  }

  const items = resolved.group.symlinkDirectories ?? input.config.symlinkDirectories;
  const pattern = resolved.group.worktreePathPattern ?? input.config.worktreePathPattern;
  const launchCommand = resolved.group.launchCommand ?? input.config.launchCommand;

  const plan = {
    name: input.name,
    baseBranch: base,
    repos: resolved.group.repos.map((r) => ({
      path: r,
      wtPath: expandWorktreePath(r, pattern, input.name),
    })),
  };

  if (input.dryRun) {
    info(`dry-run: would create worktrees for "${input.name}" from "${base}":`);
    for (const { path: repo, wtPath } of plan.repos) info(`  ${repo} -> ${wtPath}`);
    info(`dry-run: symlink items: ${items.join(", ")}`);
    return {
      kind: "group",
      worktrees: plan.repos.map(({ path: repo, wtPath }) => ({ repo, wtPath })),
      primary,
    };
  }

  const pre = await preflightCreate(plan);
  if (!pre.ok) throw new WtmuxError(pre.errors.join("\n"), "user");

  if (input.baseOverride) {
    const existing = await Promise.all(
      plan.repos.map((r) => branchExists(r.path, input.name)),
    );
    if (existing.some((e) => e)) {
      warn(`--base "${input.baseOverride}" ignored: branch "${input.name}" already exists`);
    }
  }

  const placed: { repo: string; wtPath: string; items: string[] }[] = [];
  try {
    for (const { path: repo, wtPath } of plan.repos) {
      const exists = await branchExists(repo, input.name);
      await worktreeAdd(repo, {
        path: wtPath,
        branch: input.name,
        base,
        createBranch: !exists,
      });
      const results = await replicateSymlinks({ repo, wt: wtPath, items });
      placed.push({
        repo,
        wtPath,
        items: results.filter((r) => r.action === "linked").map((r) => r.item),
      });
    }
  } catch (err) {
    debug(`rollback triggered: ${(err as Error).message}`);
    for (const p of [...placed].reverse()) {
      await removeSymlinks(p.wtPath, p.items).catch(() => undefined);
      await worktreeRemove(p.repo, p.wtPath, { force: true }).catch(() => undefined);
    }
    throw err;
  }

  if (!input.noLaunch) {
    const siblings = plan.repos
      .filter(({ path: r }) => r !== primary)
      .map((r) => r.wtPath);
    const primaryWt = plan.repos.find((r) => r.path === primary)!.wtPath;
    const argv = buildLaunchArgv({
      launchCommand,
      siblingWorktrees: siblings,
      extraArgs: input.extraArgs,
    });
    execLaunch({ argv, cwd: primaryWt });
  }

  return { kind: "group", worktrees: plan.repos.map(({ path: repo, wtPath }) => ({ repo, wtPath })), primary };
}

