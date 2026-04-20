import { execa } from "execa";
import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import {
  checkRefFormat,
  deleteBranch,
  getCurrentBranch,
  getToplevel,
  isWorktreeRoot,
  listWorktrees,
  stashList,
  statusPorcelain,
  unpushedCommits,
  worktreeAdd,
} from "../../src/git.js";
import { initRepo, makeTmpDir } from "../helpers/fixtures.js";

const tmpdirs: string[] = [];
const tmp = async () => {
  const d = await makeTmpDir();
  tmpdirs.push(d);
  return d;
};

afterEach(async () => {
  while (tmpdirs.length) await fs.rm(tmpdirs.pop()!, { recursive: true, force: true });
});

describe("git helpers", () => {
  it("getToplevel returns the repo root", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const sub = path.join(repo, "a", "b");
    await fs.mkdir(sub, { recursive: true });
    const top = await getToplevel(sub);
    expect(top).toBe(await fs.realpath(repo));
  });

  it("getToplevel returns null outside a repo", async () => {
    const dir = await tmp();
    expect(await getToplevel(dir)).toBeNull();
  });

  it("isWorktreeRoot is true for the main worktree and false for a linked worktree", async () => {
    const repo = await tmp();
    await initRepo(repo);
    expect(await isWorktreeRoot(repo)).toBe(true);

    const wtBase = await tmp();
    const wt = path.join(wtBase, "wt");
    await execa("git", ["-C", repo, "worktree", "add", "-b", "feat/x", wt]);
    expect(await isWorktreeRoot(wt)).toBe(false);
  });

  it("getCurrentBranch returns the branch name", async () => {
    const repo = await tmp();
    await initRepo(repo, "main");
    expect(await getCurrentBranch(repo)).toBe("main");
  });

  it("getCurrentBranch returns null on detached HEAD", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const sha = (await execa("git", ["-C", repo, "rev-parse", "HEAD"])).stdout.trim();
    await execa("git", ["-C", repo, "checkout", sha]);
    expect(await getCurrentBranch(repo)).toBeNull();
  });

  it("checkRefFormat accepts valid names and rejects bad ones", async () => {
    expect(await checkRefFormat("feat/foo")).toBe(true);
    expect(await checkRefFormat("feat..bar")).toBe(false);
    expect(await checkRefFormat("has space")).toBe(false);
    expect(await checkRefFormat("-leadingdash")).toBe(false);
  });

  it("listWorktrees returns the main worktree plus any linked", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const wtBase = await tmp();
    const wt = path.join(wtBase, "wt");
    await execa("git", ["-C", repo, "worktree", "add", "-b", "feat/y", wt]);

    const list = await listWorktrees(repo);
    const branches = list.map((w) => w.branch);
    expect(branches).toContain("main");
    expect(branches).toContain("feat/y");
  });

  it("statusPorcelain returns empty for clean worktrees", async () => {
    const repo = await tmp();
    await initRepo(repo);
    expect(await statusPorcelain(repo)).toBe("");
  });

  it("statusPorcelain returns non-empty for dirty worktrees", async () => {
    const repo = await tmp();
    await initRepo(repo);
    await fs.writeFile(path.join(repo, "README.md"), "dirty\n");
    expect(await statusPorcelain(repo)).not.toBe("");
  });

  it("stashList is empty by default and non-empty after stashing", async () => {
    const repo = await tmp();
    await initRepo(repo);
    expect(await stashList(repo)).toEqual([]);
    await fs.writeFile(path.join(repo, "README.md"), "stashed\n");
    await execa("git", ["-C", repo, "stash", "push", "-m", "wip"]);
    expect((await stashList(repo)).length).toBe(1);
  });

  it("unpushedCommits returns [] when there is no upstream", async () => {
    const repo = await tmp();
    await initRepo(repo);
    expect(await unpushedCommits(repo)).toEqual([]);
  });

  it("worktreeAdd creates a new worktree with a new branch", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const wt = path.join(await tmp(), "wt-new");
    await worktreeAdd(repo, { path: wt, branch: "feat/new", base: "main", createBranch: true });
    expect((await fs.stat(wt)).isDirectory()).toBe(true);
    const branch = (await execa("git", ["-C", wt, "symbolic-ref", "--short", "HEAD"])).stdout.trim();
    expect(branch).toBe("feat/new");
  });

  it("deleteBranch removes a fully-merged branch with -d", async () => {
    const repo = await tmp();
    await initRepo(repo);
    await execa("git", ["-C", repo, "branch", "feat/x"]);
    await deleteBranch(repo, "feat/x");
    const { stdout } = await execa("git", ["-C", repo, "branch", "--list", "feat/x"]);
    expect(stdout.trim()).toBe("");
  });

  it("deleteBranch with force removes an unmerged branch", async () => {
    const repo = await tmp();
    await initRepo(repo);
    await execa("git", ["-C", repo, "checkout", "-b", "feat/x"]);
    await fs.writeFile(path.join(repo, "new.md"), "x\n");
    await execa("git", ["-C", repo, "add", "new.md"]);
    await execa("git", ["-C", repo, "commit", "-m", "unmerged"]);
    await execa("git", ["-C", repo, "checkout", "main"]);
    await expect(deleteBranch(repo, "feat/x")).rejects.toThrow();
    await deleteBranch(repo, "feat/x", { force: true });
    const { stdout } = await execa("git", ["-C", repo, "branch", "--list", "feat/x"]);
    expect(stdout.trim()).toBe("");
  });
});
