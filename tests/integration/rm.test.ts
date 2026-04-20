import { execa } from "execa";
import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { createFlow } from "../../src/flows/create.js";
import { rmFlow } from "../../src/flows/rm.js";
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
  return {
    cfg: {
      symlinkDirectories: [],
      worktreePathPattern: ".worktrees/{name}",
      launchCommand: ["claude"],
      groups: [{ name: "g", repos: [ra, rb] }],
    },
    a: ra,
    b: rb,
  };
}

async function createPair(name: string, cfg: Config, cwd: string) {
  await createFlow({
    name,
    cwd,
    config: cfg,
    groupFlag: undefined,
    dryRun: false,
    noLaunch: true,
    extraArgs: [],
    baseOverride: undefined,
  });
}

describe("rmFlow", () => {
  it("removes clean worktrees across the group", async () => {
    const { cfg, a, b } = await setupGroup();
    await createPair("feat/clean", cfg, a);

    const result = await rmFlow({
      name: "feat/clean",
      cwd: a,
      config: cfg,
      groupFlag: undefined,
      dryRun: false,
      force: false,
    });

    expect(result.removed.length).toBe(2);
    await expect(fs.stat(path.join(a, ".worktrees/feat-clean"))).rejects.toThrow();
    await expect(fs.stat(path.join(b, ".worktrees/feat-clean"))).rejects.toThrow();
  });

  it("skips a dirty worktree and reports it", async () => {
    const { cfg, a, b } = await setupGroup();
    await createPair("feat/dirty", cfg, a);
    const wtA = path.join(a, ".worktrees/feat-dirty");
    await fs.writeFile(path.join(wtA, "README.md"), "dirty\n");

    const result = await rmFlow({
      name: "feat/dirty",
      cwd: a,
      config: cfg,
      groupFlag: undefined,
      dryRun: false,
      force: false,
    });

    expect(result.skipped.map((s) => s.repo)).toContain(a);
    expect(result.removed.map((r) => r.repo)).toContain(b);
  });

  it("skips a worktree with stashes and reports it", async () => {
    const { cfg, a } = await setupGroup();
    await createPair("feat/stashed", cfg, a);
    const wtA = path.join(a, ".worktrees/feat-stashed");
    await fs.writeFile(path.join(wtA, "README.md"), "stashed\n");
    await execa("git", ["-C", wtA, "stash", "push", "-m", "wip"]);

    const result = await rmFlow({
      name: "feat/stashed",
      cwd: a,
      config: cfg,
      groupFlag: undefined,
      dryRun: false,
      force: false,
    });

    expect(result.skipped.map((s) => s.repo)).toContain(a);
  });

  it("--force removes dirty worktrees", async () => {
    const { cfg, a } = await setupGroup();
    await createPair("feat/forced", cfg, a);
    const wtA = path.join(a, ".worktrees/feat-forced");
    await fs.writeFile(path.join(wtA, "README.md"), "dirty\n");

    const result = await rmFlow({
      name: "feat/forced",
      cwd: a,
      config: cfg,
      groupFlag: undefined,
      dryRun: false,
      force: true,
    });

    expect(result.removed.length).toBe(2);
    await expect(fs.stat(wtA)).rejects.toThrow();
  });

  it("dry-run skips all mutations", async () => {
    const { cfg, a } = await setupGroup();
    await createPair("feat/preview", cfg, a);
    await rmFlow({
      name: "feat/preview",
      cwd: a,
      config: cfg,
      groupFlag: undefined,
      dryRun: true,
      force: false,
    });
    expect((await fs.stat(path.join(a, ".worktrees/feat-preview"))).isDirectory()).toBe(true);
  });
});
