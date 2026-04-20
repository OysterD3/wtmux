import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { detectSiblings } from "../../src/tui/siblings.js";
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

describe("detectSiblings", () => {
  it("finds sibling git repos in cwd's parent", async () => {
    const parent = await tmp();
    const a = path.join(parent, "repo-a");
    const b = path.join(parent, "repo-b");
    const c = path.join(parent, "repo-c");
    await initRepo(a);
    await initRepo(b);
    await initRepo(c);

    const siblings = await detectSiblings(a);
    expect(siblings.sort()).toEqual([b, c].sort());
  });

  it("excludes cwd's own repo from the result", async () => {
    const parent = await tmp();
    const a = path.join(parent, "repo-a");
    const b = path.join(parent, "repo-b");
    await initRepo(a);
    await initRepo(b);

    const siblings = await detectSiblings(a);
    expect(siblings).not.toContain(a);
    expect(siblings).toEqual([b]);
  });

  it("ignores non-git directories", async () => {
    const parent = await tmp();
    const a = path.join(parent, "repo-a");
    const notRepo = path.join(parent, "plain-dir");
    await initRepo(a);
    await fs.mkdir(notRepo);

    const siblings = await detectSiblings(a);
    expect(siblings).toEqual([]);
  });

  it("returns empty array when parent has no siblings", async () => {
    const parent = await tmp();
    const a = path.join(parent, "repo-a");
    await initRepo(a);

    const siblings = await detectSiblings(a);
    expect(siblings).toEqual([]);
  });

  it("scans children instead of parent when cwd is not a git repo", async () => {
    const parent = await tmp();
    const container = path.join(parent, "container");
    const parentSibling = path.join(parent, "parent-sibling");
    const childA = path.join(container, "child-a");
    const childB = path.join(container, "child-b");
    const childPlain = path.join(container, "plain");
    await initRepo(parentSibling);
    await initRepo(childA);
    await initRepo(childB);
    await fs.mkdir(childPlain);

    const siblings = await detectSiblings(container);
    expect(siblings.sort()).toEqual([childA, childB].sort());
    expect(siblings).not.toContain(parentSibling);
  });

  it("returns empty array when cwd is neither a repo nor contains repos", async () => {
    const parent = await tmp();
    const notRepo = path.join(parent, "not-repo");
    await fs.mkdir(notRepo);
    await fs.mkdir(path.join(notRepo, "also-plain"));

    const siblings = await detectSiblings(notRepo);
    expect(siblings).toEqual([]);
  });
});
