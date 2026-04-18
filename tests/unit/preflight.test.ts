import { execa } from "execa";
import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { preflightCreate } from "../../src/preflight.js";
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

describe("preflightCreate", () => {
  it("rejects invalid branch names via check-ref-format", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const result = await preflightCreate({
      name: "has space",
      baseBranch: "main",
      repos: [{ path: repo, wtPath: path.join(repo, ".worktrees/has space") }],
    });
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toMatch(/invalid.*name/i);
  });

  it("rejects when a repo is not a worktree root", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const wtParent = await tmp();
    const wt = path.join(wtParent, "wt");
    await execa("git", ["-C", repo, "worktree", "add", "-b", "feat/x", wt]);
    const result = await preflightCreate({
      name: "feat/y",
      baseBranch: "main",
      repos: [{ path: wt, wtPath: path.join(wt, ".worktrees/feat/y") }],
    });
    expect(result.ok).toBe(false);
    expect(result.errors.join("\n")).toMatch(/worktree root/i);
  });

  it("rejects when target worktree path already exists", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const wt = path.join(repo, ".worktrees", "feat", "x");
    await fs.mkdir(wt, { recursive: true });
    const result = await preflightCreate({
      name: "feat/x",
      baseBranch: "main",
      repos: [{ path: repo, wtPath: wt }],
    });
    expect(result.ok).toBe(false);
    expect(result.errors.join("\n")).toMatch(/already exists/i);
  });

  it("rejects when branch exists and is checked out somewhere", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const wtParent = await tmp();
    const wt = path.join(wtParent, "wt-attached");
    await execa("git", ["-C", repo, "worktree", "add", "-b", "feat/taken", wt]);

    const result = await preflightCreate({
      name: "feat/taken",
      baseBranch: "main",
      repos: [{ path: repo, wtPath: path.join(repo, ".worktrees/feat/taken") }],
    });
    expect(result.ok).toBe(false);
    expect(result.errors.join("\n")).toMatch(/checked out/i);
  });

  it("rejects when branch does not exist AND base branch does not exist", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const result = await preflightCreate({
      name: "feat/new",
      baseBranch: "nope",
      repos: [{ path: repo, wtPath: path.join(repo, ".worktrees/feat/new") }],
    });
    expect(result.ok).toBe(false);
    expect(result.errors.join("\n")).toMatch(/base branch.*nope/i);
  });

  it("passes when everything is fine", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const result = await preflightCreate({
      name: "feat/new",
      baseBranch: "main",
      repos: [{ path: repo, wtPath: path.join(repo, ".worktrees/feat/new") }],
    });
    expect(result.ok).toBe(true);
    expect(result.errors).toEqual([]);
  });
});
