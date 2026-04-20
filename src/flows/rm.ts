import { WtmuxError } from "../errors.js";
import { expandWorktreePath, pathExists } from "../paths.js";
import { info, warn } from "../log.js";
import { resolveGroup } from "../group.js";
import {
  deleteBranch,
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
        warn(`skipping ${wtPath} — uncommitted changes`);
        result.skipped.push({ repo, wtPath, reason: "dirty" });
        continue;
      }
      const unpushed = await unpushedCommits(wtPath);
      if (unpushed.length > 0) {
        warn(`skipping ${wtPath} — ${unpushed.length} unpushed commit(s)`);
        result.skipped.push({ repo, wtPath, reason: "unpushed" });
        continue;
      }
    }

    await worktreeRemove(repo, wtPath, { force: input.force });
    result.removed.push({ repo, wtPath });

    try {
      await deleteBranch(repo, input.name, { force: input.force });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      info(`could not delete branch "${input.name}" in ${repo}: ${message}`);
    }

    await worktreePrune(repo);
  }

  return result;
}

