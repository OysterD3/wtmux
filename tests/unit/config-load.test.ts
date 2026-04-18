import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { loadConfig } from "../../src/config/load.js";
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

describe("loadConfig", () => {
  it("returns null when no config is discovered", async () => {
    const cwd = await tmp();
    const home = await tmp();
    const result = await loadConfig({ cwd, home, env: {} });
    expect(result).toBeNull();
  });

  it("expands tildes in repo paths before validation", async () => {
    const home = await tmp();
    const cwd = await tmp();
    const repoA = path.join(home, "repo-a");
    const repoB = path.join(home, "repo-b");
    await fs.mkdir(repoA, { recursive: true });
    await fs.mkdir(repoB, { recursive: true });

    const cfgPath = path.join(cwd, ".wtmux.json");
    await writeFile(cfgPath, JSON.stringify({
      groups: [{ name: "g", repos: ["~/repo-a", "~/repo-b"] }],
    }));

    // The loader expands ~ against the provided home, not os.homedir().
    // Pass home via a parameter so tests are deterministic.
    const result = await loadConfig({ cwd, home, env: {} });
    expect(result).not.toBeNull();
    expect(result!.config.groups[0]!.repos).toEqual([repoA, repoB]);
    expect(result!.source).toBe("walk");
    expect(result!.path).toBe(cfgPath);
  });

  it("throws on malformed JSON", async () => {
    const cwd = await tmp();
    const home = await tmp();
    await writeFile(path.join(cwd, ".wtmux.json"), "{ not json");
    await expect(loadConfig({ cwd, home, env: {} })).rejects.toThrow(/JSON/);
  });

  it("throws with repo+path context on schema violation", async () => {
    const cwd = await tmp();
    const home = await tmp();
    await writeFile(
      path.join(cwd, ".wtmux.json"),
      JSON.stringify({ groups: [{ name: "g", repos: ["/only/one"] }] }),
    );
    await expect(loadConfig({ cwd, home, env: {} })).rejects.toThrow(/groups/);
  });
});
