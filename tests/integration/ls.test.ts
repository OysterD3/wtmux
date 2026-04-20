import fs from "node:fs/promises";
import { afterEach, describe, expect, it } from "vitest";
import { createFlow } from "../../src/flows/create.js";
import { lsFlow } from "../../src/flows/ls.js";
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

describe("lsFlow", () => {
  it("returns a row per worktree name with per-repo state", async () => {
    const { cfg, a } = await setupGroup();
    await createFlow({
      name: "feat/x",
      cwd: a,
      config: cfg,
      groupFlag: undefined,
      dryRun: false,
      noLaunch: true,
      extraArgs: [],
      baseOverride: undefined,
    });

    const rows = await lsFlow({ cwd: a, config: cfg, groupFlag: undefined });
    const featX = rows.find((r) => r.name === "feat/x");
    expect(featX).toBeDefined();
    expect(featX!.perRepo.length).toBe(2);
    expect(featX!.perRepo.every((p) => p.present && p.state === "clean")).toBe(true);
  });

  it("marks repos where the worktree is absent", async () => {
    const { cfg, a, b } = await setupGroup();
    const { execa } = await import("execa");
    await execa("git", ["-C", a, "worktree", "add", "-b", "feat/orphan", `${a}/.worktrees/feat/orphan`]);

    const rows = await lsFlow({ cwd: a, config: cfg, groupFlag: undefined });
    const orphan = rows.find((r) => r.name === "feat/orphan");
    expect(orphan).toBeDefined();
    const statusB = orphan!.perRepo.find((p) => p.repo === b);
    expect(statusB!.present).toBe(false);
  });
});
