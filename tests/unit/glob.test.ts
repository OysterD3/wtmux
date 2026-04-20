import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { expandSymlinkItems } from "../../src/glob.js";
import { makeTmpDir } from "../helpers/fixtures.js";

const tmpdirs: string[] = [];
const tmp = async () => {
  const d = await makeTmpDir();
  tmpdirs.push(d);
  return d;
};

afterEach(async () => {
  while (tmpdirs.length) await fs.rm(tmpdirs.pop()!, { recursive: true, force: true });
});

describe("expandSymlinkItems", () => {
  it("passes literal paths through unchanged", async () => {
    const repo = await tmp();
    await fs.mkdir(path.join(repo, "node_modules"));
    const result = await expandSymlinkItems(repo, ["node_modules"]);
    expect(result).toEqual(["node_modules"]);
  });

  it("expands a glob and includes dotfiles by default", async () => {
    const repo = await tmp();
    await fs.writeFile(path.join(repo, ".env"), "A=1");
    await fs.writeFile(path.join(repo, ".env.local"), "B=2");
    await fs.writeFile(path.join(repo, ".env.development"), "C=3");
    const result = await expandSymlinkItems(repo, [".env*"]);
    expect(result.sort()).toEqual([".env", ".env.development", ".env.local"].sort());
  });

  it("returns empty when a glob matches nothing", async () => {
    const repo = await tmp();
    const result = await expandSymlinkItems(repo, [".env*"]);
    expect(result).toEqual([]);
  });

  it("dedupes literal + overlapping glob", async () => {
    const repo = await tmp();
    await fs.writeFile(path.join(repo, ".env"), "x");
    const result = await expandSymlinkItems(repo, [".env", ".env*"]);
    expect(result).toEqual([".env"]);
  });

  it("mixes literal and glob items", async () => {
    const repo = await tmp();
    await fs.mkdir(path.join(repo, "node_modules"));
    await fs.writeFile(path.join(repo, ".env"), "x");
    await fs.writeFile(path.join(repo, ".env.local"), "y");
    const result = await expandSymlinkItems(repo, ["node_modules", ".env*"]);
    expect(result.sort()).toEqual(["node_modules", ".env", ".env.local"].sort());
  });

  it("supports ** recursion", async () => {
    const repo = await tmp();
    await fs.mkdir(path.join(repo, "config", "sub"), { recursive: true });
    await fs.writeFile(path.join(repo, "config", "a.json"), "{}");
    await fs.writeFile(path.join(repo, "config", "sub", "b.json"), "{}");
    const result = await expandSymlinkItems(repo, ["config/**/*.json"]);
    expect(result.sort()).toEqual(["config/a.json", "config/sub/b.json"].sort());
  });
});
