import { WtmuxError } from "../errors.js";
import { resolveGroup } from "../group.js";
import { listWorktrees, stashList, statusPorcelain, unpushedCommits } from "../git.js";
import type { Config } from "../config/schema.js";

export interface LsFlowInput {
  cwd: string;
  config: Config;
  groupFlag: string | undefined;
}

export interface LsRepoStatus {
  repo: string;
  present: boolean;
  state: "clean" | "dirty" | "stashed" | "unpushed" | null;
  wtPath: string | null;
}

export interface LsRow {
  name: string;
  perRepo: LsRepoStatus[];
}

export async function lsFlow(input: LsFlowInput): Promise<LsRow[]> {
  const resolved = await resolveGroup({
    cwd: input.cwd,
    config: input.config,
    groupFlag: input.groupFlag,
  });
  if (resolved.kind === "outside") {
    throw new WtmuxError("cwd is not inside any git repository", "precondition");
  }

  const repos = resolved.kind === "group" ? resolved.group.repos : [resolved.repo];
  const perRepoWorktrees = await Promise.all(
    repos.map(async (r) => ({ repo: r, wts: await listWorktrees(r) })),
  );

  const names = new Set<string>();
  for (const { wts } of perRepoWorktrees) {
    for (const w of wts) {
      if (w.branch && w.branch !== "main" && w.branch !== "master") names.add(w.branch);
    }
  }

  const rows: LsRow[] = [];
  for (const name of [...names].sort()) {
    const perRepo: LsRepoStatus[] = [];
    for (const { repo, wts } of perRepoWorktrees) {
      const match = wts.find((w) => w.branch === name && w.worktree !== repo);
      if (!match) {
        perRepo.push({ repo, present: false, state: null, wtPath: null });
        continue;
      }
      perRepo.push({
        repo,
        present: true,
        wtPath: match.worktree,
        state: await classify(match.worktree),
      });
    }
    rows.push({ name, perRepo });
  }
  return rows;
}

async function classify(wtPath: string): Promise<LsRepoStatus["state"]> {
  if ((await statusPorcelain(wtPath)) !== "") return "dirty";
  if ((await stashList(wtPath)).length > 0) return "stashed";
  if ((await unpushedCommits(wtPath)).length > 0) return "unpushed";
  return "clean";
}
