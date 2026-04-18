import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { discoverConfigPath } from "../../src/config/discovery.js";
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

describe("discoverConfigPath", () => {
  it("returns the explicit path when one is passed", async () => {
    const dir = await tmp();
    const cfg = path.join(dir, "explicit.json");
    await writeFile(cfg, "{}");
    const result = await discoverConfigPath({ explicit: cfg, env: {}, cwd: dir, home: dir });
    expect(result).toEqual({ path: cfg, source: "flag" });
  });

  it("returns the env var path when set and no flag", async () => {
    const dir = await tmp();
    const cfg = path.join(dir, "from-env.json");
    await writeFile(cfg, "{}");
    const result = await discoverConfigPath({
      explicit: undefined,
      env: { WTMUX_CONFIG: cfg },
      cwd: dir,
      home: dir,
    });
    expect(result).toEqual({ path: cfg, source: "env" });
  });

  it("walks upward from cwd to find .wtmux.json", async () => {
    const root = await tmp();
    const child = path.join(root, "a", "b", "c");
    await fs.mkdir(child, { recursive: true });
    const cfg = path.join(root, ".wtmux.json");
    await writeFile(cfg, "{}");
    const result = await discoverConfigPath({ explicit: undefined, env: {}, cwd: child, home: root });
    expect(result).toEqual({ path: cfg, source: "walk" });
  });

  it("falls back to XDG user config", async () => {
    const home = await tmp();
    const cwd = await tmp();
    const xdg = path.join(home, ".config", "wtmux", "config.json");
    await writeFile(xdg, "{}");
    const result = await discoverConfigPath({ explicit: undefined, env: {}, cwd, home });
    expect(result).toEqual({ path: xdg, source: "xdg" });
  });

  it("returns null when nothing is found", async () => {
    const home = await tmp();
    const cwd = await tmp();
    const result = await discoverConfigPath({ explicit: undefined, env: {}, cwd, home });
    expect(result).toBeNull();
  });

  it("throws if the explicit path does not exist", async () => {
    const dir = await tmp();
    await expect(
      discoverConfigPath({ explicit: path.join(dir, "missing.json"), env: {}, cwd: dir, home: dir }),
    ).rejects.toThrow(/not found/i);
  });
});
