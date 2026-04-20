import { execa } from "execa";

async function run(cwd: string, args: string[]): Promise<{ stdout: string; exitCode: number }> {
  const result = await execa("git", args, { cwd, reject: false });
  return { stdout: result.stdout, exitCode: result.exitCode ?? 1 };
}

export async function getToplevel(cwd: string): Promise<string | null> {
  const { stdout, exitCode } = await run(cwd, ["rev-parse", "--show-toplevel"]);
  if (exitCode !== 0) return null;
  return stdout.trim();
}

export async function isWorktreeRoot(repoPath: string): Promise<boolean> {
  const [common, gitDir] = await Promise.all([
    run(repoPath, ["rev-parse", "--git-common-dir"]),
    run(repoPath, ["rev-parse", "--git-dir"]),
  ]);
  if (common.exitCode !== 0 || gitDir.exitCode !== 0) return false;
  return common.stdout.trim() === gitDir.stdout.trim();
}

export async function getCurrentBranch(repoPath: string): Promise<string | null> {
  const { stdout, exitCode } = await run(repoPath, ["symbolic-ref", "--short", "-q", "HEAD"]);
  if (exitCode !== 0) return null;
  return stdout.trim() || null;
}

export async function checkRefFormat(name: string): Promise<boolean> {
  const result = await execa("git", ["check-ref-format", "--branch", name], { reject: false });
  return (result.exitCode ?? 1) === 0;
}

export async function branchExists(repoPath: string, name: string): Promise<boolean> {
  const { exitCode } = await run(repoPath, [
    "show-ref",
    "--verify",
    "--quiet",
    `refs/heads/${name}`,
  ]);
  return exitCode === 0;
}

export async function hasRef(repoPath: string, ref: string): Promise<boolean> {
  const { exitCode } = await run(repoPath, ["rev-parse", "--verify", "--quiet", ref]);
  return exitCode === 0;
}

export interface WorktreeEntry {
  worktree: string;
  head: string;
  branch: string | null;
  detached: boolean;
}

export async function listWorktrees(repoPath: string): Promise<WorktreeEntry[]> {
  const { stdout, exitCode } = await run(repoPath, ["worktree", "list", "--porcelain"]);
  if (exitCode !== 0) return [];
  const entries: WorktreeEntry[] = [];
  let cur: Partial<WorktreeEntry> | null = null;
  for (const line of stdout.split("\n")) {
    if (line.startsWith("worktree ")) {
      if (cur) entries.push(finalize(cur));
      cur = { worktree: line.slice("worktree ".length) };
    } else if (line.startsWith("HEAD ") && cur) {
      cur.head = line.slice("HEAD ".length);
    } else if (line.startsWith("branch refs/heads/") && cur) {
      cur.branch = line.slice("branch refs/heads/".length);
    } else if (line === "detached" && cur) {
      cur.detached = true;
    } else if (line === "" && cur) {
      entries.push(finalize(cur));
      cur = null;
    }
  }
  if (cur) entries.push(finalize(cur));
  return entries;
}

function finalize(cur: Partial<WorktreeEntry>): WorktreeEntry {
  return {
    worktree: cur.worktree ?? "",
    head: cur.head ?? "",
    branch: cur.branch ?? null,
    detached: cur.detached ?? false,
  };
}

export async function statusPorcelain(repoPath: string): Promise<string> {
  const { stdout } = await run(repoPath, ["status", "--porcelain"]);
  return stdout.trim();
}

export async function stashList(repoPath: string): Promise<string[]> {
  const { stdout } = await run(repoPath, ["stash", "list"]);
  return stdout
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
}

export async function unpushedCommits(repoPath: string): Promise<string[]> {
  const upstream = await run(repoPath, [
    "rev-parse",
    "--abbrev-ref",
    "--symbolic-full-name",
    "@{u}",
  ]);
  if (upstream.exitCode !== 0) return [];
  const { stdout } = await run(repoPath, ["log", "@{u}..HEAD", "--oneline"]);
  return stdout
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
}

export async function worktreeAdd(
  repoPath: string,
  opts: { path: string; branch: string; base?: string; createBranch: boolean },
): Promise<void> {
  const args = ["worktree", "add"];
  if (opts.createBranch) args.push("-b", opts.branch, opts.path, opts.base ?? "HEAD");
  else args.push(opts.path, opts.branch);
  const result = await execa("git", args, { cwd: repoPath, reject: false });
  if ((result.exitCode ?? 1) !== 0) {
    throw new Error(`git worktree add failed in ${repoPath}: ${result.stderr}`);
  }
}

export async function worktreeRemove(
  repoPath: string,
  wtPath: string,
  opts: { force?: boolean } = {},
): Promise<void> {
  const args = ["worktree", "remove"];
  if (opts.force) args.push("--force");
  args.push(wtPath);
  const result = await execa("git", args, { cwd: repoPath, reject: false });
  if ((result.exitCode ?? 1) !== 0) {
    throw new Error(`git worktree remove failed in ${repoPath}: ${result.stderr}`);
  }
}

export async function worktreePrune(repoPath: string): Promise<void> {
  await run(repoPath, ["worktree", "prune"]);
}

export async function deleteBranch(
  repoPath: string,
  branch: string,
  opts: { force?: boolean } = {},
): Promise<void> {
  const flag = opts.force ? "-D" : "-d";
  const result = await execa("git", ["branch", flag, branch], {
    cwd: repoPath,
    reject: false,
  });
  if ((result.exitCode ?? 1) !== 0) {
    throw new Error(`git branch ${flag} failed in ${repoPath}: ${result.stderr}`);
  }
}
