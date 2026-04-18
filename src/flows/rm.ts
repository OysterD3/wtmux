import fs from "node:fs/promises";
import path from "node:path";
import { WtmuxError } from "../errors.js";
import { info, warn } from "../log.js";
import { resolveGroup } from "../group.js";
import {
  branchIsMergedInto,
  deleteBranch,
  getCurrentBranch,
  stashList,
  statusPorcelain,
  unpushedCommits,
  worktreePrune,
  worktreeRemove,
} from "../git.js";
import type { Config } from "../config/schema.js";

export interface RmFlowInput {
  name: string;
  cwd: string;
  config: Config;
  groupFlag: string | undefined;
  dryRun: boolean;
  force: boolean;
}

export interface RmFlowResult {
  removed: { repo: string; wtPath: string }[];
  skipped: { repo: string; wtPath: string; reason: string }[];
  absent: { repo: string; wtPath: string }[];
}

export async function rmFlow(input: RmFlowInput): Promise<RmFlowResult> {
  const resolved = await resolveGroup({
    cwd: input.cwd,
    config: input.config,
    groupFlag: input.groupFlag,
  });
  if (resolved.kind === "outside") {
    throw new WtmuxError("cwd is not inside any git repository", "precondition");
  }

  const repos = resolved.kind === "group" ? resolved.group.repos : [resolved.repo];
  const pattern =
    (resolved.kind === "group" && resolved.group.worktreePathPattern) ||
    input.config.worktreePathPattern;

  const result: RmFlowResult = { removed: [], skipped: [], absent: [] };

  for (const repo of repos) {
    const wtPath = expandWorktreePath(repo, pattern, input.name);
    if (!(await pathExists(wtPath))) {
      result.absent.push({ repo, wtPath });
      continue;
    }

    if (input.dryRun) {
      info(`dry-run: would remove ${wtPath} in ${repo}`);
      continue;
    }

    if (!input.force) {
      const status = await statusPorcelain(wtPath);
      if (status !== "") {
        warn(`skipping ${repo}/${input.name} — uncommitted changes`);
        result.skipped.push({ repo, wtPath, reason: "dirty" });
        continue;
      }
      const stashes = await stashList(wtPath);
      if (stashes.length > 0) {
        warn(`skipping ${repo}/${input.name} — ${stashes.length} stash(es) present`);
        result.skipped.push({ repo, wtPath, reason: "stashed" });
        continue;
      }
      const unpushed = await unpushedCommits(wtPath);
      if (unpushed.length > 0) {
        warn(`skipping ${repo}/${input.name} — ${unpushed.length} unpushed commit(s)`);
        result.skipped.push({ repo, wtPath, reason: "unpushed" });
        continue;
      }
    }

    await worktreeRemove(repo, wtPath, { force: input.force });
    result.removed.push({ repo, wtPath });

    const base = (await getCurrentBranch(repo)) ?? "HEAD";
    try {
      if (await branchIsMergedInto(repo, input.name, base)) {
        await deleteBranch(repo, input.name);
      } else {
        info(
          `branch "${input.name}" in ${repo} is not fully merged into ${base} — leaving it. Delete manually with: git -C ${repo} branch -D ${input.name}`,
        );
      }
    } catch {
      // branch deletion failures are non-fatal
    }

    await worktreePrune(repo);
  }

  return result;
}

async function pathExists(p: string): Promise<boolean> {
  try {
    await fs.lstat(p);
    return true;
  } catch {
    return false;
  }
}

function expandWorktreePath(repo: string, pattern: string, name: string): string {
  const rendered = pattern.replaceAll("{name}", name);
  return path.isAbsolute(rendered) ? rendered : path.join(repo, rendered);
}
