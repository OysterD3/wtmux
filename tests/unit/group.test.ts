import fs from "node:fs/promises";
import { afterEach, describe, expect, it } from "vitest";
import { resolveGroup, determinePrimary } from "../../src/group.js";
import type { Config } from "../../src/config/schema.js";
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

function mkCfg(groups: Config["groups"]): Config {
  return {
    symlinkDirectories: ["node_modules", ".env"],
    worktreePathPattern: ".worktrees/{name}",
    launchCommand: ["claude"],
    groups,
  };
}

describe("resolveGroup", () => {
  it("returns the group whose repos contain the cwd's toplevel", async () => {
    const a = await tmp();
    const b = await tmp();
    await initRepo(a);
    await initRepo(b);
    const cfg = mkCfg([{ name: "g", repos: [await fs.realpath(a), await fs.realpath(b)] }]);

    const result = await resolveGroup({ cwd: a, config: cfg, groupFlag: undefined });
    expect(result.kind).toBe("group");
    if (result.kind === "group") expect(result.group.name).toBe("g");
  });

  it("returns single-repo mode when cwd is in a repo not in any group", async () => {
    const stranger = await tmp();
    await initRepo(stranger);
    const cfg = mkCfg([]);
    const result = await resolveGroup({ cwd: stranger, config: cfg, groupFlag: undefined });
    expect(result.kind).toBe("single");
  });

  it("returns outside-mode when cwd is not in any repo", async () => {
    const dir = await tmp();
    const cfg = mkCfg([]);
    const result = await resolveGroup({ cwd: dir, config: cfg, groupFlag: undefined });
    expect(result.kind).toBe("outside");
  });

  it("returns the named group when --group is provided", async () => {
    const a = await tmp();
    const b = await tmp();
    await initRepo(a);
    await initRepo(b);
    const cfg = mkCfg([{ name: "g", repos: [await fs.realpath(a), await fs.realpath(b)] }]);
    const other = await tmp();
    const result = await resolveGroup({ cwd: other, config: cfg, groupFlag: "g" });
    expect(result.kind).toBe("group");
  });

  it("throws when --group names a group that doesn't exist", async () => {
    const cfg = mkCfg([]);
    const cwd = await tmp();
    await expect(resolveGroup({ cwd, config: cfg, groupFlag: "nope" })).rejects.toThrow(/nope/);
  });

  it("throws when cwd matches more than one group", async () => {
    const a = await tmp();
    await initRepo(a);
    const real = await fs.realpath(a);
    const fake = await tmp();
    await initRepo(fake);
    const cfg = mkCfg([
      { name: "g1", repos: [real, await fs.realpath(fake)] },
      { name: "g2", repos: [real, await fs.realpath(fake)] },
    ]);
    await expect(resolveGroup({ cwd: a, config: cfg, groupFlag: undefined })).rejects.toThrow(
      /more than one/i,
    );
  });
});

describe("determinePrimary", () => {
  it("returns the repo matching cwd's toplevel when it's in the group", async () => {
    const a = await tmp();
    const b = await tmp();
    await initRepo(a);
    await initRepo(b);
    const ra = await fs.realpath(a);
    const rb = await fs.realpath(b);
    const group = { name: "g", repos: [ra, rb] };
    expect(await determinePrimary(b, group)).toBe(rb);
  });

  it("falls back to the first repo in group when cwd is elsewhere", async () => {
    const a = await tmp();
    const b = await tmp();
    await initRepo(a);
    await initRepo(b);
    const ra = await fs.realpath(a);
    const rb = await fs.realpath(b);
    const group = { name: "g", repos: [ra, rb] };
    const stranger = await tmp();
    expect(await determinePrimary(stranger, group)).toBe(ra);
  });
});
