import { execa } from "execa";
import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { createFlow } from "../../src/flows/create.js";
import { initRepo, makeTmpDir } from "../helpers/fixtures.js";
import type { Config } from "../../src/config/schema.js";

const tmpdirs: string[] = [];
const tmp = async () => {
  const d = await makeTmpDir();
  tmpdirs.push(d);
  return d;
};

afterEach(async () => {
  while (tmpdirs.length) await fs.rm(tmpdirs.pop()!, { recursive: true, force: true });
});

async function setupGroup(): Promise<{ cfg: Config; a: string; b: string }> {
  const a = await tmp();
  const b = await tmp();
  await initRepo(a, "main");
  await initRepo(b, "main");
  const ra = await fs.realpath(a);
  const rb = await fs.realpath(b);
  const cfg: Config = {
    symlinkDirectories: ["node_modules", ".env"],
    worktreePathPattern: ".worktrees/{name}",
    launchCommand: ["claude"],
    groups: [{ name: "g", repos: [ra, rb] }],
  };
  await fs.mkdir(path.join(ra, "node_modules"));
  await fs.writeFile(path.join(ra, ".env"), "SECRET=1");
  await fs.mkdir(path.join(rb, "node_modules"));
  return { cfg, a: ra, b: rb };
}

describe("createFlow", () => {
  it("creates coordinated worktrees with matching branches and replicated symlinks", async () => {
    const { cfg, a, b } = await setupGroup();
    const result = await createFlow({
      name: "feat/x",
      cwd: a,
      config: cfg,
      groupFlag: undefined,
      dryRun: false,
      noLaunch: true,
      extraArgs: [],
    });

    expect(result.kind).toBe("group");
    if (result.kind !== "group") return;

    const wtA = path.join(a, ".worktrees/feat/x");
    const wtB = path.join(b, ".worktrees/feat/x");
    expect((await fs.stat(wtA)).isDirectory()).toBe(true);
    expect((await fs.stat(wtB)).isDirectory()).toBe(true);

    const branchA = (await execa("git", ["-C", wtA, "symbolic-ref", "--short", "HEAD"])).stdout.trim();
    expect(branchA).toBe("feat/x");

    expect((await fs.lstat(path.join(wtA, "node_modules"))).isSymbolicLink()).toBe(true);
    expect((await fs.lstat(path.join(wtA, ".env"))).isSymbolicLink()).toBe(true);
    expect((await fs.lstat(path.join(wtB, "node_modules"))).isSymbolicLink()).toBe(true);
  });

  it("dry-run makes no changes", async () => {
    const { cfg, a, b } = await setupGroup();
    await createFlow({
      name: "feat/dry",
      cwd: a,
      config: cfg,
      groupFlag: undefined,
      dryRun: true,
      noLaunch: true,
      extraArgs: [],
    });
    await expect(fs.stat(path.join(a, ".worktrees/feat/dry"))).rejects.toThrow();
    await expect(fs.stat(path.join(b, ".worktrees/feat/dry"))).rejects.toThrow();
  });

  it("rejects detached HEAD in primary", async () => {
    const { cfg, a } = await setupGroup();
    const sha = (await execa("git", ["-C", a, "rev-parse", "HEAD"])).stdout.trim();
    await execa("git", ["-C", a, "checkout", sha]);
    await expect(
      createFlow({
        name: "feat/x",
        cwd: a,
        config: cfg,
        groupFlag: undefined,
        dryRun: false,
        noLaunch: true,
        extraArgs: [],
      }),
    ).rejects.toThrow(/detached/i);
  });

  it("rolls back already-created worktrees when a later repo fails", async () => {
    const { cfg, a, b } = await setupGroup();
    const collision = path.join(b, ".worktrees/feat/collide");
    await fs.mkdir(collision, { recursive: true });
    await fs.writeFile(path.join(collision, "marker"), "x");

    await expect(
      createFlow({
        name: "feat/collide",
        cwd: a,
        config: cfg,
        groupFlag: undefined,
        dryRun: false,
        noLaunch: true,
        extraArgs: [],
      }),
    ).rejects.toThrow();

    await expect(fs.stat(path.join(a, ".worktrees/feat/collide"))).rejects.toThrow();
  });
});
