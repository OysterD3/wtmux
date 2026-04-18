import {
  branchExists,
  checkRefFormat,
  hasRef,
  isWorktreeRoot,
  listWorktrees,
} from "./git.js";
import { pathExists } from "./paths.js";

export interface PreflightRepoInput {
  path: string;
  wtPath: string;
}

export interface PreflightInput {
  name: string;
  baseBranch: string;
  repos: readonly PreflightRepoInput[];
}

export interface PreflightResult {
  ok: boolean;
  errors: string[];
}

export async function preflightCreate(input: PreflightInput): Promise<PreflightResult> {
  const errors: string[] = [];

  if (!(await checkRefFormat(input.name))) {
    errors.push(`invalid worktree name "${input.name}" (git check-ref-format --branch rejected it)`);
    return { ok: false, errors };
  }

  for (const { path: repo, wtPath } of input.repos) {
    if (!(await isWorktreeRoot(repo))) {
      errors.push(`${repo}: not a worktree root (is this a linked worktree?)`);
      continue;
    }

    if (await pathExists(wtPath)) {
      errors.push(`${repo}: target worktree path already exists: ${wtPath}`);
    }

    const exists = await branchExists(repo, input.name);
    if (exists) {
      const wts = await listWorktrees(repo);
      if (wts.some((w) => w.branch === input.name)) {
        errors.push(`${repo}: branch "${input.name}" is already checked out in another worktree`);
      }
    } else {
      if (!(await hasRef(repo, input.baseBranch))) {
        errors.push(`${repo}: base branch "${input.baseBranch}" does not exist`);
      }
    }
  }

  return { ok: errors.length === 0, errors };
}

