import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { replicateSymlinks, type SymlinkResult } from "../../src/symlinks.js";
import { makeTmpDir, writeFile } from "../helpers/fixtures.js";

const tmpdirs: string[] = [];
const tmp = async () => {
  const d = await makeTmpDir();
  tmpdirs.push(d);
  return d;
};

afterEach(async () => {
  while (tmpdirs.length) await fs.rm(tmpdirs.pop()!, { recursive: true, force: true });
});

describe("replicateSymlinks", () => {
  it("creates a dir symlink when source is a directory", async () => {
    const repo = await tmp();
    const wt = await tmp();
    await fs.mkdir(path.join(repo, "node_modules"));
    const results = await replicateSymlinks({ repo, wt, items: ["node_modules"] });
    const link = path.join(wt, "node_modules");
    const stat = await fs.lstat(link);
    expect(stat.isSymbolicLink()).toBe(true);
    expect(await fs.readlink(link)).toBe(path.join(repo, "node_modules"));
    expect(results).toContainEqual<SymlinkResult>({
      item: "node_modules",
      action: "linked",
    });
  });

  it("creates a file symlink when source is a file", async () => {
    const repo = await tmp();
    const wt = await tmp();
    await writeFile(path.join(repo, ".env"), "SECRET=1");
    await replicateSymlinks({ repo, wt, items: [".env"] });
    expect((await fs.lstat(path.join(wt, ".env"))).isSymbolicLink()).toBe(true);
  });

  it("skips when source does not exist", async () => {
    const repo = await tmp();
    const wt = await tmp();
    const results = await replicateSymlinks({ repo, wt, items: [".env"] });
    expect(results).toEqual([{ item: ".env", action: "skipped-no-source" }]);
    await expect(fs.lstat(path.join(wt, ".env"))).rejects.toThrow();
  });

  it("skips when target already exists", async () => {
    const repo = await tmp();
    const wt = await tmp();
    await fs.mkdir(path.join(repo, "node_modules"));
    await fs.mkdir(path.join(wt, "node_modules"));
    const results = await replicateSymlinks({ repo, wt, items: ["node_modules"] });
    expect(results).toEqual([{ item: "node_modules", action: "skipped-target-exists" }]);
  });

  it("returns a placed list so callers can roll back on failure", async () => {
    const repo = await tmp();
    const wt = await tmp();
    await fs.mkdir(path.join(repo, "node_modules"));
    await writeFile(path.join(repo, ".env"), "x");
    const results = await replicateSymlinks({ repo, wt, items: ["node_modules", ".env"] });
    const placed = results.filter((r) => r.action === "linked").map((r) => r.item);
    expect(placed).toEqual(["node_modules", ".env"]);
  });
});
