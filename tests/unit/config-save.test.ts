import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { saveConfig } from "../../src/config/save.js";
import type { Config } from "../../src/config/schema.js";
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

function valid(): Config {
  return {
    symlinkDirectories: ["node_modules", ".env"],
    worktreePathPattern: ".worktrees/{name}",
    launchCommand: ["claude"],
    groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
  };
}

describe("saveConfig", () => {
  it("writes pretty-printed JSON with trailing newline", async () => {
    const dir = await tmp();
    const target = path.join(dir, "config.json");
    await saveConfig(target, valid());

    const content = await fs.readFile(target, "utf8");
    expect(content.endsWith("\n")).toBe(true);
    const parsed = JSON.parse(content);
    expect(parsed.groups[0].name).toBe("g");
    expect(content).toContain("  ");
  });

  it("creates parent directories if they don't exist", async () => {
    const root = await tmp();
    const target = path.join(root, "deep", "nested", "config.json");
    await saveConfig(target, valid());
    expect((await fs.stat(target)).isFile()).toBe(true);
  });

  it("does not leave a .tmp file after success", async () => {
    const dir = await tmp();
    const target = path.join(dir, "config.json");
    await saveConfig(target, valid());
    await expect(fs.stat(target + ".tmp")).rejects.toThrow();
  });

  it("rejects when config fails Zod validation", async () => {
    const dir = await tmp();
    const target = path.join(dir, "config.json");
    const invalid = { ...valid(), groups: [{ name: "g", repos: ["/only/one"] }] } as Config;
    await expect(saveConfig(target, invalid)).rejects.toThrow();
    await expect(fs.stat(target)).rejects.toThrow();
  });

  it("overwrites existing file cleanly", async () => {
    const dir = await tmp();
    const target = path.join(dir, "config.json");
    await saveConfig(target, valid());
    await saveConfig(target, valid());
    expect((await fs.stat(target)).isFile()).toBe(true);
    const parsed = JSON.parse(await fs.readFile(target, "utf8"));
    expect(parsed.groups[0].name).toBe("g");
  });
});
