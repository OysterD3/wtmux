import fs from "node:fs/promises";
import { afterEach, describe, expect, it } from "vitest";
import {
  resolveRepoPath,
  validateGroupName,
  validateWorktreePattern,
} from "../../src/tui/prompts.js";
import { initRepo, makeTmpDir } from "../helpers/fixtures.js";

describe("validateGroupName", () => {
  it("accepts a unique non-empty name", () => {
    expect(validateGroupName([], "myapp")).toBeUndefined();
  });

  it("rejects empty", () => {
    expect(validateGroupName([], "")).toMatch(/empty/i);
    expect(validateGroupName([], "   ")).toMatch(/empty/i);
  });

  it("rejects names over 64 chars", () => {
    expect(validateGroupName([], "x".repeat(65))).toMatch(/too long/i);
  });

  it("rejects duplicates", () => {
    expect(validateGroupName(["myapp"], "myapp")).toMatch(/already/i);
  });

  it("allows re-using the current name when editing (existing list excludes self)", () => {
    expect(validateGroupName([], "current")).toBeUndefined();
  });
});

describe("validateWorktreePattern", () => {
  it("accepts a pattern containing {name}", () => {
    expect(validateWorktreePattern(".worktrees/{name}")).toBeUndefined();
  });

  it("accepts empty (meaning: unset override)", () => {
    expect(validateWorktreePattern("")).toBeUndefined();
  });

  it("rejects a non-empty pattern missing {name}", () => {
    expect(validateWorktreePattern(".worktrees/fixed")).toMatch(/\{name\}/);
  });
});

const tmpdirs: string[] = [];
const tmp = async () => {
  const d = await makeTmpDir();
  tmpdirs.push(d);
  return d;
};

afterEach(async () => {
  while (tmpdirs.length) await fs.rm(tmpdirs.pop()!, { recursive: true, force: true });
});

describe("resolveRepoPath", () => {
  it("accepts a real git repo path", async () => {
    const repo = await tmp();
    await initRepo(repo);
    const real = await fs.realpath(repo);
    const result = await resolveRepoPath(repo);
    expect(result.ok).toBe(true);
    expect(result.resolved).toBe(real);
  });

  it("rejects a non-git directory", async () => {
    const dir = await tmp();
    const result = await resolveRepoPath(dir);
    expect(result.ok).toBe(false);
    expect(result.error).toMatch(/not a git repository/i);
  });

  it("rejects a non-absolute path", async () => {
    const result = await resolveRepoPath("relative/path");
    expect(result.ok).toBe(false);
    expect(result.error).toMatch(/absolute/i);
  });

  it("rejects empty input", async () => {
    const result = await resolveRepoPath("   ");
    expect(result.ok).toBe(false);
    expect(result.error).toMatch(/empty/i);
  });
});
